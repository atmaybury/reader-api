package main

import (
	"context"
	"fmt"
	"log"
	"net/http"
	"os"

	"github.com/jackc/pgx/v5"
)

// TODO
//  auth middleware
//  auth calls to add subscription
//  link finder - exit if </head> reached

const (
	port = 8080
)

type Handler struct {
	conn *pgx.Conn
}

func main() {
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

	http.HandleFunc("/", handler.handleRoot)
	http.HandleFunc("/register", handler.handleRegisterUser)
	http.HandleFunc("/add", authMiddleware(handler.handleAddSubscription))

	if err := http.ListenAndServe(":8080", nil); err != nil {
		log.Fatal(err)
	}
}
