package server

import (
	"fmt"
	"encoding/json"
	"net/http"
	"io"
	"log"
	"database/sql"
	"github.com/YahiaE/listening-engine/internal/models"
	"github.com/YahiaE/listening-engine/internal/auth"
	"github.com/google/uuid"
	"time"
	"github.com/hashicorp/golang-lru/v2"
	"context"

)

var databasePool *sql.DB
var auth_cache *lru.Cache[string, string]


func handler(w http.ResponseWriter, r *http.Request){
	// var event models.ListeningEvent
	var user models.User
	var token models.AuthToken
	isNewUser := false
	// var song models.Song

	
	userID := r.Header.Get("User-ID")
	userToken := r.Header.Get("Auth-Token")
	// isRegistered := false

	if len(userToken) == 0 {
		log.Println("Received empty token. Generating ID and token for user")
		isNewUser = true
		newUserUUID := uuid.New().String()
		newUserToken := auth.GenerateToken()
		
		log.Println("Adding user to cache...")
		auth_cache.Add(newUserUUID, auth.EncryptToken(newUserToken))

		user.ID = newUserUUID

		token.Token = newUserToken
		token.UserID = user.ID

		storeUserAndToken(newUserUUID,auth.EncryptToken(newUserToken))
		tokenJson, err := json.Marshal(token)

		if err != nil {
			http.Error(w, "Internal server error: Unable to format credentials", http.StatusInternalServerError)
			return
		}

		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		w.Write(tokenJson)		

		return
	}



	if !isNewUser {
		log.Println("Found credentials. Cross-checking with database...")
		
		cachedUserToken, ok := auth_cache.Get(userID)
		matchCache := false

		if !ok {
			log.Println("User not found in cache... checking database for auth")
		} else {
			matchCache = (auth.EncryptToken(userToken) == cachedUserToken)
		}


		
		// check if token received is associated with the user
		// if token exist + connected to user => add event
		if matchCache || checkUser(userID, auth.EncryptToken(userToken)) {
			var songRead models.Song 

			bodyBytes, err := io.ReadAll(r.Body)

			if err != nil {
				http.Error(w, "Failed to read request body", http.StatusBadRequest)
				return
			} 

			defer r.Body.Close()

			err = json.Unmarshal(bodyBytes, &songRead)

			if err != nil {
				http.Error(w, "Internal server error: Unable to map data into usable object", http.StatusInternalServerError)
				return
			}

			
			if songRead.Title == "" {
				log.Println("Logged empty song")
				return
			}
			songID, err := storeSong(songRead)

			if err != nil {
				http.Error(w, "Internal server error: Unable to store song data", http.StatusInternalServerError)
				return
			}

			err = storeAndSessionizeEvent(userID, songID)
			
			if err != nil {
				http.Error(w, "Internal server error: Unable to store event data", http.StatusInternalServerError)
				return
			}

		} else {
			log.Println("Expired token! Regenerating...")
		}
	}

}

func storeUserAndToken(userID string, token string) error {
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	
	tx, err := databasePool.BeginTx(ctx, nil)

	if err != nil {
		return err
	}

	defer tx.Rollback()

	_, err = tx.ExecContext(ctx, "INSERT INTO users (id) VALUES ($1)", userID)
	if err != nil {
		log.Println(err)
    	return err
	}
	log.Println("inserted user id")

	_, err = tx.ExecContext(ctx, "INSERT INTO auth_token (token, user_id) VALUES ($1, $2)", token, userID)
	if err != nil {
		log.Println(err)
    	return err
	}

	log.Println("inserted token")

	log.Printf("Registered new user!")

	return tx.Commit()
}


func storeSong(song models.Song) (int64, error){
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	query := `
	INSERT INTO song (title, artist, album)
	VALUES ($1, $2, $3)
	ON CONFLICT (LOWER(TRIM(title)), LOWER(TRIM(artist)), LOWER(TRIM(album))) 
	DO UPDATE SET title = EXCLUDED.title
	RETURNING id;
	`
	// updated title to the same title to just trigger return from query 

	var songID int64
    err := databasePool.QueryRowContext(ctx, query, song.Title, song.Artist, song.Album).Scan(&songID)
    if err != nil {
        log.Printf("Failed to store/retrieve song: %v", err)
        return 0, err
    }

	return songID, nil
}

