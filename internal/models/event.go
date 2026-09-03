package models

import (
	"time"
	"github.com/google/uuid"
)

// attributes are capitalized for store package and other packages to access
type ListeningEvent struct {
	UserID uuid.UUID `json:"user_id"`
	Song string `json:"song"`
	Artist string `json: "artist`
	Album string `json: "album`
	PlayedAt time.Time `json:"played_at"`
}