package server

import (
	"fmt"
	"encoding/json"
	"net/http"
	"io"
	"log"
	"database/sql"
	"github.com/YahiaE/listening-engine/internal/models"
	"github.com/YahiaE/listening-engine/internal/store"
	"github.com/YahiaE/listening-engine/internal/auth"
	"github.com/google/uuid"
	"time"
	"github.com/hashicorp/golang-lru/v2"
	"context"

)

var databasePool *sql.DB
var auth_cache *lru.Cache[string, string]

func handler(w http.ResponseWriter, r *http.Request){
	userID := r.Header.Get("User-ID")
	userToken := r.Header.Get("Auth-Token")

	isNewUser := isNewUser(userID, w)

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

<<<<<<< HEAD
=======

>>>>>>> 9a8dd1e (refactor(server): deleted un-used packages + move storage logic to individual folder)
			songID := store.StoreSong(databasePool, songRead)

			if songID > -1 {
				store.StoreAndSessionizeEvent(databasePool, userID, songID)
			}

		} else {
			log.Println("Expired token! Regenerating...")
		}
	}

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

func isNewUser(userID string, w http.ResponseWriter) bool{
	if len(userID) == 0 { // If no credentials, generate user ID + token and send to C# reader
		var token models.AuthToken
		log.Println("Received empty token. Generating ID and token for user")
		newUserUUID := uuid.New().String()
		newUserToken := auth.GenerateToken()
		
		log.Println("Adding user to cache...")
		auth_cache.Add(newUserUUID, auth.EncryptToken(newUserToken))

		token.Token = newUserToken
		token.UserID = newUserUUID
<<<<<<< HEAD
=======

>>>>>>> 9a8dd1e (refactor(server): deleted un-used packages + move storage logic to individual folder)
		store.StoreUserAndToken(databasePool, newUserUUID,auth.EncryptToken(newUserToken))
		tokenJson, err := json.Marshal(token)

		if err != nil {
			http.Error(w, "Internal server error: Unable to format credentials", http.StatusInternalServerError)
			return false
		}

		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		w.Write(tokenJson)		

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
