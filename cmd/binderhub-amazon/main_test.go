//go:build integration

package main

import (
	"io"
	"net/http"
	"strings"
	"testing"
	"time"

	"github.com/prometheus/common/expfmt"
)

func TestAmazonLocalstack(t *testing.T) {

	t.Setenv("BINDERHUB_AUTH_TOKEN", "token")
	go run([]string{})
	time.Sleep(2 * time.Second)

	{
		resp, err := http.Get("http://localhost:8080/health")
		if err != nil {
			t.Fatal(err)
		}
		defer resp.Body.Close()
		if resp.StatusCode != http.StatusOK {
			t.Fatalf("Expected status: %d, got %s", http.StatusForbidden, resp.Status)
		}
		body, err := io.ReadAll(resp.Body)
		if err != nil {
			t.Fatal(err)
		}
		t.Log(string(body))
	}

	{
		// No auth, should be forbidden
		resp, err := http.Get("http://localhost:8080/repos")
		if err != nil {
			t.Fatal(err)
		}
		defer resp.Body.Close()
		if resp.StatusCode != http.StatusForbidden {
			t.Fatalf("Expected status: %d, got %s", http.StatusForbidden, resp.Status)
		}
	}

	{
		// ECR is not supported in the free localstack so this should return an error
		req, err := http.NewRequest("GET", "http://localhost:8080/repos", nil)
		if err != nil {
			t.Fatal(err)
		}
		req.Header.Set("Authorization", "Bearer token")
		resp, err := http.DefaultClient.Do(req)
		if err != nil {
			t.Fatal(err)
		}
		defer resp.Body.Close()
		if resp.StatusCode != http.StatusInternalServerError {
			t.Fatalf("Expected status: %d, got %s", http.StatusInternalServerError, resp.Status)
		}
	}

	resp, err := http.Get("http://localhost:8080/metrics")
	if err != nil {
		t.Fatal(err)
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		t.Fatal(err)
	}

	parser := expfmt.TextParser{}
	metrics, err := parser.TextToMetricFamilies(strings.NewReader(string(body)))
	if err != nil {
		t.Fatal(err)
	}

	api_response_time_seconds := metrics["binderhub_container_registry_helper_api_response_time_seconds"]
	api_response_time_seconds_sample_count := api_response_time_seconds.GetMetric()[0].GetHistogram().GetSampleCount()
	if api_response_time_seconds_sample_count != 1 {
		t.Errorf("Expected binderhub_container_registry_helper_api_response_time_seconds 1, got %d", api_response_time_seconds_sample_count)
	}

	new_repositories_total := metrics["binderhub_container_registry_helper_new_repositories_total"]
	new_repositories_total_value := new_repositories_total.GetMetric()[0].GetCounter().GetValue()
	if new_repositories_total_value != 0 {
		t.Errorf("Expected binderhub_container_registry_helper_new_repositories_total 0, got %f", new_repositories_total_value)
	}
}
