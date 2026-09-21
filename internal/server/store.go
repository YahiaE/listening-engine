package server 

import (
	"database/sql"
	"log"
	"github.com/YahiaE/listening-engine/internal/models"
	"time"
	"context"
)


func storeUserAndToken(db *sql.DB, userID string, token string) error {
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	
	tx, err := db.BeginTx(ctx, nil)

	if err != nil {
		return err
	}

	defer tx.Rollback()

	_, err = tx.ExecContext(ctx, "INSERT INTO users (id) VALUES ($1)", userID)
	if err != nil {
		log.Println(err)
    	return err
	}
	log.Println("inserted user id")

	_, err = tx.ExecContext(ctx, "INSERT INTO auth_token (token, user_id) VALUES ($1, $2)", token, userID)
	if err != nil {
		log.Println(err)
    	return err
	}

	log.Println("inserted token")

	log.Printf("Registered new user!")

	return tx.Commit()
}

func storeSong(db *sql.DB, song models.Song) int64 {
	
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	query := `
	INSERT INTO song (title, artist, album)
	VALUES ($1, $2, $3)
	ON CONFLICT (LOWER(TRIM(title)), LOWER(TRIM(artist)), LOWER(TRIM(album))) 
	DO UPDATE SET title = EXCLUDED.title
	RETURNING id;
	`
	// updated title to the same title to just trigger return from query 

	var songID int64
    err := db.QueryRowContext(ctx, query, song.Title, song.Artist, song.Album).Scan(&songID)
    if err != nil {
        log.Printf("Failed to store/retrieve song: %v", err)
        return -1
    }

	return songID
}

func storeAndSessionizeEvent(db *sql.DB, user_id string, song_id int64) error {
    ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
    defer cancel()

    tx, err := db.BeginTx(ctx, nil)
    if err != nil {
        return err
    }
    defer tx.Rollback()

    query := `SELECT event_id, s_id, time_created FROM create_event_and_session($1, $2);`
    
    row := tx.QueryRowContext(ctx, query, user_id, song_id)

    var eventID int64
    var sessionID int64
    var createdAt time.Time
    err = row.Scan(&eventID, &sessionID, &createdAt)
    if err != nil {
        log.Println(err)
        return err
    }

    log.Printf("Stored event of ID %v into Session %v. Time made: %v", eventID, sessionID, createdAt)

    return tx.Commit()
}
