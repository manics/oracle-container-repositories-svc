// Based on https://golang.cafe/blog/golang-rest-api-example.html

package main

import (
	"encoding/json"
	"log"
	"net/http"
	"os"

	"github.com/manics/oracle-container-repositories-svc/amazon"
	"github.com/manics/oracle-container-repositories-svc/oracle"
	"github.com/manics/oracle-container-repositories-svc/utils"
)

var (
	Version string
)

type healthHandler struct {
}

func (h *healthHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("content-type", "application/json")
	if r.Method == http.MethodGet && r.URL.Path == "/health" {
		versionInfo := map[string]string{
			"version": Version,
		}
		jsonBytes, err := json.Marshal(versionInfo)
		if err != nil {
			utils.InternalServerError(w, r)
			return
		}
		w.WriteHeader(http.StatusOK)
		w.Write(jsonBytes)
	} else {
		utils.NotFound(w, r)
		return
	}
}

func main() {
	_, found := os.LookupEnv("AUTH_TOKEN")
	if !found {
		log.Fatalln("AUTH_TOKEN not found, set it to a secret token or '' to disable authentication")
	}

	if len(os.Args) < 2 {
		log.Fatalf("Usage: %s [amazon|oracle] ...\n", os.Args[0])
	}

	mux := http.NewServeMux()
	mux.Handle("/health", &healthHandler{})

	provider := os.Args[1]
	var err error
	switch provider {
	case "amazon":
		err = amazon.Setup(mux, os.Args[2:])
	case "oracle":
		err = oracle.Setup(mux, os.Args[2:])
	default:
		log.Fatalf("Unknown provider: %s\n", provider)
	}
	if err != nil {
		log.Fatalln(err)
	}

	listen := "0.0.0.0:8080"
	log.Printf("Listening on %v\n", listen)
	http.ListenAndServe(listen, mux)
}
