package db

import (
	"database/sql"
	_ "github.com/jackc/pgx/v5/stdlib"
	"log"
	"fmt"
	"github.com/YahiaE/listening-engine/internal/config"
)


func NewDataBase(connString string) (*sql.DB, error){
	db, err := sql.Open("pgx", connString)

	if err != nil {
		return nil, err
	}

	if err := db.Ping(); err != nil {
		db.Close()
		return nil, err
	}


	return db, nil
}

	
func Start() *sql.DB{
	db_url := config.Get("DATABASE_URL")
	db, err := NewDataBase(db_url)
	if err != nil {
		log.Fatalf("Failed to open DB: %v", err)
	}

	
	fmt.Println("Connected to database!")
	return db
}