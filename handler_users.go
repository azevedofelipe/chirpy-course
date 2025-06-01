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

func (cfg *apiConfig) handlerTokenRefresh(w http.ResponseWriter, r *http.Request) {
	refreshToken, err := auth.GetBearerToken(r.Header)
	if err != nil {
		log.Printf("Error getting refresh token: %s", err)
		w.WriteHeader(500)
		return
	}

	userTokenData, err := cfg.queries.GetUserFromRefreshToken(r.Context(), refreshToken)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			log.Printf("No token found: %s", err)
			w.WriteHeader(401)
			return
		}
		log.Printf("Error finding refresh token and user: %s", err)
		w.WriteHeader(500)
		return
	}

	if userTokenData.ExpiresAt.Before(time.Now().UTC()) || userTokenData.RevokedAt.Valid {
		log.Print("Refresh Token Invalid\n")
		w.WriteHeader(401)
		return
	}

	//Create JWT
	jwToken, err := auth.MakeJWT(userTokenData.ID, cfg.tokenSecret)
	if err != nil {
		log.Printf("Error creating JWT: %s", err)
		w.WriteHeader(500)
		return
	}

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

func (cfg *apiConfig) handlerTokenRevoke(w http.ResponseWriter, r *http.Request) {
	refreshToken, err := auth.GetBearerToken(r.Header)
	if err != nil {
		log.Printf("Error getting refresh token from header: %s", err)
		w.WriteHeader(500)
		return
	}

	err = cfg.queries.RevokeToken(r.Context(), refreshToken)
	if err != nil {
		log.Printf("Error getting refresh token from header: %s", err)
		w.WriteHeader(401)
		w.Write([]byte("Invalid token"))
		return
	}

	w.WriteHeader(204)
}

func (cfg *apiConfig) handlerUpdateUser(w http.ResponseWriter, r *http.Request) {

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

	accessToken, err := auth.GetBearerToken(r.Header)
	if err != nil {
		log.Printf("Error getting bearer token: %s", err)
		w.WriteHeader(401)
		return
	}

	userID, err := auth.ValidateJWT(accessToken, cfg.tokenSecret)
	if err != nil {
		log.Printf("Error validating JWT: %s", err)
		w.WriteHeader(401)
		return
	}

	log.Print("Creating hashed password")
	hashedPassword, err := auth.HashPassword(params.Password)
	if err != nil {
		log.Printf("Error hashing password: %s", err)
		w.WriteHeader(500)
		return
	}

	updatedUser, err := cfg.queries.UpdateUser(r.Context(), database.UpdateUserParams{
		Email:          params.Email,
		HashedPassword: hashedPassword,
		ID:             userID,
	})
	if err != nil {
		log.Printf("Error updating user: %s", err)
		w.WriteHeader(500)
		return
	}

	response := User{
		ID:        updatedUser.ID,
		CreatedAt: updatedUser.CreatedAt,
		UpdatedAt: updatedUser.UpdatedAt,
		Email:     updatedUser.Email,
	}

	dat, err := json.Marshal(response)
	if err != nil {
		log.Printf("Error marshalling response: %s", err)
		w.WriteHeader(500)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	w.Write(dat)
	w.WriteHeader(200)

}

func (cfg *apiConfig) handlerRedWebhook(w http.ResponseWriter, r *http.Request) {
	type parameters struct {
		Event string `json:"event"`
		Data  struct {
			UserID string `json:"user_id"`
		} `json:"data"`
	}

	decoder := json.NewDecoder(r.Body)
	params := parameters{}
	err := decoder.Decode(&params)
	if err != nil {
		log.Printf("Error reading input json: %s", err)
		w.WriteHeader(500)
		return
	}

	if params.Event != "user.upgraded" {
		w.WriteHeader(204)
		return
	}

	parsedID, err := uuid.Parse(params.Data.UserID)
	if err != nil {
		log.Printf("Error parsing uuid: %s", err)
		w.WriteHeader(500)
		return
	}

	err = cfg.queries.UpdateUserRedByID(r.Context(), parsedID)
	if err != nil {
		log.Printf("Error upgrading user: %s", err)
		w.WriteHeader(404)
		return
	}

	log.Printf("Sucessfully upgraded user")
	w.WriteHeader(204)
	return

}
