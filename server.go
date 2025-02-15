package main

import (
	"fmt"
	"log"
	"net/http"
)

const (
	port = 8080
)

func main() {
	fmt.Printf("Starting server at port %d\n", port)

	http.HandleFunc("/", handleRoot)

	http.HandleFunc("/add", handleAddSubscription)

	if err := http.ListenAndServe(":8080", nil); err != nil {
		log.Fatal(err)
	}
}
