// Based on https://golang.cafe/blog/golang-rest-api-example.html

package main

import (
	"encoding/json"
	"log"
	"net/http"
	"os"

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

	mux := http.NewServeMux()
	mux.Handle("/health", &healthHandler{})
	err := oracle.Setup(mux)
	if err != nil {
		log.Fatalln(err)
	}

	listen := "0.0.0.0:8080"
	log.Printf("Listening on %v\n", listen)
	http.ListenAndServe(listen, mux)
}
