// https://aws.github.io/aws-sdk-go-v2/docs/
// https://aws.github.io/aws-sdk-go-v2/docs/handling-errors/

package amazon

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"log"
	"net/http"
	"os"
	"strconv"

	"github.com/manics/binderhub-container-registry-helper/registry"

	"github.com/aws/aws-sdk-go-v2/config"
	"github.com/aws/aws-sdk-go-v2/service/ecr"
	"github.com/aws/aws-sdk-go-v2/service/ecr/types"
	"github.com/aws/aws-sdk-go-v2/service/sts"
)

type IEcrClient interface {
	DescribeRepositories(ctx context.Context, input *ecr.DescribeRepositoriesInput, optFns ...func(*ecr.Options)) (response *ecr.DescribeRepositoriesOutput, err error)

	DescribeImages(ctx context.Context, input *ecr.DescribeImagesInput, optFns ...func(*ecr.Options)) (response *ecr.DescribeImagesOutput, err error)

	CreateRepository(ctx context.Context, input *ecr.CreateRepositoryInput, optFns ...func(*ecr.Options)) (response *ecr.CreateRepositoryOutput, err error)

	PutLifecyclePolicy(ctx context.Context, input *ecr.PutLifecyclePolicyInput, optFns ...func(*ecr.Options)) (response *ecr.PutLifecyclePolicyOutput, err error)

	DeleteRepository(ctx context.Context, input *ecr.DeleteRepositoryInput, optFns ...func(*ecr.Options)) (response *ecr.DeleteRepositoryOutput, err error)

	DeleteLifecyclePolicy(ctx context.Context, input *ecr.DeleteLifecyclePolicyInput, optFns ...func(*ecr.Options)) (response *ecr.DeleteLifecyclePolicyOutput, err error)

	GetAuthorizationToken(ctx context.Context, input *ecr.GetAuthorizationTokenInput, optFns ...func(*ecr.Options)) (response *ecr.GetAuthorizationTokenOutput, err error)
}

type ecrHandler struct {
	registryId           string
	expiresAfterPushDays int
	expiresAfterPullDays int
	client               IEcrClient
}

func (c *ecrHandler) ListRepositories(w http.ResponseWriter, r *http.Request) {
	log.Println("Listing repos")
	input := ecr.DescribeRepositoriesInput{}
	if c.registryId != "" {
		input.RegistryId = &c.registryId
	}
	repos, err := c.client.DescribeRepositories(context.TODO(), &input)
	if err != nil {
		log.Println("ERROR:", err)
		registry.InternalServerError(w, r, err)
		return
	}
	jsonBytes, err := json.Marshal(repos.Repositories)
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
		log.Println("ERROR:", err)
		return nil, err
	}
	log.Printf("Repo '%s' found: %s\n", name, *repos.Repositories[0].RepositoryUri)
	return &repos.Repositories[0], nil
}

func (c *ecrHandler) getRepositoryAsJson(r *http.Request) (bool, string, []byte, error) {
	null := []byte("null")

	name, err := registry.RepoGetName(r)
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

func (c *ecrHandler) GetRepository(w http.ResponseWriter, r *http.Request) {
	found, name, jsonBytes, err := c.getRepositoryAsJson(r)
	if err != nil {
		log.Println("ERROR:", err)
		registry.InternalServerError(w, r, err)
		return
	}

	log.Printf("Getting repo %s", name)

	if found {
		w.WriteHeader(http.StatusOK)
	} else {
		w.WriteHeader(http.StatusNotFound)
	}
	_, errw := w.Write(jsonBytes)
	if errw != nil {
		log.Println("ERROR:", errw)
	}
}

func (c *ecrHandler) GetImage(w http.ResponseWriter, r *http.Request) {
	repoName, tag, err := registry.ImageGetNameAndTag(r)
	if err != nil {
		log.Println("ERROR:", err)
		registry.InternalServerError(w, r, err)
	}
	fullname := fmt.Sprintf("%s:%s", repoName, tag)

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
			registry.NotFound(w, r)
			return
		}
		log.Println("ERROR:", err)
		registry.InternalServerError(w, r, err)
		return
	}

	image := &images.ImageDetails[0]
	log.Printf("Image '%s' found: %s\n", fullname, image.ImageTags)
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

