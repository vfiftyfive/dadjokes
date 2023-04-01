package main

import (
	"context"
	"encoding/json"
	"log"
	"os"

	"github.com/go-redis/redis/v8"
	"github.com/nats-io/nats.go"
	"github.com/sashabaranov/go-openai"
	"github.com/vfiftyfive/dadjokes/internal/constants"
	"github.com/vfiftyfive/dadjokes/internal/joke"
	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"
)

func main() {

	//Get the API key from the environment
	apiKey := os.Getenv("OPENAI_API_KEY")
	if apiKey == "" {
		log.Fatalf("OPENAI_API_KEY environment variable is not set")
	}

	//Create a new OpenAI client
	openaiClient := openai.NewClient(apiKey)

	//Connect to NATS
	nc, err := nats.Connect(constants.NatsURL)
	if err != nil {
		log.Fatalf("Failed to connect to NATS: %v", err)
	}
	defer nc.Close()

	//Connect to Redis
	rdb := redis.NewClient(&redis.Options{
		Addr: constants.RedisAddr,
	})
	defer rdb.Close()

	//Connect to MongoDB
	client, err := mongo.Connect(context.Background(), options.Client().ApplyURI(constants.MongoURI))
	if err != nil {
		log.Fatalf("Failed to connect to MongoDB: %v", err)
	}
	defer client.Disconnect(context.Background())

	jokesCollection := client.Database("jokesdb").Collection("jokes")

	//Subscribe to the "jokes.get" subject
	nc.Subscribe(constants.GetJokeSubject, func(msg *nats.Msg) {

		jokesCount, _ := jokesCollection.CountDocuments(context.Background(), bson.M{})
		var retrievedJoke joke.Joke
		for {
			//If the DB collection reaches 50 jokes, pick a random joke from the cache or the DB
			if jokesCount >= 50 {
				retrievedJoke, err = joke.GetRandomJoke(jokesCollection, rdb)
				if err == nil {
					continue
				}
				break
			}
			//Generate a new joke and make sure it's not a duplicate
			generatedJokeTxt, err := joke.GenerateJoke(openaiClient)
			if err != nil {
				log.Printf("Error generating joke: %v", err)
				continue
			}

			filter := bson.M{"text": generatedJokeTxt}
			jokeExists := jokesCollection.FindOne(context.Background(), filter).Err()
			if jokeExists == mongo.ErrNoDocuments {
				retrievedJoke = joke.Joke{Text: generatedJokeTxt}
				break
			}
		}

		//Respond with the joke
		jokeBytes, _ := json.Marshal(retrievedJoke)
		msg.Respond(jokeBytes)
	})

	nc.Subscribe(constants.SaveJokeSubject, func(msg *nats.Msg) {
		// Save the joke to the DB
		retrievedJoke := joke.Joke{Text: string(msg.Data)}
		err := joke.SaveJoke(jokesCollection, &retrievedJoke)
		if err == nil {
			joke.CacheJoke(rdb, &retrievedJoke)
		}
	})

	// Wait for messages
	select {}
}