func storeAndSessionizeEvent(user_id string, song_id int64) error {
    ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
    defer cancel()

    tx, err := databasePool.BeginTx(ctx, nil)
    if err != nil {
        return err
    }
    defer tx.Rollback()

    query := `SELECT event_id, s_id, time_created FROM create_event_and_session($1, $2);`
    
    row := tx.QueryRowContext(ctx, query, user_id, song_id)

    var eventID int64
    var sessionID int64
    var createdAt time.Time
    err = row.Scan(&eventID, &sessionID, &createdAt)
    if err != nil {
        log.Println(err)
        return err
    }

    log.Printf("Stored event of ID %v into Session %v. Time made: %v", eventID, sessionID, createdAt)

    return tx.Commit()
}

func EstablishStorageFunction(db *sql.DB) error {	
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	tx, err := db.BeginTx(ctx, nil)

	if err != nil {
		return err
	}

	defer tx.Rollback()

	query := `
	CREATE OR REPLACE FUNCTION create_event_and_session(p_user_id UUID, p_song_id BIGINT)
	RETURNS TABLE (event_id BIGINT, s_id BIGINT, time_created TIMESTAMPTZ) AS $$
	DECLARE
    	sess_id BIGINT;
    	new_event_id BIGINT;
    	created_at TIMESTAMPTZ := NOW();
    	most_recent TIMESTAMPTZ;
    	diff INT;
	BEGIN
    	SELECT id INTO sess_id
    	FROM session 
    	WHERE user_id = p_user_id
    	ORDER BY start_at DESC LIMIT 1;

    	IF sess_id IS NULL THEN
        	INSERT INTO session (user_id, start_at) VALUES (p_user_id, created_at) RETURNING id INTO sess_id;
        	INSERT INTO event (user_id, song_id, session_id, played_at) VALUES (p_user_id, p_song_id, sess_id, created_at) RETURNING id INTO new_event_id;
   		ELSE
        	SELECT played_at INTO most_recent
        	FROM event WHERE user_id = p_user_id AND session_id = sess_id 
        	ORDER BY played_at DESC LIMIT 1;

        	IF NOT FOUND THEN
            	INSERT INTO event (user_id, song_id, session_id, played_at) VALUES (p_user_id, p_song_id, sess_id, created_at) RETURNING id INTO new_event_id;
        	ELSE
            	SELECT (EXTRACT(EPOCH FROM (created_at - most_recent)) / 60)::integer INTO diff;
            	IF diff > 30 THEN
                	UPDATE session SET end_at = created_at WHERE user_id = p_user_id AND id = sess_id;
                	INSERT INTO session (user_id, start_at) VALUES (p_user_id, created_at) RETURNING id INTO sess_id;
                	INSERT INTO event (user_id, song_id, session_id, played_at) VALUES (p_user_id, p_song_id, sess_id, created_at) RETURNING id INTO new_event_id;
            	ELSE
                	INSERT INTO event (user_id, song_id, session_id, played_at) VALUES (p_user_id, p_song_id, sess_id, created_at) RETURNING id INTO new_event_id;
            	END IF;
        	END IF;
    	END IF;

    	RETURN QUERY SELECT new_event_id, sess_id, created_at;

	EXCEPTION
    	WHEN OTHERS THEN
        RAISE EXCEPTION 'An unexpected error occurred: % (Code: %)', SQLERRM, SQLSTATE;
	END;
	$$ LANGUAGE plpgsql;
	`
	_, err = tx.ExecContext(ctx, query)

	if err != nil {
		log.Println("Unable to set up logic for storing events + sessions")
		return err
	}

	log.Println("Established logic for storing events + sessions")
	
	return tx.Commit()
}

func checkUser(userID string, token string) bool{
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	row := databasePool.QueryRowContext(ctx, "SELECT token FROM auth_token WHERE user_id = $1", userID)
	

	var foundToken string

	err := row.Scan(&foundToken)
	if err == sql.ErrNoRows {
		log.Printf("No users found with id: %v", userID)
		return false
	} else if err != nil {
		log.Printf("Scan error: %v", err)
		return false
	}

	if token == foundToken {
		log.Println("Found user in db! Adding to cache...")
		auth_cache.Add(userID, token)
		return true
	} 
	
	return false
    
	
}
	

	


func Start(cache *lru.Cache[string, string], port string, db *sql.DB){
	auth_cache = cache
	databasePool = db
	http.HandleFunc("/", handler)

	fmt.Printf("Server is running on port %v\n", port)


	log.Fatal(http.ListenAndServe("0.0.0.0:"+port, nil))
}
