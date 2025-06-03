package main

import (
	"context"
	"fmt"
	"log"
	"net/http"
	"os"

	"github.com/gorilla/mux"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
	"github.com/joho/godotenv"
)

// TODO
//  link finder - exit if </head> reached
// tests

const (
	port = 8080
)

// Interface to allow using pgxmock in Handler
type PgxInterface interface {
	Exec(ctx context.Context, sql string, arguments ...any) (pgconn.CommandTag, error)
	Query(ctx context.Context, sql string, args ...any) (pgx.Rows, error)
	QueryRow(ctx context.Context, sql string, args ...any) pgx.Row
}

type Handler struct {
	conn PgxInterface
}

func SetupRouter(h *Handler) *mux.Router {
	r := mux.NewRouter()

	/* USERS */

	root := r.HandleFunc("/", corsMiddleware(h.handleRoot))
	root.Methods(http.MethodGet)

	registerUser := r.HandleFunc("/register", corsMiddleware(h.handleRegisterUser))
	registerUser.Methods(http.MethodPost, http.MethodOptions)

	deleteUser := r.HandleFunc("/delete-user", corsMiddleware(authMiddleware(h.handleDeleteUser)))
	deleteUser.Methods(http.MethodDelete, http.MethodOptions)

	login := r.HandleFunc("/login", corsMiddleware(h.handleLogin))
	login.Methods(http.MethodPost, http.MethodOptions)

	/* FOLDERS */

	// Get folders
	getUserFolders := r.HandleFunc("/user-folders", corsMiddleware(authMiddleware(h.handleGetUserFolders)))
	getUserFolders.Methods(http.MethodGet, http.MethodOptions)

	createUserFolder := r.HandleFunc("/user-folders", corsMiddleware(authMiddleware(h.handleCreateUserFolder)))
	createUserFolder.Methods(http.MethodPost, http.MethodOptions)

	/* SUBSCRIPTIONS & FEEDS */

	usersubscriptions := r.HandleFunc("/user-subscriptions", corsMiddleware(authMiddleware(h.handleGetUserSubscriptions)))
	usersubscriptions.Methods(http.MethodGet, http.MethodOptions)

	searchSubscription := r.HandleFunc("/search-subscription", corsMiddleware(authMiddleware(h.handleSearchSubscription)))
	searchSubscription.Methods(http.MethodGet, http.MethodOptions)

	addSubscription := r.HandleFunc("/add-subscriptions", corsMiddleware(authMiddleware(h.handleAddSubscriptions)))
	addSubscription.Methods(http.MethodPost, http.MethodOptions)

	deleteSubscriptions := r.HandleFunc("/delete-subscriptions", corsMiddleware(authMiddleware(h.handleDeleteSubscriptions)))
	deleteSubscriptions.Methods(http.MethodDelete, http.MethodOptions)

	fetchFeed := r.HandleFunc("/fetch-feed", corsMiddleware(authMiddleware(h.handleFetchFeed)))
	fetchFeed.Methods(http.MethodGet, http.MethodOptions)

	return r
}

func main() {
	// Load env variables
	_ = godotenv.Load()

	// Get db connection
	conn, err := getDBPool()
	if err != nil {
		fmt.Fprintf(os.Stderr, "Unable to connect to database: %v\n", err)
		return
	}
	defer conn.Close()

	// Init handler
	handler := &Handler{
		conn: conn,
	}

	mux := SetupRouter(handler)
	server := &http.Server{
		Addr:    ":8080",
		Handler: mux,
	}

	fmt.Printf("Starting server at port %d\n", port)
	if err := server.ListenAndServe(); err != nil {
		log.Fatalf("Server error: %v", err)
	}
}
