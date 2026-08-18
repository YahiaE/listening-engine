package main

import (
	"os"
	"log"
	"fmt"
	"github.com/YahiaE/listening-engine/internal/db"
	"github.com/joho/godotenv"
)

func main() {
	err := godotenv.Load()
	if err != nil {
		log.Fatalf("Failed to open .env: %v", err)
	}
	db_url := os.Getenv("DATABASE_URL")
	db, err := db.NewDataBase(db_url)
	if err != nil {
		log.Fatalf("Failed to open DB: %v", err)
	}

	

	fmt.Println("Connected!")

}