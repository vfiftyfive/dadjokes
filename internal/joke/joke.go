package joke

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"math/rand"

	"github.com/go-redis/redis/v8"
	openai "github.com/sashabaranov/go-openai"
	"github.com/texttheater/golang-levenshtein/levenshtein"
	"github.com/vfiftyfive/dadjokes/internal/constants"
	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/bson/primitive"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"
)

// Joke represents a joke
type Joke struct {
	ID   string
	Text string
}

// Generates a joke using OpenAI's GPT-3 API
func GenerateJoke(client *openai.Client) (string, error) {
	resp, err := client.CreateChatCompletion(
		context.Background(),
		openai.ChatCompletionRequest{
			Model: openai.GPT3Dot5Turbo,
			Messages: []openai.ChatCompletionMessage{
				{
					Role:    openai.ChatMessageRoleUser,
					Content: "Tell me a dad joke",
				},
			},
		},
	)
	if err != nil {
		return "", err
	}

	if len(resp.Choices) == 0 {
		return "", err
	}
	joke := resp.Choices[0].Message.Content

	return joke, nil

}

// Saves a joke to the database
func SaveJoke(jokesCollection *mongo.Collection, joke *Joke) error {
	res, err := jokesCollection.InsertOne(context.Background(), joke)
	if err != nil {
		log.Printf("Error saving joke: %v", err)
		return err
	}
	insertedID := res.InsertedID.(primitive.ObjectID)
	joke.ID = insertedID.Hex()
	return nil
}

// Returns a random joke from the database
func GetRandomJoke(jokesCollection *mongo.Collection, rdb *redis.Client) (Joke, error) {
	jokeFromDB := getJokeFromDB(jokesCollection)

	cacheKey := fmt.Sprintf("joke:%s", jokeFromDB.ID)
	jokeBytes, err := rdb.Get(context.Background(), cacheKey).Bytes()
	if err == nil {
		jokeFromCache := Joke{}
		json.Unmarshal(jokeBytes, &jokeFromCache)
		return jokeFromCache, nil
	}
	return jokeFromDB, nil
}

// Returns a random joke from the database
func getJokeFromDB(jokesCollection *mongo.Collection) Joke {
	joke := Joke{}

	count, err := jokesCollection.CountDocuments(context.Background(), bson.M{})
	if err != nil {
		log.Printf("Error retrieving joke: %v", err)
	}

	opts := options.FindOne().SetSkip(int64(rand.Intn(int(count))))
	err = jokesCollection.FindOne(context.Background(), bson.M{}, opts).Decode(&joke)
	if err != nil {
		log.Printf("Error retrieving joke: %v", err)
	}

	return joke
}

// Saves a joke to the cache and the DB
func CacheJoke(rdb *redis.Client, joke *Joke) {
	// Add the new joke to the cache
	jokeBytes, _ := json.Marshal(joke)
	rdb.Set(context.Background(), fmt.Sprintf("joke:%s", joke.ID), jokeBytes, constants.RedisTTL)
}

// Checks if a joke is similar to an existing joke
func IsSimilarJoke(joke1, joke2 string) bool {
	distance := levenshtein.DistanceForStrings([]rune(joke1), []rune(joke2), levenshtein.DefaultOptions)
	maxLength := max(len(joke1), len(joke2))
	similarity := 1 - float64(distance)/float64(maxLength)

	return similarity >= 0.8
}

// Returns the max of two integers
func max(a, b int) int {
	if a > b {
		return a
	}
	return b
}
