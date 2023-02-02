package amazon

import (
	"context"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"log"
	"net/http/httptest"
	"reflect"
	"testing"
	"time"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/service/ecr"
	"github.com/aws/aws-sdk-go-v2/service/ecr/types"
)

type MockEcrClient struct {
	describeRepoRequests  []ecr.DescribeRepositoriesInput
	describeImageRequests []ecr.DescribeImagesInput
	createRepoRequests    []ecr.CreateRepositoryInput
	deleteRepoRequests    []ecr.DeleteRepositoryInput
	getTokenRequests      []ecr.GetAuthorizationTokenInput

	createRepoNoops int
	deleteRepoNoops int
}

const registryId = "123456789012"

func timestamp() time.Time {
	return time.Date(2023, time.January, 1, 12, 34, 56, 0, time.UTC)
}

func (c *MockEcrClient) repository(name string) types.Repository {
	return types.Repository{
		RegistryId:     aws.String(registryId),
		RepositoryName: &name,
		RepositoryUri:  aws.String(fmt.Sprintf("%s.dkr.ecr.eu-west-2.amazonaws.com/%s", registryId, name)),
	}
}

func (c *MockEcrClient) image(name string, tag string) types.ImageDetail {
	return types.ImageDetail{
		ImageTags:      []string{tag},
		RegistryId:     aws.String(registryId),
		RepositoryName: &name,
	}
}

func (c *MockEcrClient) DescribeRepositories(ctx context.Context, input *ecr.DescribeRepositoriesInput, optFns ...func(*ecr.Options)) (response *ecr.DescribeRepositoriesOutput, err error) {
	c.describeRepoRequests = append(c.describeRepoRequests, *input)

	if input.RepositoryNames == nil {
		return &ecr.DescribeRepositoriesOutput{
			Repositories: []types.Repository{
				c.repository("existing-image"),
				c.repository("another-image"),
			},
		}, nil
	}

	if reflect.DeepEqual(input.RepositoryNames, []string{"existing-image"}) {
		return &ecr.DescribeRepositoriesOutput{
			Repositories: []types.Repository{
				c.repository("existing-image"),
			},
		}, nil
	}

	return nil, &types.RepositoryNotFoundException{Message: aws.String("Repository not found")}
}

func (c *MockEcrClient) DescribeImages(ctx context.Context, input *ecr.DescribeImagesInput, optFns ...func(*ecr.Options)) (response *ecr.DescribeImagesOutput, err error) {
	c.describeImageRequests = append(c.describeImageRequests, *input)

	if *input.RepositoryName == "existing-image" && *input.ImageIds[0].ImageTag == "tag" {
		return &ecr.DescribeImagesOutput{
			ImageDetails: []types.ImageDetail{
				c.image("existing-image", "tag"),
			},
		}, nil
	}

	return nil, &types.ImageNotFoundException{Message: aws.String("Image not found")}
}

func (c *MockEcrClient) CreateRepository(ctx context.Context, input *ecr.CreateRepositoryInput, optFns ...func(*ecr.Options)) (response *ecr.CreateRepositoryOutput, err error) {
	c.createRepoRequests = append(c.createRepoRequests, *input)

	if *input.RepositoryName == "new-image" {
		r := c.repository("new-image")
		return &ecr.CreateRepositoryOutput{
			Repository: &r,
		}, nil
	}

	if *input.RepositoryName == "existing-image" {
		c.createRepoNoops++
		return nil, &types.RepositoryAlreadyExistsException{Message: aws.String("Repository already exists")}
	}

	panic("ERROR")
}

func (c *MockEcrClient) DeleteRepository(ctx context.Context, input *ecr.DeleteRepositoryInput, optFns ...func(*ecr.Options)) (response *ecr.DeleteRepositoryOutput, err error) {
	c.deleteRepoRequests = append(c.deleteRepoRequests, *input)

	if *input.RepositoryName == "existing-image" {
		return &ecr.DeleteRepositoryOutput{}, nil
	}

	c.deleteRepoNoops++
	return nil, &types.RepositoryNotFoundException{Message: aws.String("Repository not found")}
}
func (c *MockEcrClient) GetAuthorizationToken(ctx context.Context, input *ecr.GetAuthorizationTokenInput, optFns ...func(*ecr.Options)) (response *ecr.GetAuthorizationTokenOutput, err error) {
	c.getTokenRequests = append(c.getTokenRequests, *input)

	return &ecr.GetAuthorizationTokenOutput{
		AuthorizationData: []types.AuthorizationData{
			{
				AuthorizationToken: aws.String("token"),
				ExpiresAt:          aws.Time(timestamp()),
			},
		},
	}, nil
}

