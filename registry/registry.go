package registry

import (
	"errors"
	"fmt"
	"log"
	"net/http"
	"os"
	"regexp"
	"strings"
	"time"
)

type RegistryToken struct {
	Token   string    `json:"token"`
	Expires time.Time `json:"expires"`
}

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
	w.Write([]byte("internal server error\n"))
}

func ClientError(w http.ResponseWriter, r *http.Request, err string) {
	log.Println("ERROR: %r", r)
	w.WriteHeader(http.StatusBadRequest)
	w.Write([]byte(fmt.Sprintf("client error: %v\n", err)))
}

func NotFound(w http.ResponseWriter, r *http.Request) {
	fmt.Println("NotFound %r", r)
	w.WriteHeader(http.StatusNotFound)
	w.Write([]byte("not found\n"))
}

func NotAuthorised(w http.ResponseWriter, r *http.Request) {
	fmt.Println("NotAuthorised %r", r)
	w.WriteHeader(http.StatusForbidden)
	w.Write([]byte("not authorised\n"))
}

func RepoGetName(r *http.Request) (string, error) {
	if !strings.HasPrefix(r.URL.Path, "/repo/") {
		err := fmt.Sprintf("Invalid path: %s", r.URL.Path)
		return "", errors.New(err)
	}
	name := strings.TrimPrefix(r.URL.Path, "/repo/")
	return name, nil
}

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
