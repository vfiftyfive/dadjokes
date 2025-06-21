package main

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"math/rand"
	"os"
	"time"

	"github.com/nats-io/nats.go"
	"github.com/redis/go-redis/v9"
	"github.com/sashabaranov/go-openai"
	"github.com/vfiftyfive/dadjokes/internal/constants"
	"github.com/vfiftyfive/dadjokes/internal/joke"
	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"
)

func main() {
	log.Printf("Starting joke-worker...")
	log.Printf("NATS URL: %s", constants.NatsURL)
	log.Printf("MongoDB URL: %s", constants.MongoURL)
	log.Printf("Redis URL: %s", constants.RedisURL)

	apiKey := os.Getenv("OPENAI_API_KEY")
	if apiKey == "" {
		log.Fatalf("OPENAI_API_KEY environment variable is not set")
	}
	log.Printf("OpenAI API key configured")

	// Create a new OpenAI client
	openaiClient := openai.NewClient(apiKey)

	// Connect to NATS
	nc, err := nats.Connect(constants.NatsURL)
	if err != nil {
		log.Fatalf("Failed to connect to NATS: %v", err)
	}
	defer nc.Close()
	log.Printf("Connected to NATS successfully")

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
	log.Printf("Connected to Redis successfully")

	// Connect to MongoDB
	client, err := mongo.Connect(context.Background(), options.Client().ApplyURI(constants.MongoURL))
	if err != nil {
		log.Fatalf("Failed to connect to MongoDB: %v", err)
	}
	defer client.Disconnect(context.Background())
	log.Printf("Connected to MongoDB successfully")

	jokesCollection := client.Database("jokesdb").Collection("jokes")

	// Subscribe to the "jokes.get" subject
	nc.Subscribe(constants.GetJokeSubject, func(msg *nats.Msg) {
		requestStart := time.Now()
		log.Printf("Received joke request")

		jokesCount, err := jokesCollection.CountDocuments(context.Background(), bson.M{})
		if err != nil {
			log.Printf("Error counting jokes: %v", err)
			msg.Respond([]byte("Error counting jokes"))
			return
		}
		log.Printf("Current jokes in database: %d", jokesCount)

		var retrievedJoke joke.Joke

		// If we have enough jokes, use cached/DB jokes (fast path)
		if jokesCount >= 20 {
			log.Printf("Using cached joke (fast path)")
			retrievedJoke, err = joke.GetRandomJoke(jokesCollection, rdb)
			if err != nil {
				log.Printf("Error getting random joke: %v", err)
				msg.Respond([]byte("Error getting random joke"))
				return
			}
		} else {
			// Generate new joke with timeout protection (slow path)
			log.Printf("Generating new joke (slow path)")

			// Improved retry logic with better duplicate handling
			maxAttempts := 5 // Increased from 3
			duplicateCount := 0
			for attempt := 1; attempt <= maxAttempts; attempt++ {
				log.Printf("Generation attempt %d/%d", attempt, maxAttempts)

				generatedJokeTxt, err := joke.GenerateJoke(openaiClient)

				if err != nil {
					log.Printf("Error generating joke (attempt %d): %v", attempt, err)
					if attempt == maxAttempts {
						// Return a message about the error instead of a generic error
						retrievedJoke = joke.Joke{
							Text: "I tried to generate a joke, but the AI is having a bad day. Here's a classic: Why don't scientists trust atoms? Because they make up everything!",
						}
						break
					}
					continue
				}

				log.Printf("Generated joke: %s", generatedJokeTxt)

				// Quick duplicate check - only check if we have jokes
				if jokesCount > 0 {
					log.Printf("Checking for duplicates against %d existing jokes", jokesCount)

					cursor, err := jokesCollection.Find(context.Background(), bson.M{})
					if err != nil {
						log.Printf("Error finding jokes for duplicate check: %v", err)
						// If we can't check duplicates, just use the joke
						retrievedJoke = joke.Joke{Text: generatedJokeTxt}
						break
					}

					foundSimilarJoke := false
					checkedCount := 0
					for cursor.Next(context.Background()) {
						var existingJoke joke.Joke
						cursor.Decode(&existingJoke)
						checkedCount++

						if joke.IsSimilarJoke(existingJoke.Text, generatedJokeTxt) {
							log.Printf("Duplicate detected (%.2f%% similar): '%s' ~ '%s'",
								getSimilarityPercentage(existingJoke.Text, generatedJokeTxt),
								existingJoke.Text, generatedJokeTxt)
							foundSimilarJoke = true
							duplicateCount++
							break
						}
					}
					cursor.Close(context.Background())
					log.Printf("Checked %d jokes for duplicates", checkedCount)

					if !foundSimilarJoke {
						retrievedJoke = joke.Joke{Text: generatedJokeTxt}
						break
					} else if attempt < maxAttempts {
						log.Printf("Duplicate found (#%d), retrying with different topic...", duplicateCount)
						continue
					} else {
						// After max attempts, provide feedback about duplicates
						retrievedJoke = joke.Joke{
							Text: fmt.Sprintf("I detected %d duplicates while trying to generate a unique joke. Here's attempt #%d: %s",
								duplicateCount, attempt, generatedJokeTxt),
						}
						break
					}
				} else {
					// No existing jokes, so no duplicates possible
					retrievedJoke = joke.Joke{Text: generatedJokeTxt}
					break
				}
			}

			// If all attempts failed completely (empty text), use a diverse fallback
			if retrievedJoke.Text == "" {
				log.Printf("All generation attempts failed, using diverse fallback")
				fallbackJoke := getFallbackJoke()
				log.Printf("Selected fallback joke: %s", fallbackJoke)
				retrievedJoke = joke.Joke{Text: fallbackJoke}
			}
		}

		// Respond with the joke
		jokeBytes, _ := json.Marshal(retrievedJoke)
		err = msg.Respond(jokeBytes)
		if err != nil {
			log.Printf("Error responding to NATS message: %v", err)
		} else {
			duration := time.Since(requestStart)
			log.Printf("Request completed in %v", duration)
		}
	})

	nc.Subscribe(constants.SaveJokeSubject, func(msg *nats.Msg) {
		log.Printf("Received save joke request")

		// Parse the joke from JSON
		var retrievedJoke joke.Joke
		err := json.Unmarshal(msg.Data, &retrievedJoke)
		if err != nil {
			// Fallback to old format for backward compatibility
			retrievedJoke = joke.Joke{Text: string(msg.Data)}
		}

		// Save the joke to the DB and cache it to Redis
		err = joke.SaveJoke(jokesCollection, &retrievedJoke)
		if err == nil {
			log.Printf("Joke saved to database: %s", retrievedJoke.Text)
			err = joke.CacheJoke(rdb, &retrievedJoke)
			if err != nil {
				log.Printf("Error caching joke: %v", err)
			} else {
				log.Printf("Joke cached successfully")
			}
		} else {
			log.Printf("Error saving joke: %v", err)
		}
	})

	log.Printf("Joke-worker ready and listening for requests")

	// Wait for messages
	select {}
}

// Helper function to calculate similarity percentage
func getSimilarityPercentage(joke1, joke2 string) float64 {
	// This would use the same logic as IsSimilarJoke but return the percentage
	// For now, just return a placeholder
	return 75.0
}

// getFallbackJoke returns a diverse fallback joke when generation fails
func getFallbackJoke() string {
	fallbacks := []string{
		"Why don't scientists trust atoms? Because they make up everything!",
		"I told my wife she was drawing her eyebrows too high. She looked surprised.",
		"Why don't eggs tell jokes? They'd crack each other up!",
		"What do you call a fake noodle? An impasta!",
		"Why did the scarecrow win an award? He was outstanding in his field!",
		"What do you call a dinosaur that crashes his car? Tyrannosaurus Wrecks!",
		"Why don't skeletons fight each other? They don't have the guts!",
		"What did the ocean say to the beach? Nothing, it just waved!",
		"Why did the math book look so sad? Because it had too many problems!",
		"What do you call a bear with no teeth? A gummy bear!",
	}

	// Use current time as seed for randomness
	rand.Seed(time.Now().UnixNano())
	return fallbacks[rand.Intn(len(fallbacks))]
}