func (e *ecrHandler) assertRequestCounts(t *testing.T, expectedDescribeRepos int, expectedCreates int, expectedDeletes int, expectedDescribeImages int, expectedTokens int) {
	nDescribeRepos := len(e.client.(*MockEcrClient).describeRepoRequests)
	nCreateRepos := len(e.client.(*MockEcrClient).createRepoRequests)
	nDeleteRepos := len(e.client.(*MockEcrClient).deleteRepoRequests)
	nDescribeImages := len(e.client.(*MockEcrClient).describeImageRequests)
	nGetTokens := len(e.client.(*MockEcrClient).getTokenRequests)
	if nDescribeRepos != expectedDescribeRepos {
		t.Errorf("Expected %v describe repo request: %v", expectedDescribeRepos, nDescribeRepos)
	}
	if nCreateRepos != expectedCreates {
		t.Errorf("Expected %v create request: %v", expectedCreates, nCreateRepos)
	}
	if nDeleteRepos != expectedDeletes {
		t.Errorf("Expected %v delete request: %v", expectedDeletes, nDeleteRepos)
	}
	if nDescribeImages != expectedDescribeImages {
		t.Errorf("Expected %v describe image request: %v", expectedDescribeImages, nDescribeImages)
	}
	if nGetTokens != expectedTokens {
		t.Errorf("Expected %v get token request: %v", expectedTokens, nGetTokens)
	}
}

func (e *ecrHandler) assertNoopCounts(t *testing.T, expectedCreateNoops int, expectedDeleteNoops int) {
	nCreateNoops := e.client.(*MockEcrClient).createRepoNoops
	nDeleteNoops := e.client.(*MockEcrClient).deleteRepoNoops
	if nCreateNoops != expectedCreateNoops {
		t.Errorf("Expected %v create noops: %v", expectedCreateNoops, nCreateNoops)
	}
	if nDeleteNoops != expectedDeleteNoops {
		t.Errorf("Expected %v delete noops: %v", expectedDeleteNoops, nDeleteNoops)
	}
}

// Tests

func TestList(t *testing.T) {
	e := &ecrHandler{
		registryId: registryId,
		client:     &MockEcrClient{},
	}

	req := httptest.NewRequest("GET", "/repos", nil)
	w := httptest.NewRecorder()
	e.ServeHTTP(w, req)
	res := w.Result()
	defer res.Body.Close()
	data, err := ioutil.ReadAll(w.Result().Body)
	if err != nil {
		t.Errorf("Unexpected error: %v", err)
	}

	if res.StatusCode != 200 {
		t.Errorf("Expected StatusCode 200: %v", res.StatusCode)
	}

	e.assertRequestCounts(t, 1, 0, 0, 0, 0)

	fmt.Println(string(data))

	var result []map[string]interface{}
	err2 := json.Unmarshal([]byte(data), &result)
	if err2 != nil {
		t.Errorf("Unexpected error: %v", err2)
	}

	if len(result) != 2 {
		t.Errorf("Expected 2 items: %v", result)
	}
	if result[0]["RepositoryName"] != "existing-image" || result[0]["RepositoryUri"] != "123456789012.dkr.ecr.eu-west-2.amazonaws.com/existing-image" {
		t.Errorf("Expected 'existing-image': %v", result[0])
	}
	if result[1]["RepositoryName"] != "another-image" || result[1]["RepositoryUri"] != "123456789012.dkr.ecr.eu-west-2.amazonaws.com/another-image" {
		t.Errorf("Expected 'another-image': %v", result[1])
	}
}

func TestGetByName(t *testing.T) {
	e := &ecrHandler{
		registryId: registryId,
		client:     &MockEcrClient{},
	}

	{
		r, err := e.getRepoByName("existing-image")
		if err != nil {
			t.Errorf("Unexpected error: %v", err)
		}
		if *r.RegistryId != registryId {
			t.Errorf("Unexpected registry id: %v", r.RegistryId)
		}
		if *r.RepositoryName != "existing-image" {
			t.Errorf("Unexpected repository name: %v", r.RepositoryName)
		}
	}

	{
		r, err := e.getRepoByName("new-image")
		if err != nil {
			t.Errorf("Unexpected error: %v", err)
		}
		if r != nil {
			t.Errorf("Unexpected response: %v", r)
		}
	}
}

