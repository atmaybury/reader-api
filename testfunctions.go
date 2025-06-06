package main

import (
	"bytes"
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gorilla/mux"
	"github.com/pashagolub/pgxmock/v4"
)

// setupTestHandler creates a mock pool and handler for testi
func setupTestHandler(t *testing.T) *Handler {
	t.Helper()

	mockPool, err := pgxmock.NewPool()
	if err != nil {
		t.Fatalf("Failed to create mock pool: %v", err)
	}

	handler := &Handler{
		conn: mockPool,
	}

	return handler
}

func invalidMethod(t *testing.T, mux *mux.Router, method string, path string) {
	t.Helper()

	// Make request and response
	req := httptest.NewRequest(method, path, nil)
	w := httptest.NewRecorder()

	// Call handler
	mux.ServeHTTP(w, req)

	// Check response
	resp := w.Result()
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		t.Fatal(err)
	}

	expectedStatus := http.StatusMethodNotAllowed
	if resp.StatusCode != expectedStatus {
		t.Errorf("Expected status %d; got %v: %v", expectedStatus, resp.Status, string(body))
	}
}

type InvalidUserInput struct {
	Username string `json:"username"`
	Email    string `json:"email"`
}

func invalidUserInput(t *testing.T, mux *mux.Router, path string) {
	t.Helper()

	invalidUserInput := &InvalidUserInput{
		Username: "user",
		Email:    "test@email.com",
	}

	data, err := json.Marshal(invalidUserInput)
	requestBody := bytes.NewReader(data)

	// Make request and response
	req := httptest.NewRequest(http.MethodPost, path, requestBody)
	w := httptest.NewRecorder()

	// Call handler
	mux.ServeHTTP(w, req)

	// Check response
	resp := w.Result()
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		t.Fatal(err)
	}

	expectedStatus := http.StatusBadRequest
	if resp.StatusCode != expectedStatus {
		t.Errorf("Expected status %d; got %v: %v", expectedStatus, resp.Status, string(body))
	}
}

func missingAuthHeader(t *testing.T, mux *mux.Router, method string, path string) {
	t.Helper()

	// Make request and response
	req := httptest.NewRequest(method, path, nil)
	w := httptest.NewRecorder()

	// Call handler
	mux.ServeHTTP(w, req)

	// Check response
	resp := w.Result()
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		t.Fatal(err)
	}

	expectedStatus := http.StatusUnauthorized
	if resp.StatusCode != expectedStatus {
		t.Errorf("Expected status %d; got %v: %v", expectedStatus, resp.Status, string(body))
	}
}

func invalidAuthHeader(t *testing.T, mux *mux.Router, method string, path string) {
	t.Helper()

	// Make request and response
	req := httptest.NewRequest(method, path, nil)
	req.Header.Add("Authorization", "Bearer test")
	w := httptest.NewRecorder()

	// Call handler
	mux.ServeHTTP(w, req)

	// Check response
	resp := w.Result()
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		t.Fatal(err)
	}

	expectedStatus := http.StatusUnauthorized
	if resp.StatusCode != expectedStatus {
		t.Errorf("Expected status %d; got %v: %v", expectedStatus, resp.Status, string(body))
	}
}
