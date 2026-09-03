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
)

var databasePool *sql.DB

type Credential struct {
	UserID string
	Token string
}

func handler(w http.ResponseWriter, r *http.Request){
	var credential Credential
	var event models.ListeningEvent
	
	userToken := r.Header.Get("Auth-Token")

	if len(userToken) == 0{
		log.Println("Received empty token. Generating ID and token for user")
		userUUID := uuid.New().String()
		userToken, err := auth.GenerateToken();

		if err != nil {
			http.Error(w, "Internal server error: Unable to generate user credentials", http.StatusInternalServerError)
			return
		}

		credential.UserID = userUUID
		credential.Token = userToken

		credentialsJson, err := json.Marshal(credential)

		if err != nil {
			http.Error(w, "Internal server error: Unable to format credentials", http.StatusInternalServerError)
			return
		}

		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		w.Write(credentialsJson)		
	}

	event.UserID = credential.UserID
	bodyBytes, err := io.ReadAll(r.Body)

	if err != nil {
		http.Error(w, "Failed to read request body", http.StatusBadRequest)
		return
	}
	defer r.Body.Close()

	err = json.Unmarshal(bodyBytes, &event)
	if err != nil {
		http.Error(w, "Internal server error: Unable to map data into usable object", http.StatusInternalServerError)
		return
	}

	
	log.Println(event.Song)



	
	
}

	

	


func Start(port string, db *sql.DB){
	databasePool = db
	stats := databasePool.Stats
	fmt.Println(stats)
	http.HandleFunc("/", handler)

	fmt.Printf("Server is running on port %v\n", port)


	log.Fatal(http.ListenAndServe("0.0.0.0:"+port, nil))
}
