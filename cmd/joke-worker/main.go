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
	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"
)

const (
	natsURL          = "nats://localhost:4222"
	redisAddr        = "localhost:6379"
	mongoURI         = "mongodb://localhost:27017"
	getJokeSubject   = "joke.get"
	saveJokeSubject  = "joke.save"
	mongoSaveSubject = "mongo.save"
)

// Joke represents a joke
type Joke struct {
	ID   string
	Text string
}

func main() {
	// Connect to NATS
	nc, err := nats.Connect(natsURL)
	if err != nil {
		log.Fatalf("Failed to connect to NATS: %v", err)
	}
	defer nc.Close()

	//Connect to Redis
	rdb := redis.NewClient(&redis.Options{
		Addr: redisAddr,
	})
	defer rdb.Close()

	//Connect to MongoDB
	client, err := mongo.Connect(context.Background(), options.Client().ApplyURI(mongoURI))
	if err != nil {
		log.Fatalf("Failed to connect to MongoDB: %v", err)
	}
	defer client.Disconnect(context.Background())

	jokesCollection := client.Database("jokesdb").Collection("jokes")

	// Subscribe to the "jokes.get" subject
	nc.Subscribe(getJokeSubject, func(msg *nats.Msg) {
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
	nc.Subscribe(saveJokeSubject, func(msg *nats.Msg) {
		// Decode the joke from the message
		joke := Joke{}
		json.Unmarshal(msg.Data, &joke)
		if err != nil {
			log.Printf("Error decoding joke: %v", err)
			return
		}

		// Save the joke to the database
		jokeID, err := saveJoke(jokesCollection, &joke)
		if err != nil {
			log.Printf("Error saving joke: %v", err)
			return
		}

		//Notify the joke-server that the joke has been saved
		nc.Publish(mongoSaveSubject, []byte(jokeID))
	})

	// Wait for messages
	select {}

}

// Returns a random joke from the database
func getRandomJoke(jokesCollection *mongo.Collection) *Joke {
	joke := &Joke{}
	opts := options.FindOne().SetSkip(int64(rand.Intn(100)))
	err := jokesCollection.FindOne(context.Background(), bson.M{}, opts).Decode(joke)
	if err != nil {
		log.Printf("Error retrieving joke: %v", err)
		return nil
	}
	return joke
}

// Saves a joke to the database
func saveJoke(jokesCollection *mongo.Collection, joke *Joke) (string, error) {
	res, err := jokesCollection.InsertOne(context.Background(), joke)
	if err != nil {
		return "", err
	}
	return res.InsertedID.(string), nil
}
