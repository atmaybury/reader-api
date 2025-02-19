package main

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/url"
	"strings"

	"github.com/jackc/pgx/v5"
	"golang.org/x/crypto/bcrypt"
	"golang.org/x/net/html"
)

type UserInput struct {
	Username string `json:"username"`
	Email    string `json:"email"`
	Password string `json:"password"`
}

type User struct {
	Id       string `json:"id"`
	Username string `json:"username"`
	Email    string `json:"email"`
	Password string `json:"password"`
}

type Subscription struct {
	title string
	href  string
}

// Middleware to print the Authorization header
func authMiddleware(next http.HandlerFunc) http.HandlerFunc {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Get the Authorization header
		authHeader := r.Header.Get("Authorization")
		if authHeader == "" {
			http.Error(w, "Invalid auth header", http.StatusForbidden)
			return
		}

		tokenString := strings.TrimPrefix(authHeader, "Bearer ")

		_, err := validateJWT(tokenString)
		if err != nil {
			http.Error(w, fmt.Sprintf("Error validating JWT: %v", err), http.StatusInternalServerError)
		}

		// Pass the request to the next handler
		next.ServeHTTP(w, r)
	})
}

// Sends 200
func (h *Handler) handleRoot(w http.ResponseWriter, r *http.Request) {}

func (h *Handler) handleRegisterUser(w http.ResponseWriter, r *http.Request) {
	// Check request method
	if r.Method != http.MethodPost {
		http.Error(w, fmt.Sprintf("Method not allowed: %v", r.Method), http.StatusMethodNotAllowed)
		return
	}

	// Parse the JSON request body
	var userInput UserInput
	if err := json.NewDecoder(r.Body).Decode(&userInput); err != nil {
		http.Error(w, "Invalid request body", http.StatusBadRequest)
		return
	}

	// Validate required fields
	if userInput.Username == "" || userInput.Email == "" || userInput.Password == "" {
		http.Error(w, "Missing required fields (username, email, password)", http.StatusBadRequest)
		return
	}

	// Check if a user with the same email already exists
	var exists bool
	if err := h.conn.QueryRow(
		context.Background(),
		"SELECT EXISTS(SELECT 1 FROM users WHERE email = $1)",
		userInput.Email,
	).Scan(&exists); err != nil {
		http.Error(w, "Error checking for existing user", http.StatusInternalServerError)
		return
	}
	if exists {
		http.Error(w, "A user with this email already exists", http.StatusConflict)
		return
	}

	// Create password hash
	hashedPassword, err := bcrypt.GenerateFromPassword([]byte(userInput.Password), bcrypt.DefaultCost)
	if err != nil {
		http.Error(w, "Failed to hash password", http.StatusInternalServerError)
		return
	}

	// Add user row to db
	var user User
	query := `
        INSERT INTO users (username, email, password)
        VALUES (@username, @email, @password)
        RETURNING id, username, email
    `
	args := pgx.NamedArgs{
		"username": userInput.Username,
		"email":    userInput.Email,
		"password": hashedPassword,
	}
	if err = h.conn.QueryRow(context.Background(), query, args).Scan(&user.Id, &user.Username, &user.Email); err != nil {
		http.Error(w, fmt.Sprintf("Error adding user to database: %v", err), http.StatusBadRequest)
		return
	}

	// create JWT
	token, err := generateJWT(user)
	if err != nil {
		http.Error(w, fmt.Sprintf("Error generating token: %v", err), http.StatusBadRequest)
		return
	}

	// send back token
	fmt.Fprint(w, token)
}

func (h *Handler) handleLogin(w http.ResponseWriter, r *http.Request) {
	// Check request method
	if r.Method != http.MethodPost {
		http.Error(w, fmt.Sprintf("Method not allowed: %v", r.Method), http.StatusMethodNotAllowed)
		return
	}

	// Parse the JSON request body
	var userInput UserInput
	if err := json.NewDecoder(r.Body).Decode(&userInput); err != nil {
		http.Error(w, "Invalid request body", http.StatusBadRequest)
		return
	}

	// Validate required fields
	if userInput.Username == "" || userInput.Email == "" || userInput.Password == "" {
		http.Error(w, "Missing required fields (username, email, password)", http.StatusBadRequest)
		return
	}

	// Check if a user with the same email already exists
	var user User
	if err := h.conn.QueryRow(
		context.Background(),
		"SELECT id, username, email, password FROM users WHERE email = $1",
		userInput.Email,
	).Scan(&user.Id, &user.Username, &user.Email, &user.Password); err != nil {
		http.Error(w, "Error checking for existing user", http.StatusInternalServerError)
		return
	}

	// Compare user password input to hashed password
	if err := bcrypt.CompareHashAndPassword([]byte(user.Password), []byte(userInput.Password)); err != nil {
		http.Error(w, "Invalid username or password", http.StatusUnauthorized)
		return
	}

	// create JWT
	token, err := generateJWT(user)
	if err != nil {
		http.Error(w, fmt.Sprintf("Error generating token: %v", err), http.StatusBadRequest)
		return
	}

	// send back token
	fmt.Fprint(w, token)
}

// Given a URL, find any rss links and save them
func (h *Handler) handleAddSubscription(w http.ResponseWriter, r *http.Request) {
	// Check request method
	if r.Method != http.MethodGet {
		http.Error(w, fmt.Sprintf("Method not allowed: %v", r.Method), http.StatusMethodNotAllowed)
		return
	}

	// Get URL for querying
	inputURL := r.URL.Query().Get("url")
	if inputURL == "" {
		http.Error(w, fmt.Sprintf("Missing url parameter: %v", r.Method), http.StatusBadRequest)
		return
	}

	// Validate url param
	parsedURL, err := url.ParseRequestURI(inputURL)
	if err != nil {
		http.Error(w, "Invalid URL", http.StatusBadRequest)
		return
	}

	// Make GET request to the URL
	resp, err := http.Get(parsedURL.String())
	if err != nil {
		http.Error(w, fmt.Sprintf("Error making GET request to %s: %v", parsedURL, err), http.StatusBadRequest)
		return
	}
	defer resp.Body.Close()

	// Check status code
	if resp.StatusCode != http.StatusOK {
		http.Error(w, fmt.Sprintf("Received %d response", resp.StatusCode), http.StatusInternalServerError)
		return
	}

	// Parse html
	doc, err := html.Parse(resp.Body)
	if err != nil {
		http.Error(w, fmt.Sprintf("Error parsing html response: %v", err), http.StatusInternalServerError)
		return
	}

	// slice of rss urls
	var urls []Subscription

	// Traverse the HTML document
	findFeedLinks(doc, &urls)

	if len(urls) == 0 {
		http.Error(w, fmt.Sprint("No feed URLs found"), http.StatusInternalServerError)
		return
	}

	// TODO save links
	query := `
        INSERT INTO subscriptions (user_id, title, url)
        VALUES (@user_id, @title, @url)
    `
	for _, feedURL := range urls {
		args := pgx.NamedArgs{
			"user_id": 7,
			"title":   feedURL.title,
			"url":     feedURL.href,
		}
		if _, err := h.conn.Exec(context.Background(), query, args); err != nil {
			http.Error(w, fmt.Sprintf("Error adding subscription to database: %v", err), http.StatusInternalServerError)
			return
		}
	}

}
