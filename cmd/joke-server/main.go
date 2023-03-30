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
	"github.com/vfiftyfive/dadjokes/internal/joke"
)

type Joke struct {
	ID   string
	Text string
}

func main() {
	// Get the API key from the environment
	apiKey := os.Getenv("OPENAI_API_KEY")
	if apiKey == "" {
		log.Fatalf("OPENAI_API_KEY environment variable is not set")
	}

	// Connect to NATS
	nc, err := nats.Connect(constants.natsURL)
	if err != nil {
		log.Fatalf("Failed to connect to NATS: %v", err)
	}
	defer nc.Close()

	// Generate 100 jokes
	openaiClient := openai.NewClient(apiKey)
	joke, err := joke.GenerateJoke(openaiClient)
	for i := 0; i < 100; i++ {
		joke := Joke{Text: joke}

		// Publish the joke to the "joke.save" subject
		nc.Publish(constants.saveJokeSubject, []byte(joke.Text))
	}

	http.HandleFunc("/joke", func(w http.ResponseWriter, r *http.Request) {
		// Get a random joke from the joke-worker
		resp, err := nc.Request(constants.getJokeSubject, nil, 5*time.Second)
		if err != nil {
			http.Error(w, "Error getting joke", http.StatusInternalServerError)
			return
		}

		var joke Joke
		json.Unmarshal(resp.Data, &joke)

		fmt.Fprintf(w, "Joke: %s", joke.Text)
	})

	//Start the HTTP server
	http.ListenAndServe(":8080", nil)
}
