package oracle

import (
	"context"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"net/http/httptest"
	"testing"

	"github.com/oracle/oci-go-sdk/v65/artifacts"
	"github.com/oracle/oci-go-sdk/v65/common"
)

// Helpers

type MockArtifactsClient struct {
	listRequests       []artifacts.ListContainerRepositoriesRequest
	listImagesRequests []artifacts.ListContainerImagesRequest
	createRequests     []artifacts.CreateContainerRepositoryRequest
	deleteRequests     []artifacts.DeleteContainerRepositoryRequest
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

	// TODO: return error already exists
	// if *request.DisplayName == "existing-image" {
	// 	return artifacts.CreateContainerRepositoryResponse{
	// 		ContainerRepository: *r,
	// 	}, nil
	// }
	// return artifacts.CreateContainerRepositoryResponse{}, fmt.Errorf("ERROR")
	panic("ERROR")
}

func (c *MockArtifactsClient) DeleteContainerRepository(ctx context.Context, request artifacts.DeleteContainerRepositoryRequest) (response artifacts.DeleteContainerRepositoryResponse, err error) {
	c.deleteRequests = append(c.deleteRequests, request)

	if *request.RepositoryId == "id-existing-image" {
		return artifacts.DeleteContainerRepositoryResponse{}, nil
	}
	return artifacts.DeleteContainerRepositoryResponse{}, fmt.Errorf("Image doesn't exist")
}

func (a *artifactsHandler) assertRequestCounts(t *testing.T, expectedLists int, expectedCreates int, expectedDeletes int, expectedListImagesRequests int) {
	nLists := len(a.client.(*MockArtifactsClient).listRequests)
	nCreates := len(a.client.(*MockArtifactsClient).createRequests)
	nDeletes := len(a.client.(*MockArtifactsClient).deleteRequests)
	nListImages := len(a.client.(*MockArtifactsClient).listImagesRequests)
	if nLists != expectedLists {
		t.Errorf("Expected %v list request: %v", expectedLists, nLists)
	}
	if nCreates != expectedCreates {
		t.Errorf("Expected %v create request: %v", expectedCreates, nCreates)
	}
	if nDeletes != expectedDeletes {
		t.Errorf("Expected %v delete request: %v", expectedDeletes, nDeletes)
	}
	if nListImages != expectedListImagesRequests {
		t.Errorf("Expected %v list images request: %v", expectedListImagesRequests, nListImages)
	}
}

// Tests

func TestGetByName(t *testing.T) {
	a := &artifactsHandler{
		compartmentId: "compartmentId",
		client:        &MockArtifactsClient{},
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
		req := httptest.NewRequest("GET", "/repo/new-image", nil)
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
	a := &artifactsHandler{
		compartmentId: "compartmentId",
		client:        &MockArtifactsClient{},
	}

	req := httptest.NewRequest("GET", "/image/existing-image:tag", nil)
	w := httptest.NewRecorder()
	a.ServeHTTP(w, req)
	res := w.Result()
	defer res.Body.Close()
	data, err := ioutil.ReadAll(w.Result().Body)
	if err != nil {
		t.Errorf("Unexpected error: %v", err)
	}

	if res.StatusCode != 200 {
		t.Errorf("Expected StatusCode 200: %v", res.StatusCode)
	}

	a.assertRequestCounts(t, 0, 0, 0, 1)

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
	a := &artifactsHandler{
		compartmentId: "compartmentId",
		client:        &MockArtifactsClient{},
	}

	req := httptest.NewRequest("GET", "/repos", nil)
	w := httptest.NewRecorder()
	a.ServeHTTP(w, req)
	res := w.Result()
	defer res.Body.Close()
	data, err := ioutil.ReadAll(w.Result().Body)
	if err != nil {
		t.Errorf("Unexpected error: %v", err)
	}

	if res.StatusCode != 200 {
		t.Errorf("Expected StatusCode 200: %v", res.StatusCode)
	}

	a.assertRequestCounts(t, 1, 0, 0, 0)

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
	a := &artifactsHandler{
		compartmentId: "compartmentId",
		client:        &MockArtifactsClient{},
	}

	req := httptest.NewRequest("POST", "/repo/new-image", nil)
	w := httptest.NewRecorder()
	a.ServeHTTP(w, req)
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
	json.Unmarshal([]byte(data), &result)
	if result["displayName"] != "new-image" {
		t.Errorf("Expected DisplayName 'new-image': %v", result["displayName"])
	}

	a.assertRequestCounts(t, 0, 1, 0, 0)
}

func TestDelete(t *testing.T) {
	a := &artifactsHandler{
		compartmentId: "compartmentId",
		client:        &MockArtifactsClient{},
	}

	{
		req := httptest.NewRequest("DELETE", "/repo/new-image", nil)
		w := httptest.NewRecorder()
		a.ServeHTTP(w, req)

		a.assertRequestCounts(t, 1, 0, 0, 0)
	}

	{
		req := httptest.NewRequest("DELETE", "/repo/existing-image", nil)
		w := httptest.NewRecorder()
		a.ServeHTTP(w, req)

		a.assertRequestCounts(t, 2, 0, 1, 0)
	}
}
