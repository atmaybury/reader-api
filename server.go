package main

import (
	"context"
	"fmt"
	"github.com/jackc/pgx/v5"
	"github.com/joho/godotenv"
	"log"
	"net/http"
	"os"
)

// TODO
//  link finder - exit if </head> reached
//  Pass user token to http handlers

const (
	port = 8080
)

type Handler struct {
	conn *pgx.Conn
}

func main() {
	// Load env vars
	err := godotenv.Load() // 👈 load .env file
	if err != nil {
		log.Fatal(err)
	}

	// Get db connection
	conn, err := getDBConnection()
	if err != nil {
		fmt.Fprintf(os.Stderr, "Unable to connect to database: %v\n", err)
		return
	}
	defer conn.Close(context.Background())

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
