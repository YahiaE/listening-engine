package processor

import (
	"time"
	"github.com/YahiaE/listening-engine/internal/models"
	"slices"
)

type Session struct {
	Events []models.ListeningEvent
}

func Sessionize(events []models.ListeningEvent) []Session {
	if len(events) == 0 {
		return nil
	}

	sessions := []Session{}

	if len(events) == 1 {
		s := Session {Events: []models.ListeningEvent{events[0]}}
		
		return append(sessions, s)
	}

	slices.SortFunc(events, func(a, b models.ListeningEvent) int {
		duration := a.PlayedAt.Sub(b.PlayedAt)
		return int(duration.Seconds())
	})

	currentSession := Session{Events: []models.ListeningEvent{}}

	for i := 0; i < len(events); i++ {
		currentSession.Events = append(currentSession.Events, events[i])
		
		if i < len(events)-1 {
			if events[i+1].PlayedAt.Sub(events[i].PlayedAt) > 30*time.Minute {
				sessions = append(sessions,currentSession)
				currentSession = Session{Events: []models.ListeningEvent{}}
			}
		}

		
		
	}

	if len(currentSession.Events) > 0 {
		sessions = append(sessions,currentSession)
	}
	

	/*
	initialize session variable to store events

	loop through events
	- add curr event to session
	- if gap b/w curr and next > 30 (curr-next): 
	* add session to list of sessions
	* reinitialize session to be empty

	if session is not empty:
	* add to list of sessions
	*/

	return sessions
}
