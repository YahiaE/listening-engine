package main

import (
	"github.com/YahiaE/listening-engine/internal/db"
	"github.com/YahiaE/listening-engine/internal/server"
	"github.com/YahiaE/listening-engine/internal/config"
	"github.com/hashicorp/golang-lru/v2"
	"log"
)

func main() {
	auth_cache, err := lru.New[string, string](200000)
	if err != nil {
		log.Println(err)
		return
	}

	otp_cache, err := lru.New[string, string](50000)
	if err != nil {
		log.Println(err)
		return
	}

	config.LoadEnv()
	port := config.Get("PORT")
	database := db.Start()
	defer database.Close()

	server.Start(auth_cache, otp_cache, port, database)

	
	
}