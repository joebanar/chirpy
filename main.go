package main

import (
	"database/sql"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"os"
	"regexp"
	"sync/atomic"

	"github.com/joebanar/chirpy/database"
	"github.com/joho/godotenv"
	_ "github.com/lib/pq"
)

type apiConfig struct {
	fileserverHits atomic.Int32
	queries        *database.Queries
}

func (cfg *apiConfig) middlewareMetricsInc(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		cfg.fileserverHits.Add(1)
		next.ServeHTTP(w, r)
	})
}

func (cfg *apiConfig) metricsHandler(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		w.Header().Set("Allow", "GET")
		http.Error(w, "Method Not Allowed", http.StatusMethodNotAllowed)
		return
	}

	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	w.WriteHeader(http.StatusOK)
	hits := cfg.fileserverHits.Load()

	const tpl = `<html>
	<body>
		<h1>Welcome, Chirpy Admin</h1>
		<p>Chirpy has been visited %d times!</p>
	</body>
</html>`

	_, _ = fmt.Fprintf(w, tpl, hits)
}

func (cfg *apiConfig) resetHandler(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		w.Header().Set("Allow", "POST")
		http.Error(w, "Method Not Allowed", http.StatusMethodNotAllowed)
		return
	}

	cfg.fileserverHits.Store(0)
	w.Header().Set("Content-Type", "text/plain; charset=utf-8")
	w.WriteHeader(http.StatusOK)
	_, _ = w.Write([]byte("OK"))
}

func (cfg *apiConfig) validateChirpHandler(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		w.Header().Set("Allow", "POST")
		w.Header().Set("Content-Type", "application/json; charset=utf-8")
		w.WriteHeader(http.StatusMethodNotAllowed)
		_ = json.NewEncoder(w).Encode(map[string]string{"error": "Method Not Allowed"})
		return
	}

	var req struct {
		Body string `json:"body"`
	}

	dec := json.NewDecoder(r.Body)
	if err := dec.Decode(&req); err != nil {
		w.Header().Set("Content-Type", "application/json; charset=utf-8")
		w.WriteHeader(http.StatusBadRequest)
		_ = json.NewEncoder(w).Encode(map[string]string{"error": "Invalid JSON"})
		return
	}

	const maxLen = 280
	if len(req.Body) > maxLen {
		w.Header().Set("Content-Type", "application/json; charset=utf-8")
		w.WriteHeader(http.StatusBadRequest)
		_ = json.NewEncoder(w).Encode(map[string]string{"error": "Chirp is too long"})
		return
	}

	// Replace profane words with "****" only when surrounded by whitespace or string boundaries.
	// Matches are case-insensitive but will NOT match if adjacent to punctuation (e.g. "Sharbert!").
	// Use a trailing capture instead of lookahead (Go regexp doesn't support lookahead).
	re := regexp.MustCompile(`(?i)(^|\s)(kerfuffle|sharbert|fornax)($|\s)`)
	// Replace while preserving surrounding whitespace captured in group 1 and group 3.
	cleaned := re.ReplaceAllString(req.Body, "$1****$3")

	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	w.WriteHeader(http.StatusOK)
	_ = json.NewEncoder(w).Encode(map[string]string{"cleaned_body": cleaned})
}

func main() {
	godotenv.Load()
	dbURL := os.Getenv("DB_URL")
	if dbURL == "" {
		log.Fatal("DB_URL environment variable is not set")
	}

	db, err := sql.Open("postgres", dbURL)
	if err != nil {
		log.Fatalf("failed to open database: %v", err)
	}
	if err := db.Ping(); err != nil {
		log.Fatalf("failed to ping database: %v", err)
	}

	dbQueries := database.New(db)
	mux := http.NewServeMux()
	apiCfg := &apiConfig{queries: dbQueries}
	// readiness endpoint
	mux.HandleFunc("/api/healthz", func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodGet {
			w.Header().Set("Allow", "GET")
			http.Error(w, "Method Not Allowed", http.StatusMethodNotAllowed)
			return
		}

		w.Header().Set("Content-Type", "text/plain; charset=utf-8")
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte("OK"))
	})

	fileServer := http.FileServer(http.Dir("."))
	// serve static files under /app/ (strip the /app/ prefix before handing to FileServer)
	mux.Handle("/app/", apiCfg.middlewareMetricsInc(http.StripPrefix("/app/", fileServer)))

	// admin metrics and reset endpoints
	mux.HandleFunc("/admin/metrics", apiCfg.metricsHandler)
	mux.HandleFunc("/admin/reset", apiCfg.resetHandler)

	// validate chirp endpoint
	mux.HandleFunc("/api/validate_chirp", apiCfg.validateChirpHandler)

	srv := &http.Server{
		Addr:    ":8080",
		Handler: mux,
	}

	log.Printf("starting server on %s", srv.Addr)
	if err := srv.ListenAndServe(); err != nil && err != http.ErrServerClosed {
		log.Fatalf("server error: %v", err)
	}
}
