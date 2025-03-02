package main

import (
	"context"
	"fmt"
	"log"
	"net/http"
	"os"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
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

func SetupRouter(handler *Handler) *http.ServeMux {
	mux := http.NewServeMux()

	mux.HandleFunc("/", corsMiddleware(handler.handleRoot))
	mux.HandleFunc("/register", corsMiddleware(handler.handleRegisterUser))
	mux.HandleFunc("/login", corsMiddleware(handler.handleLogin))

	// Auth
	mux.HandleFunc("/user-subscriptions", corsMiddleware(authMiddleware(handler.handleGetUserSubscriptions)))
	mux.HandleFunc("/add-subscription", corsMiddleware(authMiddleware(handler.handleAddSubscription)))

	return mux
}

func main() {
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
		log.Printf("Server error: %v", err)
	}
}
