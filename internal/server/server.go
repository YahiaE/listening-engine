package server

import (
	"context"
	"database/sql"
	"encoding/json"
	"errors"
	"fmt"
	"log"
	"net/http"
	"time"

	"github.com/google/uuid"
	lru "github.com/hashicorp/golang-lru/v2"

	"github.com/YahiaE/listening-engine/internal/auth"
	"github.com/YahiaE/listening-engine/internal/models"
	"github.com/YahiaE/listening-engine/internal/store"
)

// Server holds dependencies for HTTP handlers (Dependency Injection)
type Server struct {
	db        *sql.DB
	authCache *lru.Cache[string, string] // Key: encryptedToken, Value: userID
}

// NewServer initializes a new Server instance
func NewServer(db *sql.DB, cache *lru.Cache[string, string]) *Server {
	return &Server{
		db:        db,
		authCache: cache,
	}
}

func (s *Server) Routes() http.Handler {
	mux := http.NewServeMux()
	mux.HandleFunc("/telemetry", s.handleTelemetry)
	return mux
}

func (s *Server) handleTelemetry(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	userID := r.Header.Get("User-ID")
	userToken := r.Header.Get("Auth-Token")

	// 1. Unauthenticated / First-time user onboarding
	if userID == "" || userToken == "" {
		s.handleNewUser(w, r.Context())
		return
	}

	// 2. Authenticate session token
	authenticatedUserID, err := s.authenticateUser(r.Context(), userID, userToken)
	if err != nil {
		log.Printf("Authentication failed for User-ID %s: %v", userID, err)
		http.Error(w, "Unauthorized", http.StatusUnauthorized)
		return
	}

	// 3. Decode JSON stream directly
	var song models.Song
	if err := json.NewDecoder(r.Body).Decode(&song); err != nil {
		http.Error(w, "Invalid JSON payload", http.StatusBadRequest)
		return
	}

	if song.Title == "" {
		// Empty event tick received - soft succeed
		w.WriteHeader(http.StatusNoContent)
		return
	}

	// 4. Record event
	songID := store.StoreSong(s.db, song)
	if songID > -1 {
		store.StoreAndSessionizeEvent(s.db, authenticatedUserID, songID)
	}

	w.WriteHeader(http.StatusAccepted)
}

func (s *Server) authenticateUser(ctx context.Context, userID, rawToken string) (string, error) {
	encryptedToken := auth.EncryptToken(rawToken)

	// Check LRU Cache (Key = Token, Value = UserID)
	if cachedUserID, ok := s.authCache.Get(encryptedToken); ok {
		if cachedUserID == userID {
			return cachedUserID, nil
		}
	}

	// Cache miss - fallback to Database
	queryCtx, cancel := context.WithTimeout(ctx, 3*time.Second)
	defer cancel()

	var dbToken string
	query := "SELECT token FROM auth_token WHERE user_id = $1"
	err := s.db.QueryRowContext(queryCtx, query, userID).Scan(&dbToken)

	if errors.Is(err, sql.ErrNoRows) {
		return "", fmt.Errorf("user not found")
	} else if err != nil {
		return "", fmt.Errorf("db error: %w", err)
	}

	if dbToken != encryptedToken {
		return "", fmt.Errorf("invalid token match")
	}

	// Store in cache upon successful DB verification
	s.authCache.Add(encryptedToken, userID)
	return userID, nil
}

func (s *Server) handleNewUser(w http.ResponseWriter, ctx context.Context) {
	log.Println("Credentials missing. Generating new user ID and auth token.")

	newUUID := uuid.New().String()
	rawToken := auth.GenerateToken()
	encryptedToken := auth.EncryptToken(rawToken)

	// Persist credentials
	if err := store.StoreUserAndToken(s.db, newUUID, encryptedToken); err != nil {
		log.Printf("Failed to store new user in DB: %v", err)
		http.Error(w, "Internal server error", http.StatusInternalServerError)
		return
	}

	// Cache token
	s.authCache.Add(encryptedToken, newUUID)

	resp := models.AuthToken{
		UserID: newUUID,
		Token:  rawToken, // Return raw token to client once
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusCreated)
	_ = json.NewEncoder(w).Encode(resp)
}

func Start(cache *lru.Cache[string, string], port string, db *sql.DB) {
	srv := NewServer(db, cache)

	httpServer := &http.Server{
		Addr:         ":" + port,
		Handler:      srv.Routes(),
		ReadTimeout:  5 * time.Second,
		WriteTimeout: 10 * time.Second,
		IdleTimeout:  120 * time.Second,
	}

	log.Printf("Server listening on port %s...", port)
	log.Fatal(httpServer.ListenAndServe())
}