package main

import (
	"database/sql"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"os"
	"strings"
	"sync/atomic"

	"github.com/LoviceSunuwar/GOlangHTTPServer/internal/database"
	"github.com/joho/godotenv"
	_ "github.com/lib/pq"
)

type apiConfig struct {
	fileserverHits atomic.Int32
	DB             *database.Queries
}

func (cfg *apiConfig) middlewareMetricsInc(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		cfg.fileserverHits.Add(1)
		next.ServeHTTP(w, r)

	})
}

func (cfg *apiConfig) metricsHandler(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	w.WriteHeader(http.StatusOK)
	fmt.Fprintf(w, `<html>
  <body>
    <h1>Welcome, Chirpy Admin</h1>
    <p>Chirpy has been visited %d times!</p>
  </body>
</html>`, cfg.fileserverHits.Load())
}

func (cfg *apiConfig) resetHandler(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "text/plain; charset=utf-8")
	cfg.fileserverHits.Store(0)
	w.WriteHeader(http.StatusOK)
	w.Write([]byte("OK"))
}

func (cfg *apiConfig) validateChirp(w http.ResponseWriter, r *http.Request) {
	type recivedChirps struct {
		Body string `json:"body"`
	}

	chirpDecoder := json.NewDecoder(r.Body)
	availableChirps := recivedChirps{}
	err := chirpDecoder.Decode(&availableChirps)
	if err != nil {
		log.Printf("Error decoding recvied Chirps : %v", err)
		w.WriteHeader(500)
		return
	}

	type returnErrs struct {
		Error string `json:"error"`
	}

	type returnvalidity struct {
		CleanedBody string `json:"cleaned_body"`
	}
	defReturnErrs := returnErrs{}
	defReturnVal := returnvalidity{}
	lowerBody := availableChirps.Body
	profain := []string{"kerfuffle", "sharbert", "fornax"}
	if len(lowerBody) > 140 {
		defReturnErrs.Error = "Chirp is too long"
		w.WriteHeader(400)
		data, err := json.Marshal(defReturnErrs)
		if err != nil {
			log.Printf("Error marshalling JSON %v", err)
			w.WriteHeader(500)
			return
		}
		w.Header().Set("Content-Type", "application/json")
		w.Write(data)
	} else {
		cleanedWords := []string{}
		sepBodyResp := strings.Split(lowerBody, " ")
		for _, singleWord := range sepBodyResp {
			isProfane := false
			for _, profainWords := range profain {
				if strings.ToLower(profainWords) == strings.ToLower(singleWord) {
					isProfane = true
					break
				}
			}
			if isProfane {
				cleanedWords = append(cleanedWords, "****")
			} else {
				cleanedWords = append(cleanedWords, singleWord)
			}

		}
		defReturnVal.CleanedBody = strings.Join(cleanedWords, " ")
		w.WriteHeader(200)
		data, err := json.Marshal(defReturnVal)
		if err != nil {
			log.Printf("Error marshalling JSON %v", err)
			w.WriteHeader(500)
			return
		}
		w.Header().Set("Content-Type", "application/json")
		w.Write(data)
	}

}

func main() {
	godotenv.Load()
	dbURL := os.Getenv("DB_URL")
	db, err := sql.Open("postgres", dbURL)
	if err != nil {
		log.Fatalf(err.Error())
	}
	dbQueries := database.New(db)
	apiCfg := &apiConfig{
		DB: dbQueries,
	}
	serveMux := http.NewServeMux()
	serveMux.HandleFunc("GET /api/healthz", readinessHandler)
	serveMux.Handle("/app/", apiCfg.middlewareMetricsInc(http.StripPrefix("/app", http.FileServer(http.Dir(".")))))
	serveMux.Handle("GET /admin/metrics", http.HandlerFunc(apiCfg.metricsHandler))
	serveMux.Handle("POST /admin/reset", http.HandlerFunc(apiCfg.resetHandler))
	serveMux.Handle("POST /api/validate_chirp", http.HandlerFunc(apiCfg.validateChirp))
	server := http.Server{
		Addr:    ":8080",
		Handler: serveMux,
	}

	err = server.ListenAndServe()
	if err != nil {
		log.Fatal(err)
	}

}

func readinessHandler(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "text/plain; charset=utf-8")
	w.WriteHeader(200)
	w.Write([]byte("OK"))
}
