// Based on https://golang.cafe/blog/golang-rest-api-example.html
// OCI SDK: https://docs.oracle.com/en-us/iaas/tools/go/65.28.0/

package oracle

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
	"github.com/manics/oracle-container-repositories-svc/utils"
	"github.com/oracle/oci-go-sdk/v65/artifacts"
	"github.com/oracle/oci-go-sdk/v65/common"
	"github.com/oracle/oci-go-sdk/v65/common/auth"
)

var (
	listReposRe = regexp.MustCompile(`^\/repos$`)
	repoRe      = regexp.MustCompile(`^\/repo\/(\S+)$`)
)

type IArtifactsClient interface {
	ListContainerRepositories(ctx context.Context, request artifacts.ListContainerRepositoriesRequest) (response artifacts.ListContainerRepositoriesResponse, err error)

	CreateContainerRepository(ctx context.Context, request artifacts.CreateContainerRepositoryRequest) (response artifacts.CreateContainerRepositoryResponse, err error)

	DeleteContainerRepository(ctx context.Context, request artifacts.DeleteContainerRepositoryRequest) (response artifacts.DeleteContainerRepositoryResponse, err error)
}

type artifactsHandler struct {
	compartmentId string
	client        IArtifactsClient
}

func (h *artifactsHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
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
		utils.NotFound(w, r)
		return
	}
}

func (c *artifactsHandler) List(w http.ResponseWriter, r *http.Request) {
	log.Println("Listing repos")
	repos, err := c.client.ListContainerRepositories(context.Background(), artifacts.ListContainerRepositoriesRequest{
		CompartmentId: &c.compartmentId,
	})
	if err != nil {
		utils.InternalServerError(w, r)
		return
	}
	jsonBytes, err := json.Marshal(repos.Items)
	if err != nil {
		utils.InternalServerError(w, r)
		return
	}
	w.WriteHeader(http.StatusOK)
	w.Write(jsonBytes)
}

func (c *artifactsHandler) getByName(urlPath string) (*artifacts.ContainerRepositorySummary, string, error) {
	if !strings.HasPrefix(urlPath, "/repo/") {
		err := fmt.Sprintf("Invalid path: %s", urlPath)
		log.Println(err)
		return nil, "", errors.New(err)
	}
	name := strings.TrimPrefix(urlPath, "/repo/")

	repos, err := c.client.ListContainerRepositories(context.Background(), artifacts.ListContainerRepositoriesRequest{
		CompartmentId: &c.compartmentId,
		DisplayName:   &name,
	})
	if err != nil {
		log.Println("Error:", err)
		return nil, "", err
	}
	if len(repos.Items) == 0 {
		log.Printf("Repo '%s' not found\n", name)
		return nil, name, nil
	} else {
		log.Printf("Repo '%s' found: %s\n", name, *repos.Items[0].Id)
		return &repos.Items[0], name, nil
	}
}

func (c *artifactsHandler) Get(w http.ResponseWriter, r *http.Request) {
	repo, _, err := c.getByName(r.URL.Path)
	if err != nil {
		log.Println("Error:", err)
		utils.InternalServerError(w, r)
		return
	}

	if repo == nil {
		w.WriteHeader(http.StatusNotFound)
		w.Write([]byte("null"))
	} else {
		jsonBytes, err := json.Marshal(repo)
		if err != nil {
			utils.InternalServerError(w, r)
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
		utils.InternalServerError(w, r)
		return
	}

	if repo != nil {
		jsonBytes, err := json.Marshal(repo)
		if err != nil {
			utils.InternalServerError(w, r)
			return
		}
		w.WriteHeader(http.StatusOK)
		w.Write(jsonBytes)
		return
	}

	log.Println("Creating repo", name)

	createResponse, err := c.client.CreateContainerRepository(context.Background(), artifacts.CreateContainerRepositoryRequest{
		CreateContainerRepositoryDetails: artifacts.CreateContainerRepositoryDetails{
			CompartmentId: &c.compartmentId,
			DisplayName:   &name,
		},
	})

	if err != nil {
		utils.InternalServerError(w, r)
		return
	}
	jsonBytes, err := json.Marshal(createResponse.ContainerRepository)
	if err != nil {
		utils.InternalServerError(w, r)
		return
	}
	w.WriteHeader(http.StatusOK)
	w.Write(jsonBytes)
}

func (c *artifactsHandler) Delete(w http.ResponseWriter, r *http.Request) {
	repo, name, err := c.getByName(r.URL.Path)
	if err != nil {
		log.Println("Error:", err)
		utils.InternalServerError(w, r)
		return
	}

	if repo != nil {
		log.Println("Deleting repo", name)

		_, err := c.client.DeleteContainerRepository(context.Background(), artifacts.DeleteContainerRepositoryRequest{
			RepositoryId: repo.Id,
		})
		if err != nil {
			utils.InternalServerError(w, r)
			return
		}
		w.WriteHeader(http.StatusOK)
		return
	}
}

func Setup(mux *http.ServeMux) error {
	var cfg common.ConfigurationProvider
	var err error

	if len(os.Args[1:]) == 0 {
		// Instance principals (like AWS instance roles)
		// https://github.com/oracle/oci-go-sdk/blob/v65.28.1/example/example_instance_principals_test.go
		cfg, err = auth.InstancePrincipalConfigurationProvider()
		if err != nil {
			log.Printf("failed to load configuration, %v", err)
			return err
		}
	} else if len(os.Args[1:]) == 1 {
		// User principals, using configuration file
		cfg_file := os.Args[1]
		cfg, err = common.ConfigurationProviderFromFile(cfg_file, "")
		if err != nil {
			log.Printf("failed to load configuration, %v", err)
			return err
		}
	} else {
		return errors.New("arguments: [oci-config-file]")
	}

	// The OCID of the tenancy containing the compartment.
	tenancyID, err := cfg.TenancyOCID()
	if err != nil {
		return err
	}

	artifactsClient, err := artifacts.NewArtifactsClientWithConfigurationProvider(cfg)
	if err != nil {
		return err
	}

	compartmentId := os.Getenv("OCI_COMPARTMENT_ID")
	if compartmentId == "" {
		compartmentId = tenancyID
	}
	log.Println("Compartment ID:", compartmentId)

	artifactsH := &artifactsHandler{
		compartmentId: compartmentId,
		client:        &artifactsClient,
	}
	authorizedH := utils.CheckAuthorised(artifactsH)

	mux.Handle("/repos", authorizedH)
	mux.Handle("/repo/", authorizedH)
	return nil
}