func TestGetRepositoryAsJson(t *testing.T) {
	e := &ecrHandler{
		registryId: registryId,
		client:     &MockEcrClient{},
	}

	{
		req := httptest.NewRequest("GET", "/repo/existing-image", nil)
		found, name, jsonBytes, err := e.getRepositoryAsJson(req)
		if err != nil {
			t.Errorf("Unexpected error: %v", err)
		}
		if !found {
			t.Errorf("Expected repository to be found: %v", found)
		}
		if name != "existing-image" {
			t.Errorf("Unexpected repository name: %v", name)
		}
		var jsonObj map[string]string
		err = json.Unmarshal(jsonBytes, &jsonObj)
		if err != nil {
			t.Errorf("Unexpected error: %v", err)
		}

		if jsonObj["RepositoryName"] != "existing-image" || jsonObj["RegistryId"] != registryId || jsonObj["RepositoryUri"] != "123456789012.dkr.ecr.eu-west-2.amazonaws.com/existing-image" {
			t.Errorf("Unexpected json: %s", jsonBytes)
		}
	}

	{
		req := httptest.NewRequest("GET", "/repo/new-image", nil)
		found, name, jsonBytes, err := e.getRepositoryAsJson(req)
		if err != nil {
			t.Errorf("Unexpected error: %v", err)
		}
		if found {
			t.Errorf("Expected repository to not be found: %v", found)
		}
		if name != "new-image" {
			t.Errorf("Unexpected repository name: %v", name)
		}
		if string(jsonBytes) != "null" {
			t.Errorf("Unexpected json: %s", jsonBytes)
		}
	}
}

func TestGetRepo(t *testing.T) {
	testCases := []struct {
		imageName          string
		expectedStatusCode int
	}{
		{"existing-image", 200},
		{"new-image", 404},
	}

	for _, tc := range testCases {
		t.Run(fmt.Sprintf("%v,%v", tc.imageName, tc.expectedStatusCode), func(t *testing.T) {

			e := &ecrHandler{
				registryId: registryId,
				client:     &MockEcrClient{},
			}

			req := httptest.NewRequest("GET", "/repo/"+tc.imageName, nil)
			w := httptest.NewRecorder()
			e.ServeHTTP(w, req)
			res := w.Result()
			defer res.Body.Close()
			data, err := ioutil.ReadAll(w.Result().Body)
			if err != nil {
				t.Errorf("Unexpected error: %v", err)
			}

			if res.StatusCode != tc.expectedStatusCode {
				t.Errorf("Expected StatusCode %v: %v", tc.expectedStatusCode, res.StatusCode)
			}

			e.assertRequestCounts(t, 1, 0, 0, 0, 0)

			if tc.expectedStatusCode == 200 {
				var result map[string]interface{}
				err2 := json.Unmarshal([]byte(data), &result)
				if err2 != nil {
					t.Errorf("Unexpected error: %v", err2)
				}

				if result["RepositoryName"] != "existing-image" {
					t.Errorf("Expected 'existing-image': %v", result)
				}
			} else {
				if string(data) != "null" {
					t.Errorf("Expected 'null': %v", string(data))
				}
			}
		})
	}
}

func TestGetImage(t *testing.T) {
	testCases := []struct {
		imageName          string
		tag                string
		expectedStatusCode int
	}{
		{"existing-image", "tag", 200},
		{"existing-image", "new-tag", 404},
		{"new-image", "tag", 404},
	}

	for _, tc := range testCases {
		t.Run(fmt.Sprintf("%v,%v", tc.tag, tc.expectedStatusCode), func(t *testing.T) {

			e := &ecrHandler{
				registryId: registryId,
				client:     &MockEcrClient{},
			}

			req := httptest.NewRequest("GET", fmt.Sprintf("/image/%s:%s", tc.imageName, tc.tag), nil)
			w := httptest.NewRecorder()
			log.Println("Calling ServeHTTP")
			e.ServeHTTP(w, req)
			log.Println("Called ServeHTTP")
			res := w.Result()
			defer res.Body.Close()
			data, err := ioutil.ReadAll(w.Result().Body)
			if err != nil {
				t.Errorf("Unexpected error: %v", err)
			}

			if res.StatusCode != tc.expectedStatusCode {
				t.Errorf("Expected StatusCode %v: %v", tc.expectedStatusCode, res.StatusCode)
			}

			e.assertRequestCounts(t, 0, 0, 0, 1, 0)

			if tc.expectedStatusCode == 200 {
				var result map[string]interface{}
				err2 := json.Unmarshal([]byte(data), &result)
				if err2 != nil {
					t.Errorf("Unexpected error: %v", err2)
				}

				if result["RepositoryName"] != "existing-image" {
					t.Errorf("Expected 'existing-image': %v", result)
				}
			} else {
				if string(data) != "null" {
					t.Errorf("Expected 'null': %v", string(data))
				}
			}
		})
	}
}

