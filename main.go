// Based on https://golang.cafe/blog/golang-rest-api-example.html
// OCI SDK: https://docs.oracle.com/en-us/iaas/tools/go/65.28.0/

package main

import (
	"encoding/json"
	"errors"
	"fmt"
	"log"
	"net/http"
	"os"
	"regexp"
	"strings"

	"context"

	// "github.com/oracle/oci-go-sdk/identity"
	"github.com/oracle/oci-go-sdk/v65/artifacts"
	"github.com/oracle/oci-go-sdk/v65/common"
)

var (
	listReposRe = regexp.MustCompile(`^\/repos$`)
	repoRe      = regexp.MustCompile(`^\/repo\/(\S+)$`)
	Version     string
)

type artifactsHandler struct {
	compartmentId string
	client        *artifacts.ArtifactsClient
	authToken     string
}

func (h *artifactsHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	authorised := false
	if h.authToken == "" {
		authorised = true
	} else {
		authHeader := r.Header.Get("Authorization")
		if strings.HasPrefix(authHeader, "Bearer ") {
			authToken := strings.TrimPrefix(authHeader, "Bearer ")
			authorised = (authToken == h.authToken)
		}
	}

	if !authorised {
		notAuthorised(w, r)
		return
	}

	w.Header().Set("content-type", "application/json")
	switch {
	case r.Method == http.MethodGet && listReposRe.MatchString(r.URL.Path):
		h.List(w, r)
		return
	case r.Method == http.MethodGet && repoRe.MatchString(r.URL.Path):
		h.Get(w, r)
		return
	case r.Method == http.MethodPost && repoRe.MatchString(r.URL.Path):
		h.Create(w, r)
		return
	case r.Method == http.MethodDelete && repoRe.MatchString(r.URL.Path):
		h.Delete(w, r)
		return
	default:
		notFound(w, r)
		return
	}
}

func (c *artifactsHandler) List(w http.ResponseWriter, r *http.Request) {
	log.Println("Listing repos")
	repos, err := c.client.ListContainerRepositories(context.Background(), artifacts.ListContainerRepositoriesRequest{
		CompartmentId: &c.compartmentId,
	})
	if err != nil {
		fmt.Println("Error:", err)
		return
	}
	jsonBytes, err := json.Marshal(repos.Items)
	if err != nil {
		internalServerError(w, r)
		return
	}
	w.WriteHeader(http.StatusOK)
	w.Write(jsonBytes)
}

func (c *artifactsHandler) getByName(urlPath string) (*artifacts.ContainerRepositorySummary, *string, error) {
	if !strings.HasPrefix(urlPath, "/repo/") {
		err := fmt.Sprintf("Invalid path: %s", urlPath)
		log.Println(err)
		return nil, nil, errors.New(err)
	}
	name := strings.TrimPrefix(urlPath, "/repo/")

	repos, err := c.client.ListContainerRepositories(context.Background(), artifacts.ListContainerRepositoriesRequest{
		CompartmentId: &c.compartmentId,
		DisplayName:   &name,
	})
	if err != nil {
		log.Println("Error:", err)
		return nil, nil, err
	}
	if len(repos.Items) == 0 {
		log.Printf("Repo '%s' not found\n", name)
		return nil, &name, nil
	} else {
		log.Printf("Repo '%s' found: %s\n", name, *repos.Items[0].Id)
		return &repos.Items[0], &name, nil
	}
}

func (c *artifactsHandler) Get(w http.ResponseWriter, r *http.Request) {
	repo, _, err := c.getByName(r.URL.Path)
	if err != nil {
		log.Println("Error:", err)
		internalServerError(w, r)
		return
	}

	if repo == nil {
		w.WriteHeader(http.StatusNotFound)
		w.Write([]byte("null"))
	} else {
		jsonBytes, err := json.Marshal(repo)
		if err != nil {
			internalServerError(w, r)
			return
		}
		w.WriteHeader(http.StatusOK)
		w.Write(jsonBytes)
	}
}

