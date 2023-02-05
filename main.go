// Based on https://golang.cafe/blog/golang-rest-api-example.html

package main

import (
	"encoding/json"
	"errors"
	"log"
	"net/http"
	"os"

	"github.com/manics/oracle-container-repositories-svc/amazon"
	"github.com/manics/oracle-container-repositories-svc/oracle"
	"github.com/manics/oracle-container-repositories-svc/registry"
)

var (
	// Version is set at build time using the Git repository metadata
	Version string
)

// healthHandler is a http.handler that returns the version
type healthHandler struct {
}

// ServeHTTP implements http.Handler
func (h *healthHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("content-type", "application/json")
	if r.Method == http.MethodGet && r.URL.Path == "/health" {
		versionInfo := map[string]string{
			"version": Version,
		}
		jsonBytes, err := json.Marshal(versionInfo)
		if err != nil {
			log.Println("ERROR:", err)
			registry.InternalServerError(w, r, err)
			return
		}
		w.WriteHeader(http.StatusOK)
		_, errw := w.Write(jsonBytes)
		if errw != nil {
			log.Println("ERROR:", errw)
		}
	} else {
		registry.NotFound(w, r)
		return
	}
}

func getAuthToken() (string, error) {
	authToken, found := os.LookupEnv("AUTH_TOKEN")
	if !found {
		return "", errors.New("AUTH_TOKEN not found, set it to a secret token or '' to disable authentication")
	}
	return authToken, nil
}

// The main entrypoint for the service
func main() {
	log.SetFlags(log.LstdFlags | log.Lshortfile)

	authToken, err := getAuthToken()
	if err != nil {
		log.Fatalln(err)
	}

	if len(os.Args) < 2 {
		log.Fatalf("Usage: %s [amazon|oracle] ...\n", os.Args[0])
	}

	mux := http.NewServeMux()
	mux.Handle("/health", &healthHandler{})

	provider := os.Args[1]

	var registryH registry.IRegistryClient
	switch provider {
	case "amazon":
		registryH, err = amazon.Setup(mux, os.Args[2:])
	case "oracle":
		registryH, err = oracle.Setup(mux, os.Args[2:])
	default:
		log.Fatalf("Unknown provider: %s\n", provider)
	}
	if err != nil {
		log.Fatalln(err)
	}
	registry.CreateServer(mux, registryH, authToken)

	listen := "0.0.0.0:8080"
	log.Printf("Listening on %v\n", listen)
	errw := http.ListenAndServe(listen, mux)
	if errw != nil {
		log.Fatalln(errw)
	}
}
