package amazon

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"
	"reflect"
	"testing"
	"time"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/service/ecr"
	"github.com/aws/aws-sdk-go-v2/service/ecr/types"

	"github.com/manics/binderhub-container-registry-helper/common"
)

type MockEcrClient struct {
	describeRepoRequests    []ecr.DescribeRepositoriesInput
	describeImageRequests   []ecr.DescribeImagesInput
	createRepoRequests      []ecr.CreateRepositoryInput
	putLifecycleRequests    []ecr.PutLifecyclePolicyInput
	deleteRepoRequests      []ecr.DeleteRepositoryInput
	deleteLifecycleRequests []ecr.DeleteLifecyclePolicyInput
	getTokenRequests        []ecr.GetAuthorizationTokenInput

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

func (c *MockEcrClient) PutLifecyclePolicy(ctx context.Context, input *ecr.PutLifecyclePolicyInput, optFns ...func(*ecr.Options)) (response *ecr.PutLifecyclePolicyOutput, err error) {
	c.putLifecycleRequests = append(c.putLifecycleRequests, *input)

	var result map[string]interface{}
	err = json.Unmarshal([]byte(*input.LifecyclePolicyText), &result)
	if err != nil {
		fmt.Println(*input.LifecyclePolicyText)
		return nil, fmt.Errorf("Invalid JSON: %v", err)
	}

	response = &ecr.PutLifecyclePolicyOutput{
		RegistryId:          aws.String(registryId),
		RepositoryName:      input.RepositoryName,
		LifecyclePolicyText: input.LifecyclePolicyText,
	}
	return response, nil
}

func (c *MockEcrClient) DeleteRepository(ctx context.Context, input *ecr.DeleteRepositoryInput, optFns ...func(*ecr.Options)) (response *ecr.DeleteRepositoryOutput, err error) {
	c.deleteRepoRequests = append(c.deleteRepoRequests, *input)

	if *input.RepositoryName == "existing-image" {
		return &ecr.DeleteRepositoryOutput{}, nil
	}

	c.deleteRepoNoops++
	return nil, &types.RepositoryNotFoundException{Message: aws.String("Repository not found")}
}

func (c *MockEcrClient) DeleteLifecyclePolicy(ctx context.Context, input *ecr.DeleteLifecyclePolicyInput, optFns ...func(*ecr.Options)) (response *ecr.DeleteLifecyclePolicyOutput, err error) {
	c.deleteLifecycleRequests = append(c.deleteLifecycleRequests, *input)

	response = &ecr.DeleteLifecyclePolicyOutput{
		RegistryId:     aws.String(registryId),
		RepositoryName: input.RepositoryName,
	}
	return response, nil
}

func (c *MockEcrClient) GetAuthorizationToken(ctx context.Context, input *ecr.GetAuthorizationTokenInput, optFns ...func(*ecr.Options)) (response *ecr.GetAuthorizationTokenOutput, err error) {
	c.getTokenRequests = append(c.getTokenRequests, *input)

	return &ecr.GetAuthorizationTokenOutput{
		AuthorizationData: []types.AuthorizationData{
			{
				AuthorizationToken: aws.String("QVdTOnRva2Vu"),
				ExpiresAt:          aws.Time(timestamp()),
				ProxyEndpoint:      aws.String("https://123456789012.dkr.ecr.us-east-1.amazonaws.com"),
			},
		},
	}, nil
}

func (e *MockEcrClient) assertCounts(t *testing.T, expected map[string]int) {
	countRequests := map[string]int{
		"describeRepos":    len(e.describeRepoRequests),
		"createRepos":      len(e.createRepoRequests),
		"putLifecycles":    len(e.putLifecycleRequests),
		"deleteRepos":      len(e.deleteRepoRequests),
		"deleteLifecycles": len(e.deleteLifecycleRequests),
		"describeImages":   len(e.describeImageRequests),
		"getTokens":        len(e.getTokenRequests),
	}
	for k, v := range countRequests {
		e := 0
		if val, ok := expected[k]; ok {
			e = val
			delete(expected, k)
		}
		if v != e {
			t.Errorf("Expected %d %s requests: %d", e, k, v)
		}
	}

	countNoops := map[string]int{
		"createNoops": e.createRepoNoops,
		"deleteNoops": e.deleteRepoNoops,
	}
	for k, v := range countNoops {
		e := 0
		if val, ok := expected[k]; ok {
			e = val
			delete(expected, k)
		}
		if v != e {
			t.Errorf("Expected %d %s: %d", e, k, v)
		}
	}

	if len(expected) > 0 {
		t.Errorf("Invalid expected counts: %v", expected)
	}
}

func request(t *testing.T, method string, path string) (MockEcrClient, *http.Response, []byte, error) {
	ecrClient := MockEcrClient{}
	e := &ecrHandler{
		registryId: registryId,
		client:     &ecrClient,
	}
	s := &common.RegistryServer{
		Client: e,
	}

	req := httptest.NewRequest(method, path, nil)
	w := httptest.NewRecorder()
	s.ServeHTTP(w, req)
	res := w.Result()
	defer res.Body.Close()
	data, err := io.ReadAll(w.Result().Body)
	return ecrClient, res, data, err
}

// Tests

func TestList(t *testing.T) {
	ecrClient, res, data, err := request(t, "GET", "/repos/")
	if err != nil {
		t.Errorf("Unexpected error: %v", err)
	}

	if res.StatusCode != 200 {
		t.Errorf("Expected StatusCode 200: %v", res.StatusCode)
	}

	ecrClient.assertCounts(t, map[string]int{
		"describeRepos": 1,
	})

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
			ecrClient, res, data, err := request(t, "GET", "/repo/"+tc.imageName)
			if err != nil {
				t.Errorf("Unexpected error: %v", err)
			}

			if res.StatusCode != tc.expectedStatusCode {
				t.Errorf("Expected StatusCode %v: %v", tc.expectedStatusCode, res.StatusCode)
			}

			ecrClient.assertCounts(t, map[string]int{
				"describeRepos": 1,
			})

			if tc.expectedStatusCode == 200 {
				var result map[string]interface{}
				err2 := json.Unmarshal([]byte(data), &result)
				if err2 != nil {
					t.Errorf("Unexpected error: %v", err2)
				}

				if result["RepositoryName"] != "existing-image" {
					t.Errorf("Expected 'existing-image': %v", result)
				}
			} else if string(data) != "null" {
				t.Errorf("Expected 'null': %v", string(data))
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

			ecrClient, res, data, err := request(t, "GET", fmt.Sprintf("/image/%s:%s", tc.imageName, tc.tag))
			if err != nil {
				t.Errorf("Unexpected error: %v", err)
			}

			if res.StatusCode != tc.expectedStatusCode {
				t.Errorf("Expected StatusCode %v: %v", tc.expectedStatusCode, res.StatusCode)
			}

			ecrClient.assertCounts(t, map[string]int{
				"describeImages": 1,
			})

			if tc.expectedStatusCode == 200 {
				var result map[string]interface{}
				err2 := json.Unmarshal([]byte(data), &result)
				if err2 != nil {
					t.Errorf("Unexpected error: %v", err2)
				}

				if result["RepositoryName"] != "existing-image" {
					t.Errorf("Expected 'existing-image': %v", result)
				}
			} else if string(data) != "null\n" {
				t.Errorf("Expected 'null': %v", string(data))
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
			ecrClient, res, data, err := request(t, "POST", "/repo/"+tc.imageName)
			if err != nil {
				t.Errorf("Unexpected error: %v", err)
			}

			if res.StatusCode != 200 {
				t.Errorf("Expected StatusCode 200: %v", res.StatusCode)
			}

			if tc.create {
				ecrClient.assertCounts(t, map[string]int{
					"createRepos": 1,
				})
			} else {
				ecrClient.assertCounts(t, map[string]int{
					"describeRepos": 1,
					"createRepos":   1,
					"createNoops":   1,
				})
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
			ecrClient, res, _, err := request(t, "DELETE", "/repo/"+tc.imageName)
			if err != nil {
				t.Errorf("Unexpected error: %v", err)
			}

			if res.StatusCode != 200 {
				t.Errorf("Expected StatusCode 200: %v", res.StatusCode)
			}

			if tc.delete {
				ecrClient.assertCounts(t, map[string]int{
					"deleteRepos":      1,
					"deleteLifecycles": 1,
				})
			} else {
				ecrClient.assertCounts(t, map[string]int{
					"deleteRepos":      1,
					"deleteNoops":      1,
					"deleteLifecycles": 1,
				})

			}
		})
	}
}

func TestToken(t *testing.T) {
	testCases := []struct {
		suffix string
	}{
		{""},
		{"/always/ignored"},
	}

	for _, tc := range testCases {
		t.Run(tc.suffix, func(t *testing.T) {
			ecrClient, res, data, err := request(t, "POST", "/token"+tc.suffix)
			if err != nil {
				t.Errorf("Unexpected error: %v", err)
			}

			if res.StatusCode != 200 {
				t.Errorf("Expected StatusCode 200: %v", res.StatusCode)
			}

			ecrClient.assertCounts(t, map[string]int{
				"getTokens": 1,
			})

			var result map[string]string
			err2 := json.Unmarshal([]byte(data), &result)
			if err2 != nil {
				t.Errorf("Unexpected error: %v", err2)
			}

			expected := map[string]string{
				"username": "AWS",
				"password": "token",
				"expires":  "2023-01-01T12:34:56Z",
				"registry": "https://123456789012.dkr.ecr.us-east-1.amazonaws.com",
			}
			for k, v := range expected {
				if result[k] != v {
					t.Errorf("Expected %s=%s: %v", k, v, result[k])
				}
			}
		})
	}
}
