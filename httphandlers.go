package main

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/url"

	"github.com/golang-jwt/jwt"
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
	Id       string
	Username string
	Email    string
	Password string
}

type SubscriptionTag struct {
	Title string `json:"title"`
	Href  string `json:"href"`
}

type Subscription struct {
	Id    int    `json:"id"`
	Title string `json:"title"`
	Url   string `json:"url"`
}

type Token struct {
	Id       string `json:"id"`
	Username string `json:"username"`
	Email    string `json:"email"`
	Exp      int64  `json:"exp"`
	jwt.MapClaims
}

type TokenKey string

const (
	userTokenKey TokenKey = "usertoken"
)

// Sends 200
func (h *Handler) handleRoot(w http.ResponseWriter, r *http.Request) {}

func (h *Handler) handleRegisterUser(w http.ResponseWriter, r *http.Request) {
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
	tokenString, err := generateJWT(user)
	if err != nil {
		http.Error(w, fmt.Sprintf("Error generating token: %v", err), http.StatusBadRequest)
		return
	}

	// send back token
	fmt.Fprint(w, tokenString)
}

func (h *Handler) handleLogin(w http.ResponseWriter, r *http.Request) {
	// Parse the JSON request body
	var userInput UserInput
	if err := json.NewDecoder(r.Body).Decode(&userInput); err != nil {
		http.Error(w, "Invalid request body", http.StatusBadRequest)
		return
	}

	fmt.Printf("Logging in %s\n", userInput.Email)

	// Validate required fields
	if userInput.Username == "" || userInput.Email == "" || userInput.Password == "" {
		http.Error(w, "Missing required fields (username, email, password)", http.StatusBadRequest)
		return
	}

	// Get user by email
	var user User
	if err := h.conn.QueryRow(
		context.Background(),
		"SELECT id, username, email, password FROM users WHERE email = $1",
		userInput.Email,
	).Scan(&user.Id, &user.Username, &user.Email, &user.Password); err != nil {
		http.Error(w, "Error getting user from DB", http.StatusInternalServerError)
		fmt.Println(err)
		return
	}

	// Compare user password input to hashed password
	if err := bcrypt.CompareHashAndPassword([]byte(user.Password), []byte(userInput.Password)); err != nil {
		http.Error(w, "Invalid username or password", http.StatusUnauthorized)
		return
	}

	// create JWT
	tokenString, err := generateJWT(user)
	if err != nil {
		http.Error(w, fmt.Sprintf("Error generating token: %v", err), http.StatusBadRequest)
		return
	}

	// send back token
	fmt.Fprint(w, tokenString)
}

func (h *Handler) handleGetUserSubscriptions(w http.ResponseWriter, r *http.Request) {
	// Get user token
	userToken, ok := r.Context().Value(userTokenKey).(*Token)
	if !ok {
		http.Error(w, "No claims found in context", http.StatusForbidden)
		return
	}

	var subscriptions []Subscription
	rows, err := h.conn.Query(
		context.Background(),
		"SELECT id, title, url FROM subscriptions WHERE user_id = $1",
		userToken.Id,
	)
	if err != nil {
		http.Error(w, fmt.Sprintf("Error getting subscriptions for user %s", userToken.Id), http.StatusForbidden)
		return
	}
	defer rows.Close()

	for rows.Next() {
		var sub Subscription
		if err := rows.Scan(&sub.Id, &sub.Title, &sub.Url); err != nil {
			http.Error(w, "Error scanning subscription row", http.StatusInternalServerError)
			return
		}
		subscriptions = append(subscriptions, sub)
	}
	if err := rows.Err(); err != nil {
		http.Error(w, "Error iterating over subscriptions", http.StatusInternalServerError)
		return
	}

	json.NewEncoder(w).Encode(subscriptions)
}

// Given a URL, find any rss links and save them
// func (h *Handler) handleAddSubscription(w http.ResponseWriter, r *http.Request) {
// 	// Get user token
// 	userToken, ok := r.Context().Value(userTokenKey).(*Token)
// 	if !ok {
// 		http.Error(w, "No claims found in context", http.StatusForbidden)
// 		return
// 	}

// 	// Get URL for querying
// 	inputURL := r.URL.Query().Get("url")
// 	if inputURL == "" {
// 		http.Error(w, fmt.Sprintf("Missing url parameter: %v", r.Method), http.StatusBadRequest)
// 		return
// 	}

