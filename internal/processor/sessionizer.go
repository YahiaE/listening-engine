package processor

import (
	"time"
	"github.com/YahiaE/listening-engine/internal/models"
)

func Sessionize(event ListeningEvent) error {
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	fetchQuery := `SELECT (user_id, end_at) FROM session WHERE user_id = $1`


	
	return sessions
}
