package main

import (
	"context"
	"encoding/json"
	"log"

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

	//Create a new OpenAI client
	openaiClient := openai.NewClient(constants.ApiKey)

	//Connect to NATS
	nc, err := nats.Connect(constants.NatsURL)
	if err != nil {
		log.Fatalf("Failed to connect to NATS: %v", err)
	}
	defer nc.Close()

	//Connect to Redis
	rdb := redis.NewClient(&redis.Options{
		Addr: constants.RedisURL,
	})
	defer rdb.Close()

	//Connect to MongoDB
	client, err := mongo.Connect(context.Background(), options.Client().ApplyURI(constants.MongoURL))
	if err != nil {
		log.Fatalf("Failed to connect to MongoDB: %v", err)
	}
	defer client.Disconnect(context.Background())

	jokesCollection := client.Database("jokesdb").Collection("jokes")

	//Subscribe to the "jokes.get" subject
	nc.Subscribe(constants.GetJokeSubject, func(msg *nats.Msg) {

		jokesCount, err := jokesCollection.CountDocuments(context.Background(), bson.M{})
		if err != nil {
			log.Printf("Error counting jokes: %v", err)
			msg.Respond([]byte("Error counting jokes"))
			return
		}

		var retrievedJoke joke.Joke
		for {
			//If the DB collection reaches 50 jokes, pick a random joke from the cache or the DB
			if jokesCount >= 50 {
				retrievedJoke, err = joke.GetRandomJoke(jokesCollection, rdb)
				if err == nil {
					break
				}
				continue
			}
			//Generate a new joke and make sure it's not a duplicate
			generatedJokeTxt, err := joke.GenerateJoke(openaiClient)
			if err != nil {
				log.Printf("Error generating joke: %v", err)
				continue
			}

			// Check if the joke is a duplicate
			cursor, err := jokesCollection.Find(context.Background(), bson.M{})
			if err != nil {
				log.Printf("Error finding joke: %v", err)
				continue
			}
			defer cursor.Close(context.Background())

			foundSimilarJoke := false
			for cursor.Next(context.Background()) {
				var existingJoke joke.Joke
				cursor.Decode(&existingJoke)

				if joke.IsSimilarJoke(existingJoke.Text, generatedJokeTxt) {
					foundSimilarJoke = true
					break
				}
			}

			if !foundSimilarJoke {
				retrievedJoke = joke.Joke{Text: generatedJokeTxt}
				break
			}
		}

		//Respond with the joke
		jokeBytes, _ := json.Marshal(retrievedJoke)
		err = msg.Respond(jokeBytes)
		if err != nil {
			log.Printf("Error responding to NATS message: %v", err)
		}

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
