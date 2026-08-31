package store

import (
	"database/sql"
	"github.com/YahiaE/listening-engine/internal/models"
	"context"
)

type Store struct {
	db *sql.DB;
}

/* context.Context = interface that attaches to request that is 
monitored by the db driver to see if the request should
timeout / cancel
=> timeout = context signals that deadline has exceeded
=> cancellation = context signals manual cancel / parent cancellation
*/
func (s *Store) SaveEvent(ctx context.Context, e models.ListeningEvent) error {
	// returns sql.Results and error. results arent needed rn, so _ is used to signal it as a placeholder that doesnt need to be used
	_, err := s.db.ExecContext(ctx, "INSERT INTO listening_event (id, user_id, song_id, played_at) VALUES ($1, $2, $3, $4)", e.ID, e.UserID, e.SongID, e.PlayedAt)
	if err != nil{
		return err
	}

	return nil
}