func lifecyclePolicy(priority int, countType string, countNumber int) string {
	policy := map[string]interface{}{
		"rulePriority": priority,
		"description":  fmt.Sprintf("Delete images %s %d days", countType, countNumber),
		"selection": map[string]interface{}{
			"tagStatus":   "any",
			"countType":   countType,
			"countNumber": countNumber,
			"countUnit":   "days",
		},
		"action": map[string]interface{}{
			"type": "expire",
		},
	}

	jsonBytes, err := json.Marshal(policy)
	if err != nil {
		panic(err)
	}
	return string(jsonBytes)
}

// Need https://github.com/aws/containers-roadmap/issues/921
// to add support for `sinceImagePulled` in the lifecycle policy
func (c *ecrHandler) setRepositoryPolicy(repoName string) error {
	if c.expiresAfterPushDays == 0 && c.expiresAfterPullDays == 0 {
		return nil
	}

	if c.expiresAfterPushDays > 0 && c.expiresAfterPullDays > 0 {
		return errors.New("only one of expiresAfterPushDays and expiresAfterPullDays can be set")
	}

	var policy string

	if c.expiresAfterPullDays > 0 {
		return errors.New("not implemented, need https://github.com/aws/containers-roadmap/issues/921")
	}

	// https://docs.aws.amazon.com/AmazonECR/latest/userguide/LifecyclePolicies.html
	if c.expiresAfterPushDays > 0 {
		policy = fmt.Sprintf(`{"rules": [%s]}`, lifecyclePolicy(1000, "sinceImagePushed", c.expiresAfterPushDays))
	}
	input := ecr.PutLifecyclePolicyInput{
		RepositoryName:      &repoName,
		LifecyclePolicyText: &policy,
	}
	if c.registryId != "" {
		input.RegistryId = &c.registryId
	}

	policyResponse, err := c.client.PutLifecyclePolicy(context.TODO(), &input)
	if err != nil {
		return err
	}
	jsonBytes, err := json.Marshal(policyResponse.LifecyclePolicyText)
	if err != nil {
		return err
	}
	log.Printf("Policy for repo '%s' set: %s", repoName, jsonBytes)
	return nil
}

func (c *ecrHandler) CreateRepository(w http.ResponseWriter, r *http.Request) {
	name, err := registry.RepoGetName(r)
	if err != nil {
		log.Println("ERROR:", err)
		registry.InternalServerError(w, r, err)
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
	var jsonResponse []byte

	if err != nil {
		// Ignore if it already exists
		var awsErr *types.RepositoryAlreadyExistsException
		if errors.As(err, &awsErr) {
			found, _, jsonBytes, err := c.getRepositoryAsJson(r)
			if err != nil {
				log.Println("ERROR:", err)
				registry.InternalServerError(w, r, err)
				return
			}
			if !found {
				log.Printf("RepositoryAlreadyExistsException but repository not found %s: %v", name, awsErr)
				registry.InternalServerError(w, r, err)
				return
			}
			log.Println("Repo already exists", name)
			jsonResponse = jsonBytes
		} else {
			log.Println("ERROR:", err)
			registry.InternalServerError(w, r, err)
			return
		}
	}

	err = c.setRepositoryPolicy(name)
	if err != nil {
		log.Println("ERROR:", err)
		registry.InternalServerError(w, r, err)
		return
	}

	if jsonResponse == nil {
		jsonBytes, err := json.Marshal(createResponse.Repository)
		if err != nil {
			log.Println("ERROR:", err)
			registry.InternalServerError(w, r, err)
			return
		}
		jsonResponse = jsonBytes
	}
	w.WriteHeader(http.StatusOK)
	_, errw := w.Write(jsonResponse)
	if errw != nil {
		log.Println("ERROR:", errw)
	}
}

func (c *ecrHandler) deleteRepositoryPolicy(repoName string) error {
	input := ecr.DeleteLifecyclePolicyInput{
		RepositoryName: &repoName,
	}
	if c.registryId != "" {
		input.RegistryId = &c.registryId
	}
	_, err := c.client.DeleteLifecyclePolicy(context.TODO(), &input)
	if err != nil {
		// Ignore if it didn't exist
		var awsErrRepo *types.RepositoryNotFoundException
		var awsErrPolicy *types.LifecyclePolicyNotFoundException
		if errors.As(err, &awsErrRepo) || errors.As(err, &awsErrPolicy) {
			log.Println("Lifecycle policy not found", repoName)
			return nil
		}
		return err
	}
	log.Printf("Policy for repo '%s' deleted", repoName)
	return nil
}

func (c *ecrHandler) DeleteRepository(w http.ResponseWriter, r *http.Request) {
	name, err := registry.RepoGetName(r)
	if err != nil {
		log.Println("ERROR:", err)
		registry.InternalServerError(w, r, err)
		return
	}

	log.Println("Deleting repo", name)

	err = c.deleteRepositoryPolicy(name)
	if err != nil {
		log.Println("ERROR:", err)
		registry.InternalServerError(w, r, err)
		return
	}

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
		log.Println("ERROR:", err)
		registry.InternalServerError(w, r, err)
		return
	}

	w.WriteHeader(http.StatusOK)
}

