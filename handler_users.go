package main

import (
	"database/sql"
	"encoding/json"
	"errors"
	"log"
	"main/internal/auth"
	"main/internal/database"
	"net/http"
	"time"

	"github.com/google/uuid"
	_ "github.com/lib/pq"
)

type User struct {
	ID        uuid.UUID `json:"id"`
	CreatedAt time.Time `json:"created_at"`
	UpdatedAt time.Time `json:"updated_at"`
	Email     string    `json:"email"`
}

func (cfg *apiConfig) handlerUserCreation(w http.ResponseWriter, r *http.Request) {

	type parameters struct {
		Email    string `json:"email"`
		Password string `json:"password"`
	}

	header := 201

	decoder := json.NewDecoder(r.Body)
	params := parameters{}
	err := decoder.Decode(&params)
	if err != nil {
		log.Printf("Error reading json: %s", err)
		w.WriteHeader(500)
		return
	}

	log.Print("Creating hashed password")
	hashed, err := auth.HashPassword(params.Password)
	if err != nil {
		log.Printf("Error hashing password: %s", err)
		w.WriteHeader(500)
		return
	}

	log.Printf("Creating user with email, %s", params.Email)

	user, err := cfg.queries.CreateUser(r.Context(), database.CreateUserParams{
		Email:          params.Email,
		HashedPassword: hashed,
	})
	if err != nil {
		log.Printf("Error creating user in database: %s", err)
		w.WriteHeader(500)
		return
	}

	response := User{
		ID:        user.ID,
		CreatedAt: user.CreatedAt,
		UpdatedAt: user.UpdatedAt,
		Email:     user.Email,
	}

	dat, err := json.Marshal(response)
	if err != nil {
		log.Printf("Error marshalling json: %s", err)
		w.WriteHeader(500)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(header)
	w.Write(dat)

}

func (cfg *apiConfig) handlerUserLogin(w http.ResponseWriter, r *http.Request) {

	type userData struct {
		ID           uuid.UUID `json:"id"`
		CreatedAt    time.Time `json:"created_at"`
		UpdatedAt    time.Time `json:"updated_at"`
		Email        string    `json:"email"`
		Token        string    `json:"token"`
		RefreshToken string    `json:"refresh_token"`
	}

	type parameters struct {
		Email    string `json:"email"`
		Password string `json:"password"`
	}

	decoder := json.NewDecoder(r.Body)
	params := parameters{}
	err := decoder.Decode(&params)
	if err != nil {
		log.Printf("Error reading input json: %s", err)
		w.WriteHeader(500)
		return
	}

	log.Printf("Getting user with email, %s", params.Email)

	user, err := cfg.queries.GetUserByEmail(r.Context(), params.Email)
	if err != nil {
		log.Printf("Error getting user in database: %s", err)
		w.Header().Set("Content-Type", "text/plain; charset=utf-8")
		w.WriteHeader(401)
		w.Write([]byte("Invalid email or password"))
		return
	}

	err = auth.CheckPasswordHash(user.HashedPassword, params.Password)
	if err != nil {
		log.Printf("Error getting user in database: %s", err)
		w.Header().Set("Content-Type", "text/plain; charset=utf-8")
		w.WriteHeader(401)
		w.Write([]byte("Invalid email or password"))
		return
	}

	// Create JWT
	token, err := auth.MakeJWT(user.ID, cfg.tokenSecret)
	if err != nil {
		log.Printf("Error creating JWT: %s", err)
		w.WriteHeader(500)
		return
	}

	// Create Refresh Token
	refreshToken, err := auth.MakeRefreshToken()
	if err != nil {
		log.Printf("Error creating Refresh Token: %s", err)
		w.WriteHeader(500)
		return
	}

	refreshTokenJSON, err := cfg.queries.CreateToken(r.Context(), database.CreateTokenParams{
		Token:     refreshToken,
		UserID:    user.ID,
		ExpiresAt: time.Now().UTC().Add(60 * 24 * time.Hour),
	})
	if err != nil {
		log.Printf("Error creating Refresh Token: %s", err)
		w.WriteHeader(500)
		return
	}

	response := userData{
		ID:           user.ID,
		CreatedAt:    user.CreatedAt,
		UpdatedAt:    user.UpdatedAt,
		Email:        user.Email,
		Token:        token,
		RefreshToken: refreshTokenJSON.Token,
	}

	dat, err := json.Marshal(response)
	if err != nil {
		log.Printf("Error marshalling json: %s", err)
		w.WriteHeader(500)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(200)
	w.Write(dat)

}

	if err != nil {
		log.Printf("Error creating JWT: %s", err)
		w.WriteHeader(500)
		return
	}

	response := User{
		ID:        user.ID,
		CreatedAt: user.CreatedAt,
	type Token struct {
		JWToken string `json:"token"`
	}

	response := Token{
		JWToken: jwToken,
	}

	dat, err := json.Marshal(response)
	if err != nil {
		log.Printf("Error marshalling json: %s", err)
		w.WriteHeader(500)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(200)
	w.Write(dat)

}
