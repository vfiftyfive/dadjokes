package main

import (
	"context"
	"log"
	"os"
	"time"

	"github.com/nats-io/nats.go"
	"github.com/redis/go-redis/v9"
	"github.com/sashabaranov/go-openai"
	"github.com/vfiftyfive/dadjokes/internal/aws"
	"github.com/vfiftyfive/dadjokes/internal/constants"
	"github.com/vfiftyfive/dadjokes/internal/joke"
)

func main() {
	apiKey := os.Getenv("OPENAI_API_KEY")
	if apiKey == "" {
		log.Fatalf("OPENAI_API_KEY environment variable is not set")
	}

	// Create a new OpenAI client
	openaiClient := openai.NewClient(apiKey)

	// Connect to NATS
	nc, err := nats.Connect(constants.NatsURL)
	if err != nil {
		log.Fatalf("Failed to connect to NATS: %v", err)
	}
	defer nc.Close()

	// Connect to Redis
	rdb := redis.NewClient(&redis.Options{
		Addr: constants.RedisURL,
	})
	defer rdb.Close()

	// Test the connection to Redis
	_, err = rdb.Ping(context.Background()).Result()
	if err != nil {
		log.Fatalf("Failed to connect to Redis: %v with connection address set to %v", err, constants.RedisURL)
	}

	// Initialize the DynamoDB
	dynamoClient := aws.DynamoDBClient()

	// Subscribe to the "jokes.get" subject
	nc.Subscribe(constants.GetJokeSubject, func(msg *nats.Msg) {
		ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
		defer cancel()

		// Retrieve the count of jokes in Redis
		jokeCount, err := rdb.SCard(ctx, "jokes").Result()
		if err != nil {
			log.Printf("Error retrieving joke count: %v", err)
			msg.Respond([]byte("Error retrieving joke count"))
			return
		}

		var responseJoke joke.Joke
		for {
			if jokeCount >= 20 {
				// Get a random joke from the cache
				responseJoke, err = joke.GetRandomJoke(ctx, rdb)
				if err == nil {
					break
				}
				continue
			}
			// Generate a new joke using OpenAI
			responseJoke, err = joke.GenerateJoke(openaiClient)
			if err != nil {
				log.Printf("Error generating joke: %v", err)
				msg.Respond([]byte("Error generating joke"))
				return
			}

			// Check if the generated joke is a duplicate
			isUnique := true
			cachedJokes, err := rdb.SMembers(ctx, "jokes").Result()
			if err != nil {
				log.Printf("Error retrieving jokes from Redis for similarity check: %v", err)
				msg.Respond([]byte("Error retrieving joke for similarity check"))
				return
			}
			for _, cachedJoke := range cachedJokes {
				if joke.IsSimilarJoke(cachedJoke, string(responseJoke)) {
					isUnique = false
					break
				}
			}

			if !isUnique {
				continue // Generate a new joke if found similar
			}
			break
		}

		// Respond with the selected or generated joke
		log.Printf("responseJoke: %v", responseJoke)
		msg.Respond([]byte(responseJoke))
	})

	// Subscribe to joke.save subject
	nc.Subscribe(constants.SaveJokeSubject, func(msg *nats.Msg) {
		ctx := context.Background()

		newJoke := joke.Joke(msg.Data)

		log.Printf("Joke to save: %v", newJoke)
		// Save and cache the new joke
		err = joke.SaveJoke(ctx, dynamoClient, newJoke)
		if err != nil {
			log.Printf("Error saving new joke: %v", err)
			return
		}
		err = joke.CacheJoke(ctx, rdb, newJoke)
		if err != nil {
			log.Printf("Error caching joke: %v", err)
		}
	})

	// Wait for messages
	select {}
}
