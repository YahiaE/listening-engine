package models

import (
	"time"
	// "github.com/google/uuid"
)

// attributes are capitalized for store package and other packages to access

// event => connects back to user, song, and session
type ListeningEvent struct {
	// id (primary key) auto generated
	UserID string `json:"user_id"` // foreign key
	SongID string `json:"song_id"` // foreign key
	SessionID int `json: "session_id` // foreign key
	CreatedAt time.Time `json:"created_at"`
}

// user => unique id based on UUID
type User struct {
	ID string `json:"id"` // primary key
	CreatedAt time.Time `json:"created_at"`
}

// token => connects back to user
type AuthToken struct {
	Token string `json:"auth_token"` // primary key
	UserID string `json:"user_id"` // foreign key
	CreatedAt time.Time `json:"created_at"`
}

// song => used as a way to not fill duplicate data (multiple events w/ same song but diff timestamps)
type Song struct {
	ID string `json:"id"` // primary key: normalized w/ data
	Title string `json:"title"`
	Artist string `json:"artist"`
	Album string `json:"album"`
}

// session => connects back to user
type Session struct {
	// id (primary key) auto generated
	UserID string `json:"user_id"` // foreign key
	StartAt time.Time `json:"start_at"`
	EndAt time.Time `json:"end_at"`
	
}
