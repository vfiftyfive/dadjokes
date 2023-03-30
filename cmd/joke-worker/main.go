package main

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"math/rand"
	"time"

	"github.com/go-redis/redis/v8"
	"github.com/nats-io/nats.go"
	"github.com/vfiftyfive/dadjokes/internal/constants"
	"github.com/vfiftyfive/dadjokes/internal/joke"
	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"
)

func main() {
	// Connect to NATS
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

	// Subscribe to the "jokes.get" subject
	nc.Subscribe(constants.GetJokeSubject, func(msg *nats.Msg) {
		// Get a random joke from the database
		joke := getRandomJoke(jokesCollection)

		// Cache the jokes in Redis for 5 minutes
		cacheKey := fmt.Sprintf("joke:%s", joke.ID)
		jokeBytes, _ := json.Marshal(joke)
		rdb.Set(context.Background(), cacheKey, jokeBytes, 5*time.Minute)
		//Respond with the joke
		msg.Respond((jokeBytes))
	})

	// Subscribe to the "jokes.save" subject
	nc.Subscribe(constants.SaveJokeSubject, func(msg *nats.Msg) {
		// Decode the joke from the message
		joke := joke.Joke{}
		json.Unmarshal(msg.Data, &joke)
		if err != nil {
			log.Printf("Error decoding joke: %v", err)
			return
		}

		// Save the joke to the database
		_, err := saveJoke(jokesCollection, &joke)
		if err != nil {
			log.Printf("Error saving joke: %v", err)
			return
		}
	})

	// Wait for messages
	select {}

}

// Returns a random joke from the database
func getRandomJoke(jokesCollection *mongo.Collection) *joke.Joke {
	joke := &joke.Joke{}
	opts := options.FindOne().SetSkip(int64(rand.Intn(100)))
	err := jokesCollection.FindOne(context.Background(), bson.M{}, opts).Decode(joke)
	if err != nil {
		log.Printf("Error retrieving joke: %v", err)
		return nil
	}
	return joke
}

// Saves a joke to the database
func saveJoke(jokesCollection *mongo.Collection, joke *joke.Joke) (string, error) {
	res, err := jokesCollection.InsertOne(context.Background(), joke)
	if err != nil {
		return "", err
	}
	return res.InsertedID.(string), nil
}
