package eero

import (
	"context"
	"fmt"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

func TestSensitiveDataExposureInError(t *testing.T) {
	sensitiveData := "SENSITIVE_API_KEY_12345"

	// Mock server returns a non-JSON body containing sensitive data
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK) // Or any status code
		fmt.Fprintf(w, "Invalid JSON body with %s", sensitiveData)
	}))
	defer ts.Close()

	client, err := NewClient()
	if err != nil {
		t.Fatalf("Failed to create client: %v", err)
	}
	client.BaseURL = ts.URL

	// Trigger doRaw via a method that uses it, or access it if exported.
	// Since doRaw is unexported, we can use a service method that might use it,
	// or we can test `do` which has the same issue.
	// Looking at client.go, doRaw is used by generic EeroResponse[T].
	// However, do() also has the same vulnerability.
	// Let's try to trigger it via `Account.Get` which uses `do`.

	_, err = client.Account.Get(context.Background())
	if err == nil {
		t.Fatal("Expected error, got nil")
	}

	errMsg := err.Error()
	if strings.Contains(errMsg, sensitiveData) {
		t.Errorf("Vulnerability reproduced: error message contains sensitive data. Error: %s", errMsg)
	} else {
		t.Logf("Secure: error message does not contain sensitive data. Error: %s", errMsg)
	}

	expectedMsg := fmt.Sprintf("unparseable response body (%d bytes)", len("Invalid JSON body with "+sensitiveData))
	if !strings.Contains(errMsg, expectedMsg) {
		t.Errorf("Expected error message to contain '%s', got '%s'", expectedMsg, errMsg)
	}
}
