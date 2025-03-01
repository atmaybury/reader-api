package main

import (
	"bytes"
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/pashagolub/pgxmock/v4"
)

// setupTestHandler creates a mock pool and handler for testing
func setupTestHandler(t *testing.T) *Handler {
	t.Helper() // Marks this as a helper function so errors show the correct line number

	mockPool, err := pgxmock.NewPool()
	if err != nil {
		t.Fatalf("Failed to create mock pool: %v", err)
	}

	handler := &Handler{
		conn: mockPool,
	}

	return handler
}

func invalidMethod(t *testing.T, mux *http.ServeMux, method string, path string) {
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

	if resp.StatusCode != http.StatusMethodNotAllowed {
		t.Errorf("Expected status %d; got %v: %v", http.StatusMethodNotAllowed, resp.Status, string(body))
	}
}

type InvalidUserInput struct {
	Username string `json:"username"`
	Email    string `json:"email"`
}

func invalidUserInput(t *testing.T, mux *http.ServeMux, path string) {
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

	if resp.StatusCode != http.StatusBadRequest {
		t.Errorf("Expected status %d; got %v: %v", http.StatusMethodNotAllowed, resp.Status, string(body))
	}

}

func TestHandleLogin(t *testing.T) {
	path := "/login"
	handler := setupTestHandler(t)
	mux := SetupRouter(handler)

	invalidMethod(t, mux, http.MethodGet, path)
	invalidUserInput(t, mux, path)
}

func TestHandleRegister(t *testing.T) {
	path := "/register"
	handler := setupTestHandler(t)
	mux := SetupRouter(handler)

	invalidMethod(t, mux, http.MethodGet, path)
	invalidUserInput(t, mux, path)
}

func TestHandleGetUserSubscriptions(t *testing.T) {
	path := "/user-subscriptions"
	handler := setupTestHandler(t)
	mux := SetupRouter(handler)

	invalidMethod(t, mux, http.MethodPost, path)
}