// 	// Validate url param
// 	parsedURL, err := url.ParseRequestURI(inputURL)
// 	if err != nil {
// 		http.Error(w, "Invalid URL", http.StatusBadRequest)
// 		return
// 	}

// 	// Make GET request to the URL
// 	resp, err := http.Get(parsedURL.String())
// 	if err != nil {
// 		http.Error(w, fmt.Sprintf("Error making GET request to %s: %v", parsedURL, err), http.StatusBadRequest)
// 		return
// 	}
// 	defer resp.Body.Close()

// 	// Check status code
// 	if resp.StatusCode != http.StatusOK {
// 		http.Error(w, fmt.Sprintf("Received %d response", resp.StatusCode), http.StatusInternalServerError)
// 		return
// 	}

// 	// Parse html
// 	doc, err := html.Parse(resp.Body)
// 	if err != nil {
// 		http.Error(w, fmt.Sprintf("Error parsing html response: %v", err), http.StatusInternalServerError)
// 		return
// 	}

// 	// slice of rss urls
// 	var feeds []SubscriptionTag

// 	// Traverse the HTML document
// 	findFeedLinks(doc, &feeds)

// 	if len(feeds) == 0 {
// 		http.Error(w, fmt.Sprint("No feed URLs found"), http.StatusInternalServerError)
// 		return
// 	}

// 	var newSubscriptions []Subscription

// 	// Save to db
// 	query := `
//         INSERT INTO subscriptions (user_id, title, url)
//         VALUES (@user_id, @title, @url)
//         RETURNING id, title, url
//     `
// 	for _, feedURL := range feeds {
// 		args := pgx.NamedArgs{
// 			"user_id": userToken.Id,
// 			"title":   feedURL.Title,
// 			"url":     feedURL.Href,
// 		}
// 		var returnedSubscription Subscription
// 		if err = h.conn.QueryRow(
// 			context.Background(), query, args,
// 		).Scan(
// 			&returnedSubscription.Id, &returnedSubscription.Title, &returnedSubscription.Url,
// 		); err != nil {
// 			http.Error(w, fmt.Sprintf("Error adding subscription to database: %v", err), http.StatusBadRequest)
// 			return
// 		}
// 		newSubscriptions = append(newSubscriptions, returnedSubscription)
// 	}

// 	json.NewEncoder(w).Encode(newSubscriptions)
// }

func (h *Handler) handleSearchSubscription(w http.ResponseWriter, r *http.Request) {
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

	// Slice of rss urls
	var feeds []SubscriptionTag

	// Traverse the HTML document
	findFeedLinks(doc, &feeds)

	if len(feeds) == 0 {
		http.Error(w, fmt.Sprint("No feed URLs found"), http.StatusInternalServerError)
		return
	}

	json.NewEncoder(w).Encode(feeds)
}

func (h *Handler) handleAddSubscriptions(w http.ResponseWriter, r *http.Request) {
	// Get user token
	userToken, ok := r.Context().Value(userTokenKey).(*Token)
	if !ok {
		http.Error(w, "No claims found in context", http.StatusForbidden)
		return
	}

	// Subscription tags from the request
	var feeds []SubscriptionTag
	if err := json.NewDecoder(r.Body).Decode(&feeds); err != nil {
		http.Error(w, fmt.Sprintf("Error parsing html response: %v", err), http.StatusInternalServerError)
		return
	}

	fmt.Println(feeds)

	// Newly created subscriptions with ids
	var newSubscriptions []Subscription

	// Save to db
	query := `
	INSERT INTO subscriptions (user_id, title, url)
	VALUES (@user_id, @title, @url)
	RETURNING id, title, url
	`
	for _, feedURL := range feeds {
		args := pgx.NamedArgs{
			"user_id": userToken.Id,
			"title":   feedURL.Title,
			"url":     feedURL.Href,
		}
		var returnedSubscription Subscription
		if err := h.conn.QueryRow(
			context.Background(), query, args,
		).Scan(
			&returnedSubscription.Id, &returnedSubscription.Title, &returnedSubscription.Url,
		); err != nil {
			http.Error(w, fmt.Sprintf("Error adding subscription to database: %v", err), http.StatusBadRequest)
			return
		}
		newSubscriptions = append(newSubscriptions, returnedSubscription)
	}

	json.NewEncoder(w).Encode(newSubscriptions)
}
