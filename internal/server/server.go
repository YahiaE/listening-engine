package server

import (
	"fmt"
	"encoding/json"
	"net/http"
	"log"
	"database/sql"
	"github.com/YahiaE/listening-engine/internal/models"
	"github.com/google/uuid"
)

var databasePool *sql.DB

func handler(w http.ResponseWriter, r *http.Request){
	var event models.ListeningEvent

	err := json.NewDecoder(r.Body).Decode(&event)

	if err != nil {
		http.Error(w, "Invalid JSON", http.StatusBadRequest)
		return
	}

	w.WriteHeader(http.StatusOK)
	if event.UserID == uuid.Nil {
		event.UserID = uuid.New()
		byteSlice := event.UserID[:]
		log.Println(len(byteSlice))
		w.Write(byteSlice)
	}

	log.Println("Received body: ", event)

	
}

func Start(port string, db *sql.DB){
	databasePool = db
	stats := databasePool.Stats
	fmt.Println(stats)
	http.HandleFunc("/", handler)

	fmt.Printf("Server is running on port %v", port)


	log.Fatal(http.ListenAndServe("0.0.0.0:"+port, nil))
}
