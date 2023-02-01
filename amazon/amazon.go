// https://aws.github.io/aws-sdk-go-v2/docs/
// https://aws.github.io/aws-sdk-go-v2/docs/handling-errors/

package amazon

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

	"github.com/aws/aws-sdk-go-v2/config"
	"github.com/aws/aws-sdk-go-v2/service/ecr"
	"github.com/aws/aws-sdk-go-v2/service/ecr/types"
	"github.com/aws/aws-sdk-go-v2/service/sts"
	"github.com/manics/oracle-container-repositories-svc/utils"
)

var (
	listReposRe = regexp.MustCompile(`^\/repos$`)
	repoRe      = regexp.MustCompile(`^\/repo\/(\S+)$`)
	imageRe     = regexp.MustCompile(`^\/image\/(\S+)$`)
	tokenRe     = regexp.MustCompile(`^\/token$`)
)

type IEcrClient interface {
	DescribeRepositories(ctx context.Context, input *ecr.DescribeRepositoriesInput, optFns ...func(*ecr.Options)) (response *ecr.DescribeRepositoriesOutput, err error)

	DescribeImages(ctx context.Context, input *ecr.DescribeImagesInput, optFns ...func(*ecr.Options)) (response *ecr.DescribeImagesOutput, err error)

	CreateRepository(ctx context.Context, input *ecr.CreateRepositoryInput, optFns ...func(*ecr.Options)) (response *ecr.CreateRepositoryOutput, err error)

	DeleteRepository(ctx context.Context, input *ecr.DeleteRepositoryInput, optFns ...func(*ecr.Options)) (response *ecr.DeleteRepositoryOutput, err error)

	GetAuthorizationToken(ctx context.Context, input *ecr.GetAuthorizationTokenInput, optFns ...func(*ecr.Options)) (response *ecr.GetAuthorizationTokenOutput, err error)
}

type ecrHandler struct {
	registryId string
	client     IEcrClient
}

func (h *ecrHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
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
	case r.Method == http.MethodPost && tokenRe.MatchString(r.URL.Path):
		h.Token(w, r)
		return
	default:
		utils.NotFound(w, r)
		return
	}
}

func (c *ecrHandler) List(w http.ResponseWriter, r *http.Request) {
	log.Println("Listing repos")
	input := ecr.DescribeRepositoriesInput{}
	if c.registryId != "" {
		input.RegistryId = &c.registryId
	}
	repos, err := c.client.DescribeRepositories(context.TODO(), &input)
	if err != nil {
		utils.InternalServerError(w, r)
		return
	}
	jsonBytes, err := json.Marshal(repos.Repositories)
	if err != nil {
		utils.InternalServerError(w, r)
		return
	}
	w.WriteHeader(http.StatusOK)
	w.Write(jsonBytes)
}

func (c *ecrHandler) getName(r *http.Request) (string, error) {
	if !strings.HasPrefix(r.URL.Path, "/repo/") {
		err := fmt.Sprintf("Invalid path: %s", r.URL.Path)
		return "", errors.New(err)
	}
	name := strings.TrimPrefix(r.URL.Path, "/repo/")
	return name, nil
}

func (c *ecrHandler) getRepoByName(name string) (*types.Repository, error) {
	input := ecr.DescribeRepositoriesInput{
		RepositoryNames: []string{name},
	}
	if c.registryId != "" {
		input.RegistryId = &c.registryId
	}
	repos, err := c.client.DescribeRepositories(context.TODO(), &input)
	if err != nil {
		var awsErr *types.RepositoryNotFoundException
		if errors.As(err, &awsErr) {
			log.Printf("Repo '%s' not found\n", name)
			return nil, nil
		}
		log.Println("Error:", err)
		return nil, err
	}
	log.Printf("Repo '%s' found: %s\n", name, *repos.Repositories[0].RepositoryUri)
	return &repos.Repositories[0], nil
}

func (c *ecrHandler) getRepositoryAsJson(r *http.Request) (bool, string, []byte, error) {
	null := []byte("null")

	name, err := c.getName(r)
	if err != nil {
		return false, name, null, err
	}

	repo, err := c.getRepoByName(name)
	if err != nil {
		return false, name, null, err
	}

	if repo == nil {
		return false, name, null, nil
	}

	jsonBytes, err := json.Marshal(repo)
	if err != nil {
		return false, name, null, err
	}
	return true, name, jsonBytes, nil
}

func (c *ecrHandler) GetRepo(w http.ResponseWriter, r *http.Request) {
	found, name, jsonBytes, err := c.getRepositoryAsJson(r)
	if err != nil {
		log.Println("Error:", err)
		utils.InternalServerError(w, r)
		return
	}

	log.Printf("Getting repo %s", name)

	if found {
		w.WriteHeader(http.StatusOK)
	} else {
		w.WriteHeader(http.StatusNotFound)
	}
	w.Write(jsonBytes)
}

