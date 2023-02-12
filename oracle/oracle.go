// Based on https://golang.cafe/blog/golang-rest-api-example.html
// OCI SDK: https://docs.oracle.com/en-us/iaas/tools/go/65.28.0/

package oracle

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"log"
	"net/http"
	"os"
	"strings"

	"github.com/manics/binderhub-container-registry-helper/registry"

	"github.com/oracle/oci-go-sdk/identity"
	"github.com/oracle/oci-go-sdk/v65/artifacts"
	"github.com/oracle/oci-go-sdk/v65/common"
	"github.com/oracle/oci-go-sdk/v65/common/auth"
	"github.com/oracle/oci-go-sdk/v65/objectstorage"
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
	namespace     string
}

func (c *artifactsHandler) ListRepositories(w http.ResponseWriter, r *http.Request) {
	log.Println("Listing repos")
	repos, err := c.client.ListContainerRepositories(context.Background(), artifacts.ListContainerRepositoriesRequest{
		CompartmentId: &c.compartmentId,
	})
	if err != nil {
		log.Println("ERROR:", err)
		registry.InternalServerError(w, r, err)
		return
	}
	jsonBytes, err := json.Marshal(repos.Items)
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
}

func (c *artifactsHandler) dropNamespace(namespacedRepository string) (string, error) {
	// OCI has a namespace prefix which isn't part of the repository name:
	// OCIR_NAMESPACE/OCIR_REPOSITORY_NAME:TAG
	namespace, reponame, found := strings.Cut(namespacedRepository, "/")
	if !found {
		return "", fmt.Errorf("invalid namespace/repository: %s", namespacedRepository)
	}
	if namespace != c.namespace {
		return "", fmt.Errorf("namespace does not match tenancy namespace %s: %s", c.namespace, namespace)
	}
	return reponame, nil
}

