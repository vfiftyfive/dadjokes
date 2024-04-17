package main

import (
	"context"
	"encoding/json"
	"log"
	"os"

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
		ctx := context.Background()

		// Retrieve the count of jokes in Redis
		jokeCount, err := rdb.SCard(ctx, "jokeIDs").Result()
		if err != nil {
			log.Printf("Error retrieving joke count: %v", err)
			msg.Respond([]byte("Error retrieving jokes"))
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
			generatedJokeTxt, err := joke.GenerateJoke(openaiClient)
			if err != nil {
				log.Printf("Error generating joke: %v", err)
				return
			}

			// Check if the generated joke is a duplicate
			isUnique := true
			jokeIDs, _ := rdb.SMembers(ctx, "jokeIDs").Result()
			for _, id := range jokeIDs {
				existingJokeData, _ := rdb.Get(ctx, "joke:"+id).Bytes()
				var existingJoke joke.Joke
				json.Unmarshal(existingJokeData, &existingJoke)
				if joke.IsSimilarJoke(existingJoke.Text, generatedJokeTxt) {
					isUnique = false
					break
				}
			}

			if !isUnique {
				continue // Generate a new joke if found similar
			}

			responseJoke = joke.Joke{Text: generatedJokeTxt}
			// Save and cache the new, unique joke
			err = joke.SaveJoke(ctx, dynamoClient, &responseJoke)
			if err != nil {
				log.Printf("Error saving new joke: %v", err)
				msg.Respond([]byte("Error saving new joke"))
				return
			}
			joke.CacheJoke(ctx, rdb, &responseJoke)
			break
		}

		// Respond with the selected or generated joke
		jokeData, _ := json.Marshal(responseJoke)
		msg.Respond(jokeData)
	})
	// Subscribe to joke.save subject
	nc.Subscribe(constants.SaveJokeSubject, func(msg *nats.Msg) {
		ctx := context.Background()
		newJoke := joke.Joke{Text: string(msg.Data)}

		// Save and cache the new joke
		err := joke.SaveJoke(ctx, dynamoClient, &newJoke)
		if err != nil {
			log.Printf("Error saving new joke: %v", err)
			return
		}
		err = joke.CacheJoke(ctx, rdb, &newJoke)
		if err != nil {
			log.Printf("Error caching joke: %v", err)
		}
	})

	// Wait for messages
	select {}
}
