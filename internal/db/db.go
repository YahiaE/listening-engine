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

	db.SetMaxOpenConns(25)
    db.SetMaxIdleConns(25)
    db.SetConnMaxLifetime(5 * time.Minute)
    db.SetConnMaxIdleTime(1 * time.Minute)

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
	
	CREATE TABLE IF NOT EXISTS song (id BIGINT GENERATED ALWAYS AS IDENTITY PRIMARY KEY, title TEXT, artist TEXT, album TEXT);
	CREATE UNIQUE INDEX IF NOT EXISTS uq_normalized_song ON song (LOWER(TRIM(title)), LOWER(TRIM(artist)), LOWER(TRIM(album))); 
	
	CREATE TABLE IF NOT EXISTS session (
	id BIGINT GENERATED ALWAYS AS IDENTITY PRIMARY KEY, 
    user_id UUID NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    start_at TIMESTAMP WITH TIME ZONE, 
    end_at TIMESTAMP WITH TIME ZONE
	);
	CREATE INDEX IF NOT EXISTS idx_session_user_start_desc ON session(user_id, start_at DESC);

	CREATE TABLE IF NOT EXISTS event (
    id BIGINT GENERATED ALWAYS AS IDENTITY PRIMARY KEY, 
    user_id UUID NOT NULL REFERENCES users(id) ON DELETE CASCADE,
	session_id BIGINT NOT NULL REFERENCES session(id) ON DELETE CASCADE,
    song_id BIGINT NOT NULL REFERENCES song(id), 
    played_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP
	);

	CREATE INDEX IF NOT EXISTS idx_user_event_by_session_desc ON event(user_id, session_id, played_at DESC);
	`

	log.Println("Establishing tables in database...")
	// log.Println(strings.Trim(query,"\n"))
	_, err := db.ExecContext(ctx, strings.Trim(query,"\n"))
	

	if err != nil {
		log.Fatalf("Failed to establish tables in database: %v", err)
	} else {
		log.Println("Success!")
	}
}

func EstablishStorageFunction(db *sql.DB) error {	
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	tx, err := db.BeginTx(ctx, nil)

	if err != nil {
		return err
	}

	defer tx.Rollback()

	query := `
	CREATE OR REPLACE FUNCTION create_event_and_session(p_user_id UUID, p_song_id BIGINT)
	RETURNS TABLE (event_id BIGINT, s_id BIGINT, time_created TIMESTAMPTZ) AS $$
	DECLARE
    	sess_id BIGINT;
    	new_event_id BIGINT;
    	created_at TIMESTAMPTZ := NOW();
    	most_recent TIMESTAMPTZ;
    	diff INT;
	BEGIN
    	SELECT id INTO sess_id
    	FROM session 
    	WHERE user_id = p_user_id
    	ORDER BY start_at DESC LIMIT 1;

    	IF sess_id IS NULL THEN
        	INSERT INTO session (user_id, start_at) VALUES (p_user_id, created_at) RETURNING id INTO sess_id;
        	INSERT INTO event (user_id, song_id, session_id, played_at) VALUES (p_user_id, p_song_id, sess_id, created_at) RETURNING id INTO new_event_id;
   		ELSE
        	SELECT played_at INTO most_recent
        	FROM event WHERE user_id = p_user_id AND session_id = sess_id 
        	ORDER BY played_at DESC LIMIT 1;

        	IF NOT FOUND THEN
            	INSERT INTO event (user_id, song_id, session_id, played_at) VALUES (p_user_id, p_song_id, sess_id, created_at) RETURNING id INTO new_event_id;
        	ELSE
            	SELECT (EXTRACT(EPOCH FROM (created_at - most_recent)) / 60)::integer INTO diff;
            	IF diff > 30 THEN
                	UPDATE session SET end_at = created_at WHERE user_id = p_user_id AND id = sess_id;
                	INSERT INTO session (user_id, start_at) VALUES (p_user_id, created_at) RETURNING id INTO sess_id;
                	INSERT INTO event (user_id, song_id, session_id, played_at) VALUES (p_user_id, p_song_id, sess_id, created_at) RETURNING id INTO new_event_id;
            	ELSE
                	INSERT INTO event (user_id, song_id, session_id, played_at) VALUES (p_user_id, p_song_id, sess_id, created_at) RETURNING id INTO new_event_id;
            	END IF;
        	END IF;
    	END IF;

    	RETURN QUERY SELECT new_event_id, sess_id, created_at;

	EXCEPTION
    	WHEN OTHERS THEN
        RAISE EXCEPTION 'An unexpected error occurred: % (Code: %)', SQLERRM, SQLSTATE;
	END;
	$$ LANGUAGE plpgsql;
	`
	_, err = tx.ExecContext(ctx, query)

	if err != nil {
		log.Println("Unable to set up logic for storing events + sessions")
		return err
	}

	log.Println("Established logic for storing events + sessions")
	
	return tx.Commit()
}

	
func Start() *sql.DB{
	db_url := config.Get("DATABASE_URL")
	db, err := NewDataBase(db_url)
	if err != nil {
		log.Fatalf("Failed to open DB: %v", err)
	}
	log.Println("Connected to database!")

	CreateTables(db)
	err = EstablishStorageFunction(db)
	if err != nil {
		log.Fatalf("Failed to set storage logic for DB: %v", err)
	}
	return db
}