func (c *artifactsHandler) Create(w http.ResponseWriter, r *http.Request) {
	repo, name, err := c.getByName(r.URL.Path)
	if err != nil {
		log.Println("Error:", err)
		internalServerError(w, r)
		return
	}

	if repo != nil {
		jsonBytes, err := json.Marshal(repo)
		if err != nil {
			internalServerError(w, r)
			return
		}
		w.WriteHeader(http.StatusOK)
		w.Write(jsonBytes)
		return
	}

	log.Println("Creating repo", *name)

	createResponse, err := c.client.CreateContainerRepository(context.Background(), artifacts.CreateContainerRepositoryRequest{
		CreateContainerRepositoryDetails: artifacts.CreateContainerRepositoryDetails{
			CompartmentId: &c.compartmentId,
			DisplayName:   name,
		},
	})

	if err != nil {
		internalServerError(w, r)
		return
	}
	jsonBytes, err := json.Marshal(createResponse.ContainerRepository)
	if err != nil {
		internalServerError(w, r)
		return
	}
	w.WriteHeader(http.StatusOK)
	w.Write(jsonBytes)
}

func (c *artifactsHandler) Delete(w http.ResponseWriter, r *http.Request) {
	repo, name, err := c.getByName(r.URL.Path)
	if err != nil {
		log.Println("Error:", err)
		internalServerError(w, r)
		return
	}

	if repo != nil {
		log.Println("Deleting repo", *name)

		_, err := c.client.DeleteContainerRepository(context.Background(), artifacts.DeleteContainerRepositoryRequest{
			RepositoryId: repo.Id,
		})
		if err != nil {
			internalServerError(w, r)
			return
		}
		w.WriteHeader(http.StatusOK)
		return
	}
}

func internalServerError(w http.ResponseWriter, r *http.Request) {
	log.Println("ERROR: %r", r)
	w.WriteHeader(http.StatusInternalServerError)
	w.Write([]byte("\"internal server error\""))
}

func notFound(w http.ResponseWriter, r *http.Request) {
	fmt.Println("notFound %r", r)
	w.WriteHeader(http.StatusNotFound)
	w.Write([]byte("not found"))
}

func notAuthorised(w http.ResponseWriter, r *http.Request) {
	fmt.Println("notAuthorised %r", r)
	w.WriteHeader(http.StatusForbidden)
	w.Write([]byte("not authorised"))
}

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
			internalServerError(w, r)
			return
		}
		w.WriteHeader(http.StatusOK)
		w.Write(jsonBytes)
	} else {
		notFound(w, r)
		return
	}
}

func main() {
	if len(os.Args[1:]) != 1 {
		log.Fatalf("Usage: %s oci-config-file\n", os.Args[0])
	}
	cfg_file := os.Args[1]
	cfg, err := common.ConfigurationProviderFromFile(cfg_file, "")
	if err != nil {
		log.Fatal(err)
	}

	// The OCID of the tenancy containing the compartment.
	tenancyID, err := cfg.TenancyOCID()
	if err != nil {
		log.Fatalln(err)
	}

	artifactsClient, err := artifacts.NewArtifactsClientWithConfigurationProvider(cfg)
	if err != nil {
		log.Fatal(err)
	}

	compartmentId := os.Getenv("OCI_COMPARTMENT_ID")
	if compartmentId == "" {
		compartmentId = tenancyID
	}
	log.Println("Compartment ID:", compartmentId)

	authToken, found := os.LookupEnv("AUTH_TOKEN")
	if !found {
		log.Fatalln("AUTH_TOKEN not found, set it to a secret token or '' to disable authentication")
	}
	log.Println("Auth token:", authToken)

	mux := http.NewServeMux()
	artifactsH := &artifactsHandler{
		compartmentId: compartmentId,
		client:        &artifactsClient,
		authToken:     authToken,
	}

	mux.Handle("/repos", artifactsH)
	mux.Handle("/repo/", artifactsH)
	mux.Handle("/health", &healthHandler{})

	http.ListenAndServe("0.0.0.0:8080", mux)
}
