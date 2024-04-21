package main

import (
	"fmt"
	"log"
	"net/http"
	"time"

	"github.com/nats-io/nats.go"
	"github.com/vfiftyfive/dadjokes/internal/constants"
	"github.com/vfiftyfive/dadjokes/internal/joke"
)

func main() {
	// Connect to NATS
	nc, err := nats.Connect(constants.NatsURL)
	if err != nil {
		log.Fatalf("Failed to connect to NATS: %v", err)
	}
	defer nc.Close()

	http.HandleFunc("/joke", func(w http.ResponseWriter, r *http.Request) {
		// Request a joke from the joke-worker
		resp, err := nc.Request(constants.GetJokeSubject, nil, 20*time.Second)
		if err != nil {
			log.Printf("Error getting joke: %v", err) // Log error
			http.Error(w, "Error getting joke", http.StatusInternalServerError)
			return
		}
		fmt.Fprintf(w, "Joke: %s", joke.Joke(resp.Data))
		nc.Publish(constants.SaveJokeSubject, resp.Data)
	})

	// Start the HTTP server
	http.ListenAndServe(":8080", nil)
}