func TestCreate(t *testing.T) {
	testCases := []struct {
		imageName string
		create    bool
	}{
		{"existing-image", false},
		{"new-image", true},
	}

	for _, tc := range testCases {
		t.Run(fmt.Sprintf("%v,%v", tc.imageName, tc.create), func(t *testing.T) {

			e := &ecrHandler{
				registryId: registryId,
				client:     &MockEcrClient{},
			}

			req := httptest.NewRequest("POST", "/repo/"+tc.imageName, nil)
			w := httptest.NewRecorder()
			e.ServeHTTP(w, req)
			res := w.Result()
			defer res.Body.Close()
			data, err := ioutil.ReadAll(w.Result().Body)
			if err != nil {
				t.Errorf("Unexpected error: %v", err)
			}

			if res.StatusCode != 200 {
				t.Errorf("Expected StatusCode 200: %v", res.StatusCode)
			}

			if tc.create {
				e.assertRequestCounts(t, 0, 1, 0, 0, 0)
				e.assertNoopCounts(t, 0, 0)
			} else {
				e.assertRequestCounts(t, 1, 1, 0, 0, 0)
				e.assertNoopCounts(t, 1, 0)
			}

			var result map[string]interface{}
			err2 := json.Unmarshal([]byte(data), &result)
			if err2 != nil {
				t.Errorf("Unexpected error: %v", err2)
			}

			if result["RepositoryName"] != tc.imageName {
				t.Errorf("Expected '%v': %v", tc.imageName, result)
			}
		})
	}
}

func TestDelete(t *testing.T) {
	testCases := []struct {
		imageName string
		delete    bool
	}{
		{"existing-image", true},
		{"new-image", false},
	}

	for _, tc := range testCases {
		t.Run(fmt.Sprintf("%v,%v", tc.imageName, tc.delete), func(t *testing.T) {

			e := &ecrHandler{
				registryId: registryId,
				client:     &MockEcrClient{},
			}

			req := httptest.NewRequest("DELETE", "/repo/"+tc.imageName, nil)
			w := httptest.NewRecorder()
			e.ServeHTTP(w, req)
			res := w.Result()
			defer res.Body.Close()
			_, err := ioutil.ReadAll(w.Result().Body)
			if err != nil {
				t.Errorf("Unexpected error: %v", err)
			}

			if res.StatusCode != 200 {
				t.Errorf("Expected StatusCode 200: %v", res.StatusCode)
			}

			if tc.delete {
				e.assertRequestCounts(t, 0, 0, 1, 0, 0)
				e.assertNoopCounts(t, 0, 0)
			} else {
				e.assertRequestCounts(t, 0, 0, 1, 0, 0)
				e.assertNoopCounts(t, 0, 1)
			}
		})
	}
}

func TestToken(t *testing.T) {
	e := &ecrHandler{
		registryId: registryId,
		client:     &MockEcrClient{},
	}

	req := httptest.NewRequest("POST", "/token", nil)
	w := httptest.NewRecorder()
	e.ServeHTTP(w, req)
	res := w.Result()
	defer res.Body.Close()
	data, err := ioutil.ReadAll(w.Result().Body)
	if err != nil {
		t.Errorf("Unexpected error: %v", err)
	}

	if res.StatusCode != 200 {
		t.Errorf("Expected StatusCode 200: %v", res.StatusCode)
	}

	var result map[string]interface{}
	err2 := json.Unmarshal([]byte(data), &result)
	if err2 != nil {
		t.Errorf("Unexpected error: %v", err2)
	}

	if result["token"] != "token" {
		t.Errorf("Expected token:'token': %v", result)
	}
	expectedExpires := "2023-01-01T12:34:56Z"
	if result["expires"] != expectedExpires {
		t.Errorf("Expected expires:'%v': %v", expectedExpires, result)
	}
}
