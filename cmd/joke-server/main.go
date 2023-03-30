package main

import (
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"os"
	"time"

	"github.com/nats-io/nats.go"
	openai "github.com/sashabaranov/go-openai"
	"github.com/vfiftyfive/dadjokes/internal/constants"
	"github.com/vfiftyfive/dadjokes/internal/joke"
)

func main() {
	// Get the API key from the environment
	apiKey := os.Getenv("OPENAI_API_KEY")
	if apiKey == "" {
		log.Fatalf("OPENAI_API_KEY environment variable is not set")
	}

	// Connect to NATS
	nc, err := nats.Connect(constants.NatsURL)
	if err != nil {
		log.Fatalf("Failed to connect to NATS: %v", err)
	}
	defer nc.Close()

	// Generate 100 jokes
	openaiClient := openai.NewClient(apiKey)

	for i := 0; i < 100; i++ {
		jokeTxt, err := joke.GenerateJoke(openaiClient)
		if err != nil {
			log.Printf("Failed to generate joke: %v", err)
			continue
		}
		joke := joke.Joke{Text: jokeTxt}

		// Publish the joke to the "joke.save" subject
		nc.Publish(constants.SaveJokeSubject, []byte(joke.Text))
	}

	http.HandleFunc("/joke", func(w http.ResponseWriter, r *http.Request) {
		// Get a random joke from the joke-worker
		resp, err := nc.Request(constants.GetJokeSubject, nil, 5*time.Second)
		if err != nil {
			http.Error(w, "Error getting joke", http.StatusInternalServerError)
			return
		}

		var joke joke.Joke
		json.Unmarshal(resp.Data, &joke)

		fmt.Fprintf(w, "Joke: %s", joke.Text)
	})

	//Start the HTTP server
	http.ListenAndServe(":8080", nil)
}
