package main

import (
	"github.com/YahiaE/listening-engine/internal/config"
	"github.com/YahiaE/listening-engine/internal/db"
	"github.com/YahiaE/listening-engine/internal/server"
	"github.com/hashicorp/golang-lru/v2"
	"log"
)

func main() {
	// Bound in-memory caches keep short-lived OTP and authentication data out of
	// the database while protecting the server from unbounded memory growth.
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

	// Database startup also creates tables and the sessionizing SQL function.
	database := db.Start()
	defer database.Close()

	// HTTP handlers receive the database and caches through Server dependencies.
	server.Start(auth_cache, otp_cache, port, database)

}