func (c *ecrHandler) GetImage(w http.ResponseWriter, r *http.Request) {
	if !strings.HasPrefix(r.URL.Path, "/image/") {
		log.Printf("Invalid path: %s\n", r.URL.Path)
		utils.InternalServerError(w, r)
		return
	}
	fullname := strings.TrimPrefix(r.URL.Path, "/image/")
	repoName := fullname
	tag := "latest"
	sep := strings.LastIndex(fullname, ":")
	if sep > -1 {
		repoName = fullname[:sep]
		tag = fullname[sep+1:]
	}

	input := ecr.DescribeImagesInput{
		RepositoryName: &repoName,
		ImageIds:       []types.ImageIdentifier{{ImageTag: &tag}},
	}
	if c.registryId != "" {
		input.RegistryId = &c.registryId
	}
	images, err := c.client.DescribeImages(context.TODO(), &input)
	if err != nil {
		var awsErrImage *types.ImageNotFoundException
		var awsErrRepo *types.RepositoryNotFoundException
		if errors.As(err, &awsErrImage) || errors.As(err, &awsErrRepo) {
			log.Printf("Repo '%s' not found\n", fullname)
			w.WriteHeader(http.StatusNotFound)
			w.Write([]byte("null"))
			return
		}
		log.Println("Error:", err)
		utils.InternalServerError(w, r)
		return
	}

	image := &images.ImageDetails[0]
	log.Printf("Image '%s' found: %s\n", fullname, image.ImageTags)
	jsonBytes, err := json.Marshal(image)
	if err != nil {
		utils.InternalServerError(w, r)
		return
	}
	w.WriteHeader(http.StatusOK)
	w.Write(jsonBytes)
}

func (c *ecrHandler) Create(w http.ResponseWriter, r *http.Request) {
	name, err := c.getName(r)
	if err != nil {
		log.Println("Error:", err)
		utils.InternalServerError(w, r)
		return
	}

	log.Println("Creating repo", name)

	input := ecr.CreateRepositoryInput{
		RepositoryName: &name,
	}
	if c.registryId != "" {
		input.RegistryId = &c.registryId
	}
	createResponse, err := c.client.CreateRepository(context.TODO(), &input)

	if err != nil {
		// Ignore if it already exists
		var awsErr *types.RepositoryAlreadyExistsException
		if errors.As(err, &awsErr) {
			found, _, jsonBytes, err := c.getRepositoryAsJson(r)
			if err != nil {
				log.Println("Error:", err)
				utils.InternalServerError(w, r)
				return
			}
			if found {
				log.Println("Repo already exists", name)
				w.WriteHeader(http.StatusOK)
				w.Write(jsonBytes)
				return
			}
			log.Printf("RepositoryAlreadyExistsException but repository not found %s: %v", name, awsErr)
			utils.InternalServerError(w, r)
			return
		}

		log.Println("Error:", err)
		utils.InternalServerError(w, r)
		return
	}

	jsonBytes, err := json.Marshal(createResponse.Repository)
	if err != nil {
		log.Println("Error:", err)
		utils.InternalServerError(w, r)
		return
	}
	w.WriteHeader(http.StatusOK)
	w.Write(jsonBytes)
}

func (c *ecrHandler) Delete(w http.ResponseWriter, r *http.Request) {
	name, err := c.getName(r)
	if err != nil {
		log.Println("Error:", err)
		utils.InternalServerError(w, r)
		return
	}

	log.Println("Deleting repo", name)

	input := ecr.DeleteRepositoryInput{
		RepositoryName: &name,
	}
	if c.registryId != "" {
		input.RegistryId = &c.registryId
	}
	_, err = c.client.DeleteRepository(context.TODO(), &input)

	if err != nil {
		// Ignore if it didn't exist
		var awsErr *types.RepositoryNotFoundException
		if errors.As(err, &awsErr) {
			log.Println("Repo not found", name)
			w.WriteHeader(http.StatusOK)
			return
		}
		utils.InternalServerError(w, r)
		return
	}

	w.WriteHeader(http.StatusOK)
}

func (c *ecrHandler) Token(w http.ResponseWriter, r *http.Request) {
	token, err := c.client.GetAuthorizationToken(context.TODO(), &ecr.GetAuthorizationTokenInput{})
	if err != nil {
		log.Println("Error:", err)
		utils.InternalServerError(w, r)
		return
	}
	if len(token.AuthorizationData) != 1 {
		log.Println("Error: expected 1 token, got", len(token.AuthorizationData))
		utils.InternalServerError(w, r)
		return
	}

	resp := &utils.RegistryToken{
		Token:   *token.AuthorizationData[0].AuthorizationToken,
		Expires: *token.AuthorizationData[0].ExpiresAt,
	}
	jsonBytes, err := json.Marshal(resp)
	if err != nil {
		log.Println("Error:", err)
		utils.InternalServerError(w, r)
		return
	}
	w.WriteHeader(http.StatusOK)
	w.Write(jsonBytes)
}

func Setup(mux *http.ServeMux, args []string) error {
	if len(args) != 0 {
		return errors.New("no arguments expected")
	}
	// Automatically looks for a usable configuration
	cfg, err := config.LoadDefaultConfig(context.TODO())
	if err != nil {
		log.Printf("failed to load configuration, %v", err)
		return err
	}

	stsClient := sts.NewFromConfig(cfg)
	identity, err := stsClient.GetCallerIdentity(context.TODO(), &sts.GetCallerIdentityInput{})
	if err != nil {
		log.Printf("failed to get identity, %v", err)
		return err
	}
	log.Printf("Identity: %v", *identity.Arn)

	ecrClient := ecr.NewFromConfig(cfg)

	registryId := os.Getenv("AWS_REGISTRY_ID")
	log.Println("Registry ID:", registryId)

	ecrH := &ecrHandler{
		registryId: registryId,
		client:     ecrClient,
	}
	authorizedH := utils.CheckAuthorised(ecrH)

	mux.Handle("/repos", authorizedH)
	mux.Handle("/repo/", authorizedH)
	mux.Handle("/image/", authorizedH)
	mux.Handle("/token", authorizedH)
	return nil
}