func (c *artifactsHandler) getByName(r *http.Request) (*artifacts.ContainerRepositorySummary, string, error) {
	namespacedRepository, err := registry.RepoGetName(r)
	if err != nil {
		return nil, "", err
	}
	name, err := c.dropNamespace(namespacedRepository)
	if err != nil {
		return nil, "", err
	}

	repos, err := c.client.ListContainerRepositories(context.Background(), artifacts.ListContainerRepositoriesRequest{
		CompartmentId: &c.compartmentId,
		DisplayName:   &name,
	})
	if err != nil {
		log.Println("ERROR:", err)
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

func (c *artifactsHandler) GetRepository(w http.ResponseWriter, r *http.Request) {
	repo, name, err := c.getByName(r)
	if err != nil {
		log.Println("ERROR:", err)
		registry.InternalServerError(w, r, err)
		return
	}

	log.Printf("Getting repo %s", name)

	if repo == nil {
		registry.NotFound(w, r)
		return
	} else {
		jsonBytes, err := json.Marshal(repo)
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
	}
}

func (c *artifactsHandler) GetImage(w http.ResponseWriter, r *http.Request) {
	namespacedRepository, tag, err := registry.ImageGetNameAndTag(r)
	if err != nil {
		log.Println("ERROR:", err)
		registry.InternalServerError(w, r, err)
	}
	repoName, err := c.dropNamespace(namespacedRepository)
	if err != nil {
		log.Println("ERROR:", err)
		registry.InternalServerError(w, r, err)
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
		registry.InternalServerError(w, r, err)
	}

	if len(images.Items) == 0 {
		log.Printf("Image '%s' not found\n", fullname)
		registry.NotFound(w, r)
		return
	}

	image := images.Items[0]
	log.Printf("Image '%s' found: %s\n", fullname, *image.Id)
	jsonBytes, err := json.Marshal(image)
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
}

func (c *artifactsHandler) CreateRepository(w http.ResponseWriter, r *http.Request) {
	namespacedRepository, err := registry.RepoGetName(r)
	if err != nil {
		log.Println("ERROR:", err)
		registry.InternalServerError(w, r, err)
		return
	}
	name, err := c.dropNamespace(namespacedRepository)
	if err != nil {
		log.Println("ERROR:", err)
		registry.InternalServerError(w, r, err)
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
		log.Println("ERROR:", err)
		serviceErr, ok := common.IsServiceError(err)
		if ok && serviceErr.GetCode() == "NAMESPACE_CONFLICT" {
			log.Printf("Repository already exists: %v\n", err)

			repo, name, err := c.getByName(r)
			if err != nil {
				log.Println("ERROR:", err)
				registry.InternalServerError(w, r, err)
				return
			}

			if repo == nil {
				log.Printf("NAMESPACE_CONFLICT but repository not found %s: %v", name, err)
				registry.InternalServerError(w, r, err)
				return
			}

			jsonBytes, err := json.Marshal(repo)
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
			return
		} else {
			registry.InternalServerError(w, r, err)
			return
		}
	}

	jsonBytes, err := json.Marshal(createResponse.ContainerRepository)
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
}

func (c *artifactsHandler) DeleteRepository(w http.ResponseWriter, r *http.Request) {
	repo, name, err := c.getByName(r)
	if err != nil {
		log.Println("ERROR:", err)
		registry.InternalServerError(w, r, err)
		return
	}

	if repo != nil {
		log.Println("Deleting repo", name)

		_, err := c.client.DeleteContainerRepository(context.Background(), artifacts.DeleteContainerRepositoryRequest{
			RepositoryId: repo.Id,
		})
		if err != nil {
			log.Println("ERROR:", err)
			registry.InternalServerError(w, r, err)
			return
		}
		w.WriteHeader(http.StatusOK)
		return
	}
}

func (c *artifactsHandler) GetToken(w http.ResponseWriter, r *http.Request) {
	log.Println("GetToken not implemented")
	registry.NotFound(w, r)
}

func Setup(args []string) (registry.IRegistryClient, error) {
	var cfg common.ConfigurationProvider
	var err error

	switch len(args) {
	case 0:
		// Instance principals (like AWS instance roles)
		// https://github.com/oracle/oci-go-sdk/blob/v65.28.1/example/example_instance_principals_test.go
		cfg, err = auth.InstancePrincipalConfigurationProvider()
		if err != nil {
			log.Printf("failed to load configuration, %v", err)
			return nil, err
		}
	case 1:
		// User principals, using configuration file
		cfg_file := args[0]
		cfg, err = common.ConfigurationProviderFromFile(cfg_file, "")
		if err != nil {
			log.Printf("failed to load configuration, %v", err)
			return nil, err
		}
	default:
		return nil, errors.New("arguments: [oci-config-file]")
	}

	// The OCID of the tenancy containing the compartment.
	tenancyID, err := cfg.TenancyOCID()
	if err != nil {
		return nil, err
	}

	identityClient, err := identity.NewIdentityClientWithConfigurationProvider(cfg)
	if err != nil {
		return nil, err
	}
	comp, err := identityClient.GetCompartment(context.Background(), identity.GetCompartmentRequest{
		CompartmentId: &tenancyID,
	})
	if err != nil {
		return nil, err
	}
	log.Printf("Compartment: %v\n", comp.Compartment)

	objectClient, err := objectstorage.NewObjectStorageClientWithConfigurationProvider(cfg)
	if err != nil {
		return nil, err
	}
	ns, err := objectClient.GetNamespace(context.Background(), objectstorage.GetNamespaceRequest{})
	if err != nil {
		return nil, err
	}
	namespace := *ns.Value

	artifactsClient, err := artifacts.NewArtifactsClientWithConfigurationProvider(cfg)
	if err != nil {
		return nil, err
	}

	compartmentId := os.Getenv("OCI_COMPARTMENT_ID")
	if compartmentId == "" {
		compartmentId = tenancyID
	}
	log.Println("Compartment ID:", compartmentId)
	log.Println("Namespace:", namespace)

	artifactsH := &artifactsHandler{
		compartmentId: compartmentId,
		client:        &artifactsClient,
		namespace:     namespace,
	}

	return artifactsH, nil
}
