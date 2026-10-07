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
	"github.com/YahiaE/listening-engine/internal/email"
)

// Server holds dependencies for HTTP handlers (Dependency Injection)
type Server struct {
	db        *sql.DB
	authCache *lru.Cache[string, string] // Key: encryptedToken, Value: userID
	otpCache *lru.Cache[string, string] // Key: email, Value: one-time passcode
}

// NewServer initializes a new Server instance
func NewServer(db *sql.DB, cache_auth *lru.Cache[string, string], cache_otp *lru.Cache[string, string]) *Server {
	return &Server{
		db:        db,
		authCache: cache_auth,
		otpCache: cache_otp,
	}
}

func (s *Server) Routes() http.Handler {
	mux := http.NewServeMux()
	mux.HandleFunc("/", s.handleTelemetry)
	mux.HandleFunc("/send-otp", s.handleOTP)
	mux.HandleFunc("/verify-otp", s.handleNewUser)
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
		log.Println("Unauthorized User Detected. Post to /grab-otp to authorize")
		http.Error(w, "Unauthorized", http.StatusUnauthorized)
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

func (s *Server) handleNewUser(w http.ResponseWriter, r *http.Request) {
	var authOTP models.OTPPairing
	isVerified := false

	err := json.NewDecoder(r.Body).Decode(&authOTP)
	if err != nil {
		log.Printf("Unable to receive user's entered OTP: %v", err)
		http.Error(w, "Invalid JSON payload", http.StatusBadRequest)
		return
	}

	if cachedOTP, ok := s.otpCache.Get(authOTP.Email); ok {
		if cachedOTP == authOTP.Code {
			isVerified = true
			log.Println("OTP verified!")
		} else {
			log.Printf("Invalid OTP entered! %s was entered but it is %s", authOTP.Code, cachedOTP)
			w.WriteHeader(http.StatusUnauthorized)
			return
		}
	}

	if isVerified {
		hasExistingEmail := false
		queryCtx, cancel := context.WithTimeout(r.Context(), 3*time.Second)
		defer cancel()

		var userID string
		var rawToken string
		var newUUID string
		var resp models.AuthToken


		query := "SELECT id FROM users WHERE email = $1"
		err = s.db.QueryRowContext(queryCtx, query, authOTP.Email).Scan(&userID)

		if errors.Is(err, sql.ErrNoRows) {
			log.Printf("email (%s) not found: %v", authOTP.Email, err)
		} else if err != nil {
			log.Printf("db error: %v", err)
			return 
		}

		if userID != "" {
			hasExistingEmail = true
		}

		if hasExistingEmail {

			/*
				Generate new token
				Send to local machine to use for verification!
				return to stop function
			*/
			log.Printf("Found existing email! Generating new token for user %s", userID)
			rawToken = auth.GenerateToken()
			encryptedToken := auth.EncryptToken(rawToken)
			if err = store.StoreToken(s.db, userID, encryptedToken); err != nil {
				log.Printf("Failed to store new credential in DB: %v", err)
				http.Error(w, "Internal server error", http.StatusInternalServerError)
				return
			}

			s.authCache.Add(encryptedToken, userID)
			resp = models.AuthToken{
				UserID: userID,
				Token:  rawToken, // Return raw token to client once
			}

			

		} else {
			log.Println("No existing email detected! Generating new user ID and auth token...")

			rawToken = auth.GenerateToken()
			newUUID = uuid.New().String()

			// log.Println(rawToken, newUUID)
		
			encryptedToken := auth.EncryptToken(rawToken)
			// Persist credentials
			if err = store.StoreUserAndToken(s.db, newUUID, encryptedToken, authOTP.Email); err != nil {
				log.Printf("Failed to store new user in DB: %v", err)
				http.Error(w, "Internal server error", http.StatusInternalServerError)
				return
			}

			// Cache token
			s.authCache.Add(encryptedToken, newUUID)

			resp = models.AuthToken{
				UserID: newUUID,
				Token:  rawToken, // Return raw token to client once
			}
		}

		

		

		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusCreated)
		_ = json.NewEncoder(w).Encode(resp)
	}
	
}

func (s *Server) handleOTP(w http.ResponseWriter, r *http.Request) {

	userEmail := r.Header.Get("Email")
	log.Printf("Sending OTP to user's email: %s", userEmail)
	otp, err := email.SendCode(userEmail)
	if err != nil {
		log.Printf("Unable to send OTP to %s: %v", userEmail, err)
		return
	}
	s.otpCache.Add(userEmail, otp)

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusCreated)
}

func Start(cache_auth *lru.Cache[string, string], cache_otp *lru.Cache[string, string], port string, db *sql.DB) {
	srv := NewServer(db, cache_auth, cache_otp)

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