package models

import (
	"time"
	"github.com/google/uuid"
)

// attributes are capitalized for store package and other packages to access
type ListeningEvent struct {
	ID uuid.UUID `json:"id"`
	UserID string `json:"user_id"`
	SongID string `json:"song_id"`
	PlayedAt time.Time `json:"played_at"`
}