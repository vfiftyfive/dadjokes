package main

import (
	"encoding/json"
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
		resp, err := nc.Request(constants.GetJokeSubject, nil, 5*time.Second)
		if err != nil {
			http.Error(w, "Error getting joke", http.StatusInternalServerError)
			return
		}

		joke := joke.Joke{}
		json.Unmarshal(resp.Data, &joke)
		fmt.Fprintf(w, "Joke: %s", joke.Text)

		// Publish the joke text to the "joke.save" subject
		nc.Publish(constants.SaveJokeSubject, []byte(joke.Text))

	})

	//Start the HTTP server
	http.ListenAndServe(":8080", nil)
}
