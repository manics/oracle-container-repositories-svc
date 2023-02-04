package utils

import (
	"net/http"
	"regexp"
)

var (
	listReposRe = regexp.MustCompile(`^\/repos$`)
	repoRe      = regexp.MustCompile(`^\/repo\/(\S+)$`)
	imageRe     = regexp.MustCompile(`^\/image\/(\S+)$`)
	tokenRe     = regexp.MustCompile(`^\/token$`)
)

type IRegistryClient interface {
	ListRepositories(w http.ResponseWriter, r *http.Request)
	GetRepository(w http.ResponseWriter, r *http.Request)
	GetImage(w http.ResponseWriter, r *http.Request)
	CreateRepository(w http.ResponseWriter, r *http.Request)
	DeleteRepository(w http.ResponseWriter, r *http.Request)
	GetToken(w http.ResponseWriter, r *http.Request)
}

type RegistryServer struct {
	Client IRegistryClient
}

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

func CreateServer(mux *http.ServeMux, registryH IRegistryClient, auth bool) {
	serverH := &RegistryServer{
		Client: registryH,
	}
	var h http.Handler
	if auth {
		h = CheckAuthorised(serverH)
	} else {
		h = serverH
	}

	mux.Handle("/repos", h)
	mux.Handle("/repo/", h)
	mux.Handle("/image/", h)
	mux.Handle("/token", h)
}