func (c *ecrHandler) GetToken(w http.ResponseWriter, r *http.Request) {
	token, err := c.client.GetAuthorizationToken(context.TODO(), &ecr.GetAuthorizationTokenInput{})
	if err != nil {
		log.Println("ERROR:", err)
		registry.InternalServerError(w, r, err)
		return
	}
	if len(token.AuthorizationData) != 1 {
		msg := fmt.Errorf("expected 1 token, got %d", len(token.AuthorizationData))
		log.Println("ERROR:", msg)
		registry.InternalServerError(w, r, msg)
		return
	}

	resp := &registry.RegistryToken{
		Token:   *token.AuthorizationData[0].AuthorizationToken,
		Expires: *token.AuthorizationData[0].ExpiresAt,
	}
	jsonBytes, err := json.Marshal(resp)
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

func envvarIntGreaterThanZero(envvar string) (int, error) {
	s := os.Getenv(envvar)
	if s == "" {
		return 0, nil
	}
	i, err := strconv.Atoi(s)
	if err != nil {
		return 0, fmt.Errorf("ERROR: Invalid %s: %v", envvar, err)
	}
	if i < 0 {
		return 0, fmt.Errorf("%s must be >= 0, got %d", envvar, i)
	}
	return i, nil
}

func Setup(args []string) (registry.IRegistryClient, error) {
	if len(args) != 0 {
		return nil, errors.New("no arguments expected")
	}
	// Automatically looks for a usable configuration
	cfg, err := config.LoadDefaultConfig(context.TODO())
	if err != nil {
		log.Printf("failed to load configuration, %v", err)
		return nil, err
	}

	stsClient := sts.NewFromConfig(cfg)
	identity, err := stsClient.GetCallerIdentity(context.TODO(), &sts.GetCallerIdentityInput{})
	if err != nil {
		log.Printf("failed to get identity, %v", err)
		return nil, err
	}
	log.Printf("Identity: %v", *identity.Arn)

	ecrClient := ecr.NewFromConfig(cfg)

	registryId := os.Getenv("AWS_REGISTRY_ID")
	log.Println("Registry ID:", registryId)

	ecrH := &ecrHandler{
		registryId: registryId,
		client:     ecrClient,
	}

	expiresAfterPushDays, err := envvarIntGreaterThanZero("AWS_ECR_EXPIRES_AFTER_PUSH_DAYS")
	if err != nil {
		return nil, err
	}
	ecrH.expiresAfterPushDays = expiresAfterPushDays

	// Not yet supported by AWS ECR
	// expiresAfterPullDays, err := envvarIntGreaterThanZero("AWS_ECR_EXPIRES_AFTER_PULL_DAYS")
	// if err != nil {
	// 	return nil, err
	// }
	// ecrH.expiresAfterPullDays = expiresAfterPullDays

	return ecrH, nil
}
