package main

import (
	"fmt"
	"log"
	"net/http"
)

const (
	uploadPath = "./uploads"
)

func main() {
	// Simple handler for testing
	http.HandleFunc("/ping", func(w http.ResponseWriter, r *http.Request) {
		fmt.Println("hit")
		fmt.Fprintf(w, "pong")
	})

	port := "8080"
	fmt.Printf("Starting server on port %s\n", port)
	// Start the HTTP server
	if err := http.ListenAndServe(":"+port, nil); err != nil {
		log.Fatalf("Could not start server: %s\n", err)
	}
}
