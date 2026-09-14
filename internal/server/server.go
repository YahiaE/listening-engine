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
	"context"

)

var databasePool *sql.DB


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
		isValidUser := checkUser(userID, auth.EncryptToken(userToken))
		
		// check if token received is associated with the user
		// if token exist + connected to user => add event
		if isValidUser {
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

			
			
			songID, err := storeSong(songRead)

			if err != nil {
				http.Error(w, "Internal server error: Unable to store song data", http.StatusInternalServerError)
				return
			}

			log.Printf("Song: %v, ID: %v", songRead, songID)

			log.Println(userID)
			userEvent, err := storeEvent(userID, songID)
			log.Printf("user id: %v \n song id: %v \n session id: %v", userEvent.UserID, userEvent.SongID, userEvent.SessionID)

			if err != nil {
				http.Error(w, "Internal server error: Unable to store event data", http.StatusInternalServerError)
				return
			}
			
			
		
			

		} else {
			log.Println("Expired token! Regenerating...")
		}
		// when event added => sessionize it
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

func storeEvent(user_id string, song_id int64) (models.ListeningEvent, error){
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	var userEvent models.ListeningEvent
	userEvent.UserID = user_id
	userEvent.SongID = song_id
	userEvent.PlayedAt = time.Now()

	err := Sessionize(&userEvent)
	if err != nil {
		log.Println(err)
		return models.ListeningEvent{}, err
	}

	query := `INSERT INTO event (user_id, song_id, session_id, played_at) VALUES ($1, $2, $3, $4)`
	_, err = databasePool.ExecContext(ctx, query, user_id, song_id, userEvent.SessionID, userEvent.PlayedAt)

	

	return userEvent, nil

	
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

func Sessionize(event *models.ListeningEvent) error {	
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	tx, err := databasePool.BeginTx(ctx, nil)

	if err != nil {
		return err
	}

	defer tx.Rollback()


	query := `SELECT id, end_at FROM session WHERE user_id = $1 ORDER BY start_at DESC LIMIT 1;`
	row := tx.QueryRowContext(ctx, query, event.UserID)
	
	/*
	timeEnd not defined = 0 value => basically a placeholder for now. 
	basically should have an end when a new session is made beyond the current 
	(current event time - latest event time from most recent session > 30 minutes)
	*/
	
	var timeEnd time.Time
	var session_id int64
	err = row.Scan(&session_id, &timeEnd)

	if err == sql.ErrNoRows { // no sessions
		log.Printf("User %v has no listening sessions. Creating session now...", event.UserID)
		
		query = `INSERT into session (user_id, start_at) VALUES ($1, $2) RETURNING id;`
		err = tx.QueryRowContext(ctx, query, event.UserID, event.PlayedAt).Scan(&session_id)

		if err != nil {
			log.Println("Failed to insert new session for user")
			return err
		}

		event.SessionID = session_id

	} else if err != nil { // error with session
		log.Println("Failed to grab recent session from user")
		return err
	} else { // check current session for time gap > 30 or not

		// if gap > 30 => update end_time for current session and create new session that new event will belong to
		query = `SELECT played_at FROM event WHERE user_id = $1 AND session_id = $2 ORDER BY played_at DESC LIMIT 1;` // grab time from most rec event
		row = tx.QueryRowContext(ctx, query, event.UserID, session_id)
		var mostRecentEventTime time.Time
		err := row.Scan(&mostRecentEventTime)
		if err != nil {
			log.Println("Failed to grab most recent event session for user")
			return err
		}
		
		if event.PlayedAt.Sub(mostRecentEventTime) > (30 * time.Minute) { 
			query = `UPDATE session SET end_at = $1 WHERE user_id = $2 AND id = $3`
			_, err = tx.ExecContext(ctx, query, event.PlayedAt, event.UserID, session_id)

			if err != nil {
				log.Println("Failed to update end time of most recent event session for user")
				return err
			}
			

			query = `INSERT into session (user_id, start_at) VALUES ($1, $2, $3) RETURNING id;`
			err = tx.QueryRowContext(ctx, query, event.UserID, event.PlayedAt).Scan(&session_id)

			if err != nil {
				log.Println("Failed to insert new session for user")
				return err
			}

			event.SessionID = session_id

		} else { // event belongs in current session
			event.SessionID = session_id
		}	
	}

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
		return true
	} 
	log.Println(foundToken)
	return false
    
	
}
	

	


func Start(port string, db *sql.DB){
	databasePool = db
	http.HandleFunc("/", handler)

	fmt.Printf("Server is running on port %v\n", port)


	log.Fatal(http.ListenAndServe("0.0.0.0:"+port, nil))
}
