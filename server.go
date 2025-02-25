package main

import (
	"fmt"
	"log"
	"net/http"
	"os"

	"github.com/jackc/pgx/v5/pgxpool"
)

// TODO
//  link finder - exit if </head> reached
//  Pass user token to http handlers

const (
	port = 8080
)

type Handler struct {
	conn *pgxpool.Pool
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

	fmt.Printf("Starting server at port %d\n", port)

	// No auth
	http.HandleFunc("/", handler.handleRoot)
	http.HandleFunc("/register", handler.handleRegisterUser)
	http.HandleFunc("/login", handler.handleLogin)

	// Auth
	http.HandleFunc("/add", authMiddleware(handler.handleAddSubscription))
	http.HandleFunc("/user-subscriptions", authMiddleware(handler.handleGetUserSubscriptions))

	if err := http.ListenAndServe(":8080", nil); err != nil {
		log.Fatal(err)
	}
}
