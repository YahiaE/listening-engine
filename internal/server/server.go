package server

import (
	"fmt"
	"io"
	"net/http"
	"log"
	"database/sql"
)

var databasePool *sql.DB

func handler(w http.ResponseWriter, r *http.Request){
    bodyBytes, err := io.ReadAll(r.Body)

    if err != nil {
        http.Error(w, "Failed to read request body", http.StatusBadRequest)
        return
    }

    log.Println("Received:", string(bodyBytes))

    w.WriteHeader(http.StatusOK)
    w.Write([]byte("received"))
}

func Start(port string, db *sql.DB){
	databasePool := db
	stats := databasePool.Stats
	fmt.Println(stats)
	http.HandleFunc("/", handler)

	fmt.Printf("Server is running on port %v", port)


	log.Fatal(http.ListenAndServe("0.0.0.0:"+port, nil))
}
