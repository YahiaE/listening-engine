package main

import (
	"github.com/YahiaE/listening-engine/internal/db"
	"github.com/YahiaE/listening-engine/internal/server"
	"github.com/YahiaE/listening-engine/internal/config"
	"github.com/hashicorp/golang-lru/v2"
	"log"
)

func main() {
	cache, err := lru.New[string, string](400000)
	if err != nil {
		log.Println(err)
		return
	}

	config.LoadEnv()
	port := config.Get("PORT")
	database := db.Start()
	defer database.Close()

	err = server.EstablishStorageFunction(database)

	if err != nil {
		log.Println(err)
	}
	
	server.Start(cache, port, database)

	
	
}