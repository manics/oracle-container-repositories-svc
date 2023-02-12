package common

import (
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"os"
)

const AUTH_TOKEN_ENV_VAR = "BINDERHUB_AUTH_TOKEN"

// healthHandler is a http.handler that returns the version
type healthHandler struct {
	healthInfo *map[string]string
}

// ServeHTTP implements http.Handler
func (h *healthHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("content-type", "application/json")
	if r.Method == http.MethodGet && r.URL.Path == "/health" {
		jsonBytes, err := json.Marshal(*h.healthInfo)
		if err != nil {
			log.Println("ERROR:", err)
			InternalServerError(w, r, err)
			return
		}
		w.WriteHeader(http.StatusOK)
		_, errw := w.Write(jsonBytes)
		if errw != nil {
			log.Println("ERROR:", errw)
		}
	} else {
		NotFound(w, r)
		return
	}
}

func getAuthToken() (string, error) {
	authToken, found := os.LookupEnv(AUTH_TOKEN_ENV_VAR)
	if !found {
		return "", fmt.Errorf("%s not found, set it to a secret token or '' to disable authentication", AUTH_TOKEN_ENV_VAR)
	}
	return authToken, nil
}

// The main entrypoint for the service
func Run(registryH IRegistryClient, healthInfo *map[string]string, listen string) {
	authToken, err := getAuthToken()
	if err != nil {
		log.Fatalln(err)
	}

	health := healthHandler{
		healthInfo: healthInfo,
	}

	mux := http.NewServeMux()
	mux.Handle("/health", &health)

	CreateServer(mux, registryH, authToken)

	log.Printf("Listening on %v\n", listen)
	errw := http.ListenAndServe(listen, mux)
	if errw != nil {
		log.Fatalln(errw)
	}
}
