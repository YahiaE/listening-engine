package main

import (
	"github.com/YahiaE/listening-engine/internal/db"
	"github.com/YahiaE/listening-engine/internal/server"
	"github.com/YahiaE/listening-engine/internal/config"
)

func main() {
	config.LoadEnv()
	port := config.Get("PORT")
	database := db.Start()

	defer database.Close()

	server.Start(port, database)
}