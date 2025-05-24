package main

import (
	"database/sql"
	"encoding/json"
	"fmt"
	"log"
	"main/internal/database"
	"net/http"
	"os"
	"strings"
	"sync/atomic"
	"time"

	"github.com/google/uuid"
	"github.com/joho/godotenv"
	_ "github.com/lib/pq"
)

type apiConfig struct {
	fileServerHits atomic.Int32
	queries        *database.Queries
	platform       string
}

type User struct {
	ID        uuid.UUID `json:"id"`
	CreatedAt time.Time `json:"created_at"`
	UpdatedAt time.Time `json:"updated_at"`
	Email     string    `json:"email"`
}

func main() {
	godotenv.Load()

	dbURL := os.Getenv("DB_URL")

	db, err := sql.Open("postgres", dbURL)
	if err != nil {
		fmt.Errorf("Error opening database: %s", err)
	}
	dbQueries := database.New(db)

	const port = "8080"
	const filepathRoot = "."

	apiCfg := apiConfig{
		fileServerHits: atomic.Int32{},
		queries:        dbQueries,
		platform:       os.Getenv("PLATFORM"),
	}

	mux := http.NewServeMux()
	mux.Handle("/app/", apiCfg.middlewareMetricsInc(http.StripPrefix("/app", http.FileServer(http.Dir(filepathRoot)))))
	mux.HandleFunc("GET /api/healthz", handlerReadiness)
	mux.HandleFunc("GET /admin/metrics", apiCfg.handlerHitCount)
	mux.HandleFunc("POST /admin/reset", apiCfg.handlerResetCount)
	mux.HandleFunc("POST /api/validate_chirp", handlerValidateChirp)
	mux.HandleFunc("POST /api/users", apiCfg.handlerUserCreation)

	srv := &http.Server{
		Addr:    ":" + port,
		Handler: mux,
	}

	log.Printf("Serving on port: %s\n", port)
	log.Fatal(srv.ListenAndServe())

}

func handlerReadiness(w http.ResponseWriter, r *http.Request) {
	w.Header().Add("Content-Type", "text/plain; charset=utf-8")
	w.WriteHeader(http.StatusOK)
	w.Write([]byte(http.StatusText(http.StatusOK)))
}

func (cfg *apiConfig) handlerUserCreation(w http.ResponseWriter, r *http.Request) {

	type parameters struct {
		Email string `json:"email"`
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
	log.Printf("Creating user with email, %s", params.Email)

	user, err := cfg.queries.CreateUser(r.Context(), params.Email)
	if err != nil {
		log.Printf("Error creating user in database: %s", err)
		w.WriteHeader(500)
		return
	}

	dat, err := json.Marshal(user)
	if err != nil {
		log.Printf("Error marshalling json: %s", err)
		w.WriteHeader(500)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(header)
	w.Write(dat)

}

func handlerValidateChirp(w http.ResponseWriter, r *http.Request) {
	type parameters struct {
		Body string `json:"body"`
	}

	type returnVals struct {
		Valid       bool   `json:"valid"`
		CleanedBody string `json:"cleaned_body"`
	}

	valid := true
	header := 200

	decoder := json.NewDecoder(r.Body)
	params := parameters{}
	err := decoder.Decode(&params)
	if err != nil {
		log.Printf("Error reading json: %s", err)
		w.WriteHeader(500)
		return
	}
	body := params.Body
	if len(body) > 140 {
		log.Print("Chirp is too long", err)
		valid = false
		header = 400
	}

	updated_body := checkProfanity(body)

	respBody := returnVals{
		Valid:       valid,
		CleanedBody: updated_body,
	}

	dat, err := json.Marshal(respBody)
	if err != nil {
		log.Printf("Error marshalling json: %s", err)
		w.WriteHeader(500)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(header)
	w.Write(dat)

}

func (cfg *apiConfig) handlerHitCount(w http.ResponseWriter, r *http.Request) {
	w.Header().Add("Content-Type", "text/html; charset=utf-8")
	w.WriteHeader(http.StatusOK)
	response := fmt.Sprintf("<html> <body> <h1>Welcome, Chirpy Admin</h1> <p>Chirpy has been visited %d times!</p> </body> </html>", cfg.fileServerHits.Load())
	fmt.Fprintf(w, response)
}

func (cfg *apiConfig) handlerResetCount(w http.ResponseWriter, r *http.Request) {
	if cfg.platform != "dev" {
		w.WriteHeader(403)
		return
	}

	err := cfg.queries.DeleteUsers(r.Context())
	if err != nil {
		log.Printf("Error deleting users: %s", err)
		w.WriteHeader(500)
		return
	}
	w.WriteHeader(200)
	log.Print("All users deleted")
	return
}

func (cfg *apiConfig) middlewareMetricsInc(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		cfg.fileServerHits.Add(1)
		next.ServeHTTP(w, r)
	})
}

func checkProfanity(s string) string {
	profanity := []string{"kerfuffle", "sharbert", "fornax"}
	words := strings.Split(s, " ")
	for idx, word := range words {
		for _, prof := range profanity {
			if strings.ToLower(word) == prof {
				words[idx] = "****"
			}
		}
	}

	return strings.Join(words, " ")
}
