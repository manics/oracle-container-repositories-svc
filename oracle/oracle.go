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

	"context"

	"github.com/manics/oracle-container-repositories-svc/utils"
	"github.com/oracle/oci-go-sdk/identity"
	"github.com/oracle/oci-go-sdk/v65/artifacts"
	"github.com/oracle/oci-go-sdk/v65/common"
	"github.com/oracle/oci-go-sdk/v65/common/auth"
)

var (
	listReposRe = regexp.MustCompile(`^\/repos$`)
	repoRe      = regexp.MustCompile(`^\/repo\/(\S+)$`)
	imageRe     = regexp.MustCompile(`^\/image\/(\S+)$`)
)

type IArtifactsClient interface {
	ListContainerRepositories(ctx context.Context, request artifacts.ListContainerRepositoriesRequest) (response artifacts.ListContainerRepositoriesResponse, err error)

	ListContainerImages(ctx context.Context, request artifacts.ListContainerImagesRequest) (response artifacts.ListContainerImagesResponse, err error)

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
		h.GetRepo(w, r)
		return
	case r.Method == http.MethodGet && imageRe.MatchString(r.URL.Path):
		h.GetImage(w, r)
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

func (c *artifactsHandler) getByName(r *http.Request) (*artifacts.ContainerRepositorySummary, string, error) {
	name, err := utils.RepoGetName(r)
	if err != nil {
		log.Println(err)
		return nil, "", err
	}

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

func (c *artifactsHandler) GetRepo(w http.ResponseWriter, r *http.Request) {
	repo, name, err := c.getByName(r)
	if err != nil {
		log.Println("Error:", err)
		utils.InternalServerError(w, r)
		return
	}

	log.Printf("Getting repo %s", name)

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

func (c *artifactsHandler) GetImage(w http.ResponseWriter, r *http.Request) {
	repoName, tag, err := utils.ImageGetNameAndTag(r)
	if err != nil {
		log.Printf("ERROR: %v\n", err)
		utils.InternalServerError(w, r)
	}
	fullname := fmt.Sprintf("%s:%s", repoName, tag)

	log.Printf("Getting image %s", fullname)

	images, err := c.client.ListContainerImages(context.Background(), artifacts.ListContainerImagesRequest{
		CompartmentId:  &c.compartmentId,
		DisplayName:    &fullname,
		RepositoryName: &repoName,
	})
	if err != nil {
		log.Println("ERROR:", err)
		utils.InternalServerError(w, r)
	}

	if len(images.Items) == 0 {
		log.Printf("Image '%s' not found\n", fullname)
		w.WriteHeader(http.StatusNotFound)
		w.Write([]byte("null"))
		return
	}

	image := images.Items[0]
	log.Printf("Image '%s' found: %s\n", fullname, *image.Id)
	jsonBytes, err := json.Marshal(image)
	if err != nil {
		utils.InternalServerError(w, r)
		return
	}
	w.WriteHeader(http.StatusOK)
	w.Write(jsonBytes)
}

func (c *artifactsHandler) Create(w http.ResponseWriter, r *http.Request) {
	name, err := utils.RepoGetName(r)
	if err != nil {
		log.Println("ERROR:", err)
		utils.InternalServerError(w, r)
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
		log.Printf("ERROR: %v\n", err)
		serviceErr, ok := common.IsServiceError(err)
		if ok && serviceErr.GetCode() == "NAMESPACE_CONFLICT" {
			log.Printf("Repository already exists: %v\n", err)

			repo, name, err := c.getByName(r)
			if err != nil {
				log.Println("Error:", err)
				utils.InternalServerError(w, r)
				return
			}

			if repo == nil {
				log.Printf("NAMESPACE_CONFLICT but repository not found %s: %v", name, err)
				utils.InternalServerError(w, r)
				return
			}

			jsonBytes, err := json.Marshal(repo)
			if err != nil {
				log.Printf("ERROR: %v\n", err)
				utils.InternalServerError(w, r)
				return
			}

			w.WriteHeader(http.StatusOK)
			w.Write(jsonBytes)
			return
		}

		return
	}

	jsonBytes, err := json.Marshal(createResponse.ContainerRepository)
	if err != nil {
		log.Printf("ERROR: %v\n", err)
		utils.InternalServerError(w, r)
		return
	}
	w.WriteHeader(http.StatusOK)
	w.Write(jsonBytes)
}

func (c *artifactsHandler) Delete(w http.ResponseWriter, r *http.Request) {
	repo, name, err := c.getByName(r)
	if err != nil {
		log.Println("ERROR:", err)
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

func Setup(mux *http.ServeMux, args []string) error {
	var cfg common.ConfigurationProvider
	var err error

	if len(args) == 0 {
		// Instance principals (like AWS instance roles)
		// https://github.com/oracle/oci-go-sdk/blob/v65.28.1/example/example_instance_principals_test.go
		cfg, err = auth.InstancePrincipalConfigurationProvider()
		if err != nil {
			log.Printf("failed to load configuration, %v", err)
			return err
		}
	} else if len(args) == 1 {
		// User principals, using configuration file
		cfg_file := args[0]
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

	identityClient, err := identity.NewIdentityClientWithConfigurationProvider(cfg)
	if err != nil {
		return err
	}
	comp, err := identityClient.GetCompartment(context.Background(), identity.GetCompartmentRequest{
		CompartmentId: &tenancyID,
	})
	if err != nil {
		return err
	}
	log.Printf("Compartment: %v\n", comp.Compartment)

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
	mux.Handle("/image/", authorizedH)
	return nil
}
