package main

import (
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"sync/atomic"
)

type apiConfig struct {
	fileserverHits atomic.Int32
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
		Valid bool `json:"valid"`
	}
	defReturnErrs := returnErrs{}
	defReturnVal := returnvalidity{}
	if len(availableChirps.Body) > 140 {
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
		defReturnVal.Valid = true
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
	apiCfg := &apiConfig{}
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

	err := server.ListenAndServe()
	if err != nil {
		log.Fatal(err)
	}

}

func readinessHandler(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "text/plain; charset=utf-8")
	w.WriteHeader(200)
	w.Write([]byte("OK"))
}
