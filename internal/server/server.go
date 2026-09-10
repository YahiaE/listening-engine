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
	"strings"

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

			log.Println(songRead)
			storeSong(songRead)

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

func normalize(s string) string {
    return strings.ToLower(strings.TrimSpace(s))
}

func storeSong(song models.Song) {
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	query := `
	INSERT INTO song (title, artist, album)
	VALUES ($1, $2, $3)
	ON CONFLICT (LOWER(TRIM(title)), LOWER(TRIM(artist)), LOWER(TRIM(album))) 
	DO NOTHING
	RETURNING id;
	`

	_, err := databasePool.ExecContext(ctx, query, song.Title, song.Artist, song.Album)
	if err != nil {
		log.Println(err)
	}
	log.Println("inserted song")
}

func checkUser(userID string, token string) bool{
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	row := databasePool.QueryRowContext(ctx, "SELECT token FROM auth_token WHERE user_id = $1", userID)
	defer cancel()

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
