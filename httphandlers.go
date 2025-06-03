package main

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/url"
	"time"

	"github.com/golang-jwt/jwt"
	"github.com/gorilla/mux"
	"github.com/jackc/pgx/v5"
	"github.com/mmcdole/gofeed"
	"golang.org/x/crypto/bcrypt"
	"golang.org/x/net/html"
)

type UserLoginInput struct {
	Email    string `json:"email"`
	Password string `json:"password"`
}

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

type FeedTag struct {
	Title string `json:"title"`
	Href  string `json:"href"`
}

type FeedResponse struct {
	Title       string     `json:"title"`
	Description string     `json:"description"`
	Items       []FeedItem `json:"items"`
}

type FeedItem struct {
	Title       string `json:"title"`
	Content     string `json:"content"`
	Description string `json:"description"`
}

type UserFolder struct {
	Id   string `json:"id"`
	Name string `json:"name"`
}

type UserSubscription struct {
	Id          int       `json:"id"`
	Title       string    `json:"title"`
	Url         string    `json:"url"`
	LastChecked time.Time `json:"last_checked"`
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

func (h *Handler) handleDeleteUser(w http.ResponseWriter, r *http.Request) {
	// Get URL for querying
	id := r.URL.Query().Get("id")
	if id == "" {
		http.Error(w, "Missing id parameter", http.StatusBadRequest)
		return
	}

	if _, err := h.conn.Exec(
		context.Background(),
		"DELETE FROM users WHERE id = $1",
		id,
	); err != nil {
		http.Error(w, "Error deleting user from DB", http.StatusInternalServerError)
		fmt.Println(err)
		return
	}
}

func (h *Handler) handleLogin(w http.ResponseWriter, r *http.Request) {
	// Parse the JSON request body
	var userInput UserLoginInput
	if err := json.NewDecoder(r.Body).Decode(&userInput); err != nil {
		http.Error(w, "Invalid request body", http.StatusBadRequest)
		return
	}

	fmt.Printf("Logging in %s\n", userInput.Email)

	// Validate required fields
	if userInput.Email == "" || userInput.Password == "" {
		http.Error(w, "Missing required fields (email, password)", http.StatusBadRequest)
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

func (h *Handler) handleCreateUserFolder(w http.ResponseWriter, r *http.Request) {
	// Get user token
	userToken, ok := r.Context().Value(userTokenKey).(*Token)
	if !ok {
		http.Error(w, "No claims found in context", http.StatusForbidden)
		return
	}

	// Get folder name from request
	folderName := r.URL.Query().Get("name")
	if folderName == "" {
		http.Error(w, fmt.Sprintf("Missing name parameter: %v", r.Method), http.StatusBadRequest)
		return
	}

	// Add user row to db
	var folder UserFolder
	query := `
        INSERT INTO folders (user_id, name)
        VALUES ($1, $2)
        RETURNING id, name
    `
	if err := h.conn.QueryRow(context.Background(), query, userToken.Id, folderName).Scan(&folder.Id, &folder.Name); err != nil {
		http.Error(w, fmt.Sprintf("Error adding folder to database: %v", err), http.StatusBadRequest)
		return
	}

	json.NewEncoder(w).Encode(folder)
}

func (h *Handler) handleGetUserFolders(w http.ResponseWriter, r *http.Request) {
	// Get user token
	userToken, ok := r.Context().Value(userTokenKey).(*Token)
	if !ok {
		http.Error(w, "No claims found in context", http.StatusForbidden)
		return
	}

	userFolders := []UserFolder{}
	rows, err := h.conn.Query(
		context.Background(),
		"SELECT id, name FROM folders f WHERE user_id = $1",
		userToken.Id,
	)
	if err != nil {
		http.Error(w, fmt.Sprintf("Error getting folders for user %s: %v", userToken.Id, err), http.StatusInternalServerError)
		return
	}
	defer rows.Close()

	for rows.Next() {
		var folder UserFolder
		if err := rows.Scan(&folder.Id, &folder.Name); err != nil {
			http.Error(w, fmt.Sprintf("Error scanning folders row: %v", err), http.StatusInternalServerError)
			return
		}
		userFolders = append(userFolders, folder)
	}
	if err := rows.Err(); err != nil {
		http.Error(w, fmt.Sprintf("Error iterating over subscriptions: %v", err), http.StatusInternalServerError)
		return
	}

	fmt.Println("USER FOLDERS: ", userFolders)

	json.NewEncoder(w).Encode(userFolders)
}

func (h *Handler) handleGetFolderSubscriptions(w http.ResponseWriter, r *http.Request) {
	// Get user token
	userToken, ok := r.Context().Value(userTokenKey).(*Token)
	if !ok {
		http.Error(w, "No claims found in context", http.StatusForbidden)
		return
	}

	vars := mux.Vars(r)
	folderId := vars["folderId"]

	var userSubscriptions []UserSubscription
	rows, err := h.conn.Query(
		context.Background(),
		"SELECT s.id, f.title, f.url FROM subscriptions s LEFT JOIN feeds f ON f.id = s.feed_id WHERE user_id = $1 AND folder_id = $2",
		userToken.Id, folderId,
	)
	if err != nil {
		http.Error(w, fmt.Sprintf("Error getting subscriptions for user %s: %v", userToken.Id, err), http.StatusInternalServerError)
		return
	}
	defer rows.Close()

	for rows.Next() {
		var sub UserSubscription
		if err := rows.Scan(&sub.Id, &sub.Title, &sub.Url); err != nil {
			http.Error(w, fmt.Sprintf("Error scanning subscription row: %v", err), http.StatusInternalServerError)
			return
		}
		userSubscriptions = append(userSubscriptions, sub)
	}
	if err := rows.Err(); err != nil {
		http.Error(w, fmt.Sprintf("Error iterating over subscriptions: %v", err), http.StatusInternalServerError)
		return
	}

	json.NewEncoder(w).Encode(userSubscriptions)
}

func (h *Handler) handleGetUserSubscriptions(w http.ResponseWriter, r *http.Request) {
	// Get user token
	userToken, ok := r.Context().Value(userTokenKey).(*Token)
	if !ok {
		http.Error(w, "No claims found in context", http.StatusForbidden)
		return
	}

	var userSubscriptions []UserSubscription
	rows, err := h.conn.Query(
		context.Background(),
		"SELECT s.id, f.title, f.url FROM subscriptions s LEFT JOIN feeds f ON f.id = s.feed_id WHERE user_id = $1",
		userToken.Id,
	)
	if err != nil {
		http.Error(w, fmt.Sprintf("Error getting subscriptions for user %s: %v", userToken.Id, err), http.StatusInternalServerError)
		return
	}
	defer rows.Close()

	for rows.Next() {
		var sub UserSubscription
		if err := rows.Scan(&sub.Id, &sub.Title, &sub.Url); err != nil {
			http.Error(w, fmt.Sprintf("Error scanning subscription row: %v", err), http.StatusInternalServerError)
			return
		}
		userSubscriptions = append(userSubscriptions, sub)
	}
	if err := rows.Err(); err != nil {
		http.Error(w, fmt.Sprintf("Error iterating over subscriptions: %v", err), http.StatusInternalServerError)
		return
	}

	json.NewEncoder(w).Encode(userSubscriptions)
}

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
	var feeds []FeedTag

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

	// Feed tags from the request
	var feeds []FeedTag
	if err := json.NewDecoder(r.Body).Decode(&feeds); err != nil {
		http.Error(w, fmt.Sprintf("Error parsing html response: %v", err), http.StatusInternalServerError)
		return
	}

	var newFeeds []int

	addFeedQuery := `
    INSERT INTO feeds (url, title)
    VALUES (@url, @title)
    ON CONFLICT (url) DO UPDATE SET title = EXCLUDED.title
    RETURNING id
    `

	for _, feedURL := range feeds {
		args := pgx.NamedArgs{
			"url":   feedURL.Href,
			"title": feedURL.Title,
		}

		var feedID int
		if err := h.conn.QueryRow(context.Background(), addFeedQuery, args).Scan(&feedID); err != nil {
			http.Error(w, fmt.Sprintf("Error adding feed to database: %v", err), http.StatusInternalServerError)
			return
		}

		newFeeds = append(newFeeds, feedID)
	}

	// Newly created subscriptions with ids
	var newSubscriptions []UserSubscription

	addSubscriptionQuery := `
    WITH inserted_sub AS (
        INSERT INTO subscriptions (user_id, feed_id)
        VALUES (@user_id, @feed_id)
        RETURNING id, user_id, feed_id
    )
    SELECT s.id, f.title, f.url, f.last_checked
    FROM inserted_sub s
    JOIN feeds f ON s.feed_id = f.id
    `

	for _, feedId := range newFeeds {
		args := pgx.NamedArgs{
			"user_id": userToken.Id,
			"feed_id": feedId,
		}
		var returnedSubscription UserSubscription
		if err := h.conn.QueryRow(
			context.Background(), addSubscriptionQuery, args,
		).Scan(
			&returnedSubscription.Id, &returnedSubscription.Title, &returnedSubscription.Url, &returnedSubscription.LastChecked,
		); err != nil {
			http.Error(w, fmt.Sprintf("Error adding subscription to database: %v", err), http.StatusInternalServerError)
			return
		}
		newSubscriptions = append(newSubscriptions, returnedSubscription)
	}

	json.NewEncoder(w).Encode(newSubscriptions)
}

func (h *Handler) handleDeleteSubscriptions(w http.ResponseWriter, r *http.Request) {
	// Get user token
	userToken, ok := r.Context().Value(userTokenKey).(*Token)
	if !ok {
		http.Error(w, "No claims found in context", http.StatusForbidden)
		return
	}

	// Subscription tags from the request
	var ids []int
	if err := json.NewDecoder(r.Body).Decode(&ids); err != nil {
		http.Error(w, fmt.Sprintf("Error parsing html response: %v", err), http.StatusInternalServerError)
		return
	}

	// Delete from db
	query := `
    DELETE FROM subscriptions
    WHERE id = ANY($1)
        AND user_id = $2
	RETURNING id
	`

	rows, err := h.conn.Query(context.Background(), query, ids, userToken.Id)
	if err != nil {
		http.Error(w, fmt.Sprintf("Error deleting subscriptions: %v", err), http.StatusInternalServerError)
		return
	}
	defer rows.Close()

	var deletedIDs []int
	for rows.Next() {
		var id int
		rows.Scan(&id)
		deletedIDs = append(deletedIDs, id)
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(deletedIDs)
}

func (h *Handler) handleFetchFeed(w http.ResponseWriter, r *http.Request) {
	// Get URL for querying
	href := r.URL.Query().Get("href")
	if href == "" {
		http.Error(w, fmt.Sprintf("Missing href parameter: %v", r.Method), http.StatusBadRequest)
		return
	}

	// Fetch feed
	fp := gofeed.NewParser()
	feed, err := fp.ParseURL(href)
	if err != nil {
		http.Error(w, fmt.Sprintf("Error fetching feed: %v", err), http.StatusInternalServerError)
		return
	}

	// Create response
	var items []FeedItem
	for _, item := range feed.Items {
		newItem := FeedItem{
			Title:       item.Title,
			Content:     item.Content,
			Description: item.Description,
		}
		items = append(items, newItem)
	}

	feedResponse := FeedResponse{
		Title:       feed.Title,
		Description: feed.Description,
		Items:       items,
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(feedResponse)
}
