package main

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
	listRequests   []artifacts.ListContainerRepositoriesRequest
	createRequests []artifacts.CreateContainerRepositoryRequest
	deleteRequests []artifacts.DeleteContainerRepositoryRequest
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

func (a *artifactsHandler) assertRequestCounts(t *testing.T, expectedLists int, expectedCreates int, expectedDeletes int) {
	nLists := len(a.client.(*MockArtifactsClient).listRequests)
	nCreates := len(a.client.(*MockArtifactsClient).createRequests)
	nDeletes := len(a.client.(*MockArtifactsClient).deleteRequests)
	if nLists != expectedLists {
		t.Errorf("Expected %v list request: %v", expectedLists, nLists)
	}
	if nCreates != expectedCreates {
		t.Errorf("Expected %v create request: %v", expectedCreates, nCreates)
	}
	if nDeletes != expectedDeletes {
		t.Errorf("Expected %v delete request: %v", expectedDeletes, nDeletes)
	}
}

// Tests

func TestGetByName(t *testing.T) {
	a := &artifactsHandler{
		compartmentId: "compartmentId",
		client:        &MockArtifactsClient{},
	}

	{
		_, _, err := a.getByName("existing-image")
		if err == nil {
			t.Errorf("Expected error: %v", err)
		}
	}

	{
		container, name, err := a.getByName("/repo/existing-image")
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
		container, name, err := a.getByName("/repo/new-image")
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

func TestList(t *testing.T) {
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

	a.assertRequestCounts(t, 1, 0, 0)

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

	a.assertRequestCounts(t, 1, 1, 0)
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

		a.assertRequestCounts(t, 1, 0, 0)
	}

	{
		req := httptest.NewRequest("DELETE", "/repo/existing-image", nil)
		w := httptest.NewRecorder()
		a.ServeHTTP(w, req)

		a.assertRequestCounts(t, 2, 0, 1)
	}
}

// TODO: Test auth token
