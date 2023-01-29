package utils

import (
	"fmt"
	"log"
	"net/http"
	"os"
	"strings"
)

func CheckAuthorised(originalHandler http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		authToken, found := os.LookupEnv("AUTH_TOKEN")
		if !found {
			log.Fatalln("AUTH_TOKEN not found, set it to a secret token or '' to disable authentication")
		}

		authorised := false
		if authToken == "" {
			authorised = true
		} else {
			authHeader := r.Header.Get("Authorization")
			if strings.HasPrefix(authHeader, "Bearer ") {
				bearerToken := strings.TrimPrefix(authHeader, "Bearer ")
				authorised = (authToken == bearerToken)
			}
		}

		if !authorised {
			NotAuthorised(w, r)
			return
		}
		originalHandler.ServeHTTP(w, r)
	})
}

func InternalServerError(w http.ResponseWriter, r *http.Request) {
	log.Println("ERROR: %r", r)
	w.WriteHeader(http.StatusInternalServerError)
	w.Write([]byte("\"internal server error\""))
}

func NotFound(w http.ResponseWriter, r *http.Request) {
	fmt.Println("NotFound %r", r)
	w.WriteHeader(http.StatusNotFound)
	w.Write([]byte("not found"))
}

func NotAuthorised(w http.ResponseWriter, r *http.Request) {
	fmt.Println("NotAuthorised %r", r)
	w.WriteHeader(http.StatusForbidden)
	w.Write([]byte("not authorised"))
}
