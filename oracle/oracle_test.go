package oracle

import (
	"context"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/manics/oracle-container-repositories-svc/registry"

	"github.com/oracle/oci-go-sdk/v65/artifacts"
	"github.com/oracle/oci-go-sdk/v65/common"
)

// Helpers

type MockServiceError struct {
	code string
}

func (e MockServiceError) GetHTTPStatusCode() int {
	panic("Not implemented")
}

func (e MockServiceError) GetMessage() string {
	panic("Not implemented")
}

func (e MockServiceError) GetCode() string {
	return e.code
}

func (e MockServiceError) GetOpcRequestID() string {
	panic("Not implemented")
}

func (e MockServiceError) Error() string {
	return e.code
}

type MockArtifactsClient struct {
	listRequests       []artifacts.ListContainerRepositoriesRequest
	listImagesRequests []artifacts.ListContainerImagesRequest
	createRequests     []artifacts.CreateContainerRepositoryRequest
	deleteRequests     []artifacts.DeleteContainerRepositoryRequest

	createRepoNoops int
	deleteRepoNoops int
}

func (c *MockArtifactsClient) containerRepositorySummary(name string) *artifacts.ContainerRepositorySummary {
	return &artifacts.ContainerRepositorySummary{
		CompartmentId:     nil,
		DisplayName:       common.String(name),
		Id:                common.String("id-" + name),
		ImageCount:        nil,
		IsPublic:          nil,
		LayerCount:        nil,
		LayersSizeInBytes: nil,
		LifecycleState:    "",
		TimeCreated:       nil,
		BillableSizeInGBs: nil,
	}
}

func (c *MockArtifactsClient) containerRepository(name string) *artifacts.ContainerRepository {
	return &artifacts.ContainerRepository{
		CompartmentId:     nil,
		CreatedBy:         nil,
		DisplayName:       common.String(name),
		Id:                common.String("id-" + name),
		ImageCount:        nil,
		IsImmutable:       nil,
		IsPublic:          nil,
		LayerCount:        nil,
		LayersSizeInBytes: nil,
		LifecycleState:    "",
		TimeCreated:       nil,
		BillableSizeInGBs: nil,
		Readme:            nil,
		TimeLastPushed:    nil,
	}
}

func (c *MockArtifactsClient) ListContainerRepositories(ctx context.Context, request artifacts.ListContainerRepositoriesRequest) (response artifacts.ListContainerRepositoriesResponse, err error) {
	c.listRequests = append(c.listRequests, request)

	if request.DisplayName == nil {
		fmt.Println(request.DisplayName, request)
		return artifacts.ListContainerRepositoriesResponse{
			ContainerRepositoryCollection: artifacts.ContainerRepositoryCollection{
				Items: []artifacts.ContainerRepositorySummary{
					*c.containerRepositorySummary("existing-image"),
					*c.containerRepositorySummary("another-image"),
				},
			},
		}, nil
	}

	if *request.DisplayName == "existing-image" {
		return artifacts.ListContainerRepositoriesResponse{
			ContainerRepositoryCollection: artifacts.ContainerRepositoryCollection{
				Items: []artifacts.ContainerRepositorySummary{
					*c.containerRepositorySummary("existing-image"),
				},
			},
		}, nil
	}

	return artifacts.ListContainerRepositoriesResponse{}, nil
}

func (c *MockArtifactsClient) ListContainerImages(ctx context.Context, request artifacts.ListContainerImagesRequest) (response artifacts.ListContainerImagesResponse, err error) {
	c.listImagesRequests = append(c.listImagesRequests, request)

	existing := artifacts.ContainerImageSummary{
		DisplayName:    common.String("existing-image:tag"),
		Id:             common.String("id-existing-image:tag"),
		RepositoryName: common.String("existing-image"),
	}

	if *request.DisplayName == "existing-image:tag" {
		fmt.Println(request.DisplayName, request)
		return artifacts.ListContainerImagesResponse{
			ContainerImageCollection: artifacts.ContainerImageCollection{
				Items: []artifacts.ContainerImageSummary{
					existing,
				},
			},
		}, nil
	}

	return artifacts.ListContainerImagesResponse{}, nil
}

func (c *MockArtifactsClient) CreateContainerRepository(ctx context.Context, request artifacts.CreateContainerRepositoryRequest) (response artifacts.CreateContainerRepositoryResponse, err error) {

	c.createRequests = append(c.createRequests, request)

	if *request.DisplayName == "new-image" {
		return artifacts.CreateContainerRepositoryResponse{
			ContainerRepository: *c.containerRepository("new-image"),
		}, nil
	}

	if *request.DisplayName == "existing-image" {
		c.createRepoNoops++
		return artifacts.CreateContainerRepositoryResponse{}, MockServiceError{code: "NAMESPACE_CONFLICT"}
	}

	panic("ERROR")
}

func (c *MockArtifactsClient) DeleteContainerRepository(ctx context.Context, request artifacts.DeleteContainerRepositoryRequest) (response artifacts.DeleteContainerRepositoryResponse, err error) {
	c.deleteRequests = append(c.deleteRequests, request)

	if *request.RepositoryId == "id-existing-image" {
		return artifacts.DeleteContainerRepositoryResponse{}, nil
	}
	return artifacts.DeleteContainerRepositoryResponse{}, fmt.Errorf("Image doesn't exist")
}

