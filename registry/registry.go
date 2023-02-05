// Package registry contains common types and functions used by implementations
package registry

import (
	"encoding/json"
	"errors"
	"fmt"
	"log"
	"net/http"
	"regexp"
	"strings"
	"time"
)

// RegistryToken is an object containing a Token that can be used to login to a registry
// and an Expires time when the token will expire
type RegistryToken struct {
	Token   string    `json:"token"`
	Expires time.Time `json:"expires"`
}

// CheckAuthorised wraps originalHandler to check for a valid Authorization header
// and returns a http.Handler
func CheckAuthorised(originalHandler http.Handler, authToken string) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
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

// InternalServerError is a handler that returns a 500 HTTP error
func InternalServerError(w http.ResponseWriter, r *http.Request, errorResponse error) {
	errObj := map[string]string{
		"error": errorResponse.Error(),
	}
	jsonBytes, err := json.Marshal(errObj)
	if err != nil {
		log.Println("ERROR:", err)
		jsonBytes = []byte(`{"error": "internal server error"}` + "\n")
	}
	jsonBytes = append(jsonBytes, byte('\n'))
	w.WriteHeader(http.StatusInternalServerError)
	w.Write(jsonBytes)
}

// NotFound is a handler that returns a 404 HTTP error
func NotFound(w http.ResponseWriter, r *http.Request) {
	w.WriteHeader(http.StatusNotFound)
	w.Write([]byte("null\n"))
}

// NotAuthorised is a handler that returns a 403 HTTP error
func NotAuthorised(w http.ResponseWriter, r *http.Request) {
	fmt.Println("NotAuthorised %r", r)
	w.WriteHeader(http.StatusForbidden)
	w.Write([]byte(`{"error": "not authorised"}` + "\n"))
}

// RepoGetName extracts the repository name from the request path
func RepoGetName(r *http.Request) (string, error) {
	if !strings.HasPrefix(r.URL.Path, "/repo/") {
		err := fmt.Sprintf("Invalid path: %s", r.URL.Path)
		return "", errors.New(err)
	}
	name := strings.TrimPrefix(r.URL.Path, "/repo/")
	return name, nil
}

// ImageGetNameAndTag extracts the repository name and tag from the request path
func ImageGetNameAndTag(r *http.Request) (string, string, error) {
	if !strings.HasPrefix(r.URL.Path, "/image/") {
		err := fmt.Sprintf("Invalid path: %s", r.URL.Path)
		return "", "", errors.New(err)
	}

	fullname := strings.TrimPrefix(r.URL.Path, "/image/")
	repoName := fullname
	tag := "latest"
	sep := strings.LastIndex(fullname, ":")
	if sep > -1 {
		repoName = fullname[:sep]
		tag = fullname[sep+1:]
	}

	if len(tag) == 0 {
		err := fmt.Sprintf("Invalid tag in path: %s", r.URL.Path)
		return "", "", errors.New(err)
	}

	return repoName, tag, nil
}

var (
	listReposRe = regexp.MustCompile(`^\/repos$`)
	repoRe      = regexp.MustCompile(`^\/repo\/(\S+)$`)
	imageRe     = regexp.MustCompile(`^\/image\/(\S+)$`)
	tokenRe     = regexp.MustCompile(`^\/token$`)
)

// IRegistryClient is an interface that all registry helpers must implement
type IRegistryClient interface {
	ListRepositories(w http.ResponseWriter, r *http.Request)
	GetRepository(w http.ResponseWriter, r *http.Request)
	GetImage(w http.ResponseWriter, r *http.Request)
	CreateRepository(w http.ResponseWriter, r *http.Request)
	DeleteRepository(w http.ResponseWriter, r *http.Request)
	GetToken(w http.ResponseWriter, r *http.Request)
}

// RegistryServer is http.handler that passes requests to the registry helper implementation
type RegistryServer struct {
	Client IRegistryClient
}

// ServeHTTP passes requests to the registry helper implementation
func (h *RegistryServer) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("content-type", "application/json")
	switch {
	case r.Method == http.MethodGet && listReposRe.MatchString(r.URL.Path):
		h.Client.ListRepositories(w, r)
		return
	case r.Method == http.MethodGet && repoRe.MatchString(r.URL.Path):
		h.Client.GetRepository(w, r)
		return
	case r.Method == http.MethodGet && imageRe.MatchString(r.URL.Path):
		h.Client.GetImage(w, r)
		return
	case r.Method == http.MethodPost && repoRe.MatchString(r.URL.Path):
		h.Client.CreateRepository(w, r)
		return
	case r.Method == http.MethodDelete && repoRe.MatchString(r.URL.Path):
		h.Client.DeleteRepository(w, r)
		return
	case r.Method == http.MethodPost && tokenRe.MatchString(r.URL.Path):
		h.Client.GetToken(w, r)
		return
	default:
		NotFound(w, r)
		return
	}
}

// CreateServer configures a new http handler for the registry helper
func CreateServer(mux *http.ServeMux, registryH IRegistryClient, authToken string) {
	serverH := &RegistryServer{
		Client: registryH,
	}
	h := CheckAuthorised(serverH, authToken)

	mux.Handle("/repos", h)
	mux.Handle("/repo/", h)
	mux.Handle("/image/", h)
	mux.Handle("/token", h)
}
