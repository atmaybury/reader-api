package main

import (
	"context"
	"fmt"
	"net/http"
	"strings"
)

func corsMiddleware(next http.HandlerFunc) http.HandlerFunc {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Set CORS headers
		w.Header().Set("Access-Control-Allow-Origin", "*")
		w.Header().Set("Access-Control-Allow-Methods", "GET, POST, OPTIONS")
		w.Header().Set("Access-Control-Allow-Headers", "Content-Type, Authorization")

		// Handle preflight request
		if r.Method == http.MethodOptions {
			w.WriteHeader(http.StatusNoContent)
			return
		}

		next.ServeHTTP(w, r)
	})
}

// Middleware to print the Authorization header
func authMiddleware(next http.HandlerFunc) http.HandlerFunc {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Get the Authorization header
		authHeader := r.Header.Get("Authorization")
		if authHeader == "" {
			http.Error(w, "Invalid auth header", http.StatusUnauthorized)
			return
		}

		tokenString := strings.TrimPrefix(authHeader, "Bearer ")

		token, err := validateJWT(tokenString)
		if err != nil {
			http.Error(w, fmt.Sprintf("Error validating JWT: %v", err), http.StatusUnauthorized)
			return
		}

		// Context to hold token
		ctx := context.WithValue(r.Context(), userTokenKey, token)

		// Pass the request to the next handler
		next.ServeHTTP(w, r.WithContext(ctx))
	})
}