func (e *MockArtifactsClient) assertCounts(t *testing.T, expected map[string]int) {
	countRequests := map[string]int{
		"listRepos":   len(e.listRequests),
		"createRepos": len(e.createRequests),
		"deleteRepos": len(e.deleteRequests),
		"listImages":  len(e.listImagesRequests),
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

func request(t *testing.T, method string, path string) (MockArtifactsClient, *http.Response, []byte, error) {
	art := MockArtifactsClient{}
	a := &artifactsHandler{
		compartmentId: "compartmentId",
		client:        &art,
		namespace:     "namespace",
	}
	s := &registry.RegistryServer{
		Client: a,
	}

	req := httptest.NewRequest(method, path, nil)
	w := httptest.NewRecorder()
	s.ServeHTTP(w, req)
	res := w.Result()
	defer res.Body.Close()
	data, err := ioutil.ReadAll(w.Result().Body)
	return art, res, data, err
}

// Tests

func TestGetByName(t *testing.T) {
	a := &artifactsHandler{
		compartmentId: "compartmentId",
		client:        &MockArtifactsClient{},
		namespace:     "namespace",
	}

	{
		req := httptest.NewRequest("GET", "/existing-image", nil)
		_, _, err := a.getByName(req)
		if err == nil {
			t.Errorf("Expected error: %v", err)
		}
	}

	{
		req := httptest.NewRequest("GET", "/repo/existing-image", nil)
		_, _, err := a.getByName(req)
		if err == nil {
			t.Errorf("Expected error: %v", err)
		}
	}

	{
		req := httptest.NewRequest("GET", "/repo/incorrect-namespace/existing-image", nil)
		_, _, err := a.getByName(req)
		if err == nil {
			t.Errorf("Expected error: %v", err)
		}
	}

	{
		req := httptest.NewRequest("GET", "/repo/namespace/existing-image", nil)
		container, name, err := a.getByName(req)
		if err != nil {
			t.Errorf("Unexpected error: %v", err)
		}
		if name != "existing-image" {
			t.Errorf("Expected name 'existing-image': %v", name)
		}
		if *container.DisplayName != "existing-image" {
			t.Errorf("Expected DisplayName 'existing-image': %v", container.DisplayName)
		}
		if *container.Id != "id-existing-image" {
			t.Errorf("Expected Id 'id-existing-image': %v", container.DisplayName)
		}
	}

	{
		req := httptest.NewRequest("GET", "/repo/namespace/new-image", nil)
		container, name, err := a.getByName(req)
		if err != nil {
			t.Errorf("Unexpected error: %v", err)
		}
		if name != "new-image" {
			t.Errorf("Expected name 'new-image': %v", name)
		}
		if container != nil {
			t.Errorf("Unexpected container: %v", container)
		}
	}
}

func TestGetImage(t *testing.T) {
	art, res, data, err := request(t, "GET", "/image/namespace/existing-image:tag")
	if err != nil {
		t.Errorf("Unexpected error: %v", err)
	}

	if res.StatusCode != 200 {
		t.Errorf("Expected StatusCode 200: %v", res.StatusCode)
	}

	art.assertCounts(t, map[string]int{
		"listImages": 1,
	})

	fmt.Println(string(data))

	var result map[string]interface{}
	err2 := json.Unmarshal([]byte(data), &result)
	if err2 != nil {
		t.Errorf("Unexpected error: %v", err2)
	}

	if result["displayName"] != "existing-image:tag" || result["id"] != "id-existing-image:tag" {
		t.Errorf("Expected 'existing-image': %v", result)
	}
}

func TestListRepos(t *testing.T) {
	art, res, data, err := request(t, "GET", "/repos")
	if err != nil {
		t.Errorf("Unexpected error: %v", err)
	}

	if res.StatusCode != 200 {
		t.Errorf("Expected StatusCode 200: %v", res.StatusCode)
	}

	art.assertCounts(t, map[string]int{
		"listRepos": 1,
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
	if result[0]["displayName"] != "existing-image" || result[0]["id"] != "id-existing-image" {
		t.Errorf("Expected 'existing-image': %v", result[0])
	}
	if result[1]["displayName"] != "another-image" || result[1]["id"] != "id-another-image" {
		t.Errorf("Expected 'another-image': %v", result[1])
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

			art, res, data, err := request(t, "POST", "/repo/namespace/"+tc.imageName)
			if err != nil {
				t.Errorf("Unexpected error: %v", err)
			}

			if res.StatusCode != 200 {
				t.Errorf("Expected StatusCode 200: %v", res.StatusCode)
			}

			if tc.create {
				art.assertCounts(t, map[string]int{
					"createRepos": 1,
				})
			} else {
				art.assertCounts(t, map[string]int{
					"listRepos":   1,
					"createRepos": 1,
					"createNoops": 1,
				})
			}

			var result map[string]interface{}
			err2 := json.Unmarshal([]byte(data), &result)
			if err2 != nil {
				t.Errorf("Unexpected error: %v", err2)
			}

			if result["displayName"] != tc.imageName {
				t.Errorf("Expected '%v': %v", tc.imageName, result)
			}
		})
	}
}

func TestDelete(t *testing.T) {
	{
		art, res, _, err := request(t, "DELETE", "/repo/namespace/new-image")
		if err != nil {
			t.Errorf("Unexpected error: %v", err)
		}

		if res.StatusCode != 200 {
			t.Errorf("Expected StatusCode 200: %v", res.StatusCode)
		}

		art.assertCounts(t, map[string]int{
			"listRepos": 1,
		})
	}

	{
		art, res, _, err := request(t, "DELETE", "/repo/namespace/existing-image")
		if err != nil {
			t.Errorf("Unexpected error: %v", err)
		}

		if res.StatusCode != 200 {
			t.Errorf("Expected StatusCode 200: %v", res.StatusCode)
		}

		art.assertCounts(t, map[string]int{
			"listRepos":   1,
			"deleteRepos": 1,
		})
	}
}
