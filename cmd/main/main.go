package main

import (
	"github.com/YahiaE/listening-engine/internal/db"
	"github.com/YahiaE/listening-engine/internal/server"
	"github.com/YahiaE/listening-engine/internal/config"
	"log"
)

func main() {
	config.LoadEnv()
	port := config.Get("PORT")
	database := db.Start()
	defer database.Close()

	err := server.EstablishStorageFunction(database)

	if err != nil {
		log.Println(err)
	}
	
	server.Start(port, database)

	
	
}