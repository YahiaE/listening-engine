package db

import (
	"database/sql"
	_ "github.com/jackc/pgx/v5/stdlib"
	"context"
	"log"
	"github.com/YahiaE/listening-engine/internal/config"
	"strings"
	"time"
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

func CreateTables(db *sql.DB) {
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
    defer cancel()
	
	query := `
	CREATE TABLE IF NOT EXISTS users (id UUID PRIMARY KEY, created_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP);
	CREATE TABLE IF NOT EXISTS auth_token (token TEXT PRIMARY KEY, user_id UUID NOT NULL REFERENCES users(id) ON DELETE CASCADE);
	CREATE TABLE IF NOT EXISTS song (id TEXT PRIMARY KEY, title TEXT, artist TEXT, album TEXT);
	CREATE TABLE IF NOT EXISTS session (
	id BIGINT GENERATED ALWAYS AS IDENTITY PRIMARY KEY, 
    user_id UUID NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    start_at TIMESTAMP WITH TIME ZONE, 
    end_at TIMESTAMP WITH TIME ZONE
	);
	CREATE TABLE IF NOT EXISTS event (
    id BIGINT GENERATED ALWAYS AS IDENTITY PRIMARY KEY, 
    user_id UUID NOT NULL REFERENCES users(id) ON DELETE CASCADE,
	session_id BIGINT NOT NULL REFERENCES session(id) ON DELETE CASCADE,
    song_id TEXT NOT NULL REFERENCES song(id), 
    created_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP
	);
	`

	log.Println("Establishing tables in database...")
	// log.Println(strings.Trim(query,"\n"))
	_, err := db.ExecContext(ctx, strings.Trim(query,"\n"))
	

	if err != nil {
		log.Println("Failed to establish tables in database")
	} else {
		log.Println("Success!")
	}
}

	
func Start() *sql.DB{
	db_url := config.Get("DATABASE_URL")
	db, err := NewDataBase(db_url)
	if err != nil {
		log.Fatalf("Failed to open DB: %v", err)
	}
	log.Println("Connected to database!")

	CreateTables(db)
	return db
}