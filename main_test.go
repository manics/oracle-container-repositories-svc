package main

import (
	"fmt"
	"testing"
)

func TestAuthToken(t *testing.T) {
	testCases := []struct {
		authTokenEnvVar interface{}
		shouldError     bool
	}{
		{"token", false},
		{"", false},
		{nil, true},
	}
	for _, tc := range testCases {
		t.Run(fmt.Sprintf("%v,%v", tc.authTokenEnvVar, tc.shouldError), func(t *testing.T) {
			if tc.authTokenEnvVar != nil {
				t.Setenv("AUTH_TOKEN", tc.authTokenEnvVar.(string))
			}

			authToken, err := getAuthToken()

			if tc.shouldError {
				if err == nil {
					t.Errorf("Expected error: %v", err)
				}
			} else {
				if err != nil {
					t.Errorf("Unexpected error: %v", err)
				}
				if authToken != tc.authTokenEnvVar {
					t.Errorf("Expected %v: %v", tc.authTokenEnvVar, authToken)
				}
			}
		})
	}
}
