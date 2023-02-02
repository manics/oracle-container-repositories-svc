package utils

import (
	"fmt"
	"io/ioutil"
	"net/http"
	"net/http/httptest"
	"testing"
)

type mockHandler struct {
}

func (h *mockHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	w.WriteHeader(http.StatusOK)
	w.Write([]byte("hello!"))
}

func TestCheckAuthorised(t *testing.T) {
	testCases := []struct {
		authTokenEnvVar interface{}
		authToken       interface{}
		expectedStatus  interface{}
	}{
		{"", "ignored", 200},
		{"token", "token", 200},
		{"token", "incorrect", 403},
		{"token", "", 403},
		{"token", nil, 403},
		// TODO: Test unset AUTH_TOKEN (should fail and exit)
	}

	for _, tc := range testCases {
		t.Run(fmt.Sprintf("%v,%v,%v", tc.authTokenEnvVar, tc.authToken, tc.expectedStatus), func(t *testing.T) {
			if tc.authTokenEnvVar != nil {
				t.Setenv("AUTH_TOKEN", tc.authTokenEnvVar.(string))
			}

			a := CheckAuthorised(&mockHandler{})
			req := httptest.NewRequest("GET", "/", nil)
			if tc.authToken != nil {
				req.Header.Set("Authorization", fmt.Sprintf("Bearer %s", tc.authToken))
			}

			w := httptest.NewRecorder()
			a.ServeHTTP(w, req)
			res := w.Result()
			defer res.Body.Close()
			_, err := ioutil.ReadAll(w.Result().Body)
			if err != nil {
				t.Errorf("Unexpected error: %v", err)
			}

			if res.StatusCode != tc.expectedStatus {
				t.Errorf("Expected StatusCode %v: %v", tc.expectedStatus, res.StatusCode)
			}
		})
	}
}

func TestGetName(t *testing.T) {
	{
		req := httptest.NewRequest("GET", "/existing-image", nil)
		_, err := RepoGetName(req)
		if err == nil {
			t.Errorf("Expected error: %v", err)
		}
	}

	{
		req := httptest.NewRequest("GET", "/repo/existing-image", nil)
		r, err := RepoGetName(req)
		if err != nil {
			t.Errorf("Unexpected error: %v", err)
		}
		if r != "existing-image" {
			t.Errorf("Unexpected repository name: %v", r)
		}
	}
}

func TestImageGetNameAndTag(t *testing.T) {
	{
		req := httptest.NewRequest("GET", "/image/existing-image", nil)
		name, tag, err := ImageGetNameAndTag(req)
		if err != nil {
			t.Errorf("Unexpected error: %v", err)
		}
		if name != "existing-image" {
			t.Errorf("Unexpected image name: %v", name)
		}
		if tag != "latest" {
			t.Errorf("Unexpected image tag: %v", tag)
		}
	}

	{
		req := httptest.NewRequest("GET", "/image/existing-image:tag", nil)
		name, tag, err := ImageGetNameAndTag(req)
		if err != nil {
			t.Errorf("Unexpected error: %v", err)
		}
		if name != "existing-image" {
			t.Errorf("Unexpected image name: %v", name)
		}
		if tag != "tag" {
			t.Errorf("Unexpected image tag: %v", tag)
		}
	}
}
