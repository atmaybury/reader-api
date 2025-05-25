package main

import (
	"net/http"
	"testing"
)

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
	method := http.MethodGet
	path := "/user-subscriptions"
	handler := setupTestHandler(t)
	mux := SetupRouter(handler)

	missingAuthHeader(t, mux, method, path)
	invalidAuthHeader(t, mux, method, path)
}

func TestHandleSearchSubscription(t *testing.T) {
	method := http.MethodGet
	path := "/search-subscription"
	handler := setupTestHandler(t)
	mux := SetupRouter(handler)

	missingAuthHeader(t, mux, method, path)
	invalidAuthHeader(t, mux, method, path)
}

func TestHandleAddSubscriptions(t *testing.T) {
	method := http.MethodPost
	path := "/add-subscriptions"
	handler := setupTestHandler(t)
	mux := SetupRouter(handler)

	invalidMethod(t, mux, http.MethodGet, path)
	missingAuthHeader(t, mux, method, path)
	invalidAuthHeader(t, mux, method, path)
}

func TestHandleDeleteSubscriptions(t *testing.T) {
	method := http.MethodDelete
	path := "/delete-subscriptions"
	handler := setupTestHandler(t)
	mux := SetupRouter(handler)

	invalidMethod(t, mux, http.MethodPost, path)
	missingAuthHeader(t, mux, method, path)
	invalidAuthHeader(t, mux, method, path)
}
