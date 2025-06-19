package joke

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"log"
	"math/rand"
	"regexp"
	"strings"
	"time"

	"github.com/redis/go-redis/v9"
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

// Rating represents a joke rating
type Rating struct {
	JokeID  string    `bson:"joke_id" json:"joke_id"`
	Score   int       `bson:"score" json:"score"`
	RatedAt time.Time `bson:"rated_at" json:"rated_at"`
}

// JokeStats represents aggregated rating statistics
type JokeStats struct {
	JokeID       string  `bson:"_id" json:"joke_id"`
	AverageScore float64 `bson:"avg_score" json:"avg_score"`
	Count        int     `bson:"count" json:"count"`
}

// Generates a joke using OpenAI's GPT-3 API
func GenerateJoke(client *openai.Client) (string, error) {
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	// Retry mechanism with timeout
	for {
		select {
		case <-ctx.Done():
			return "", errors.New("GPT-3 API call timed out")
		default:
			message := []openai.ChatCompletionMessage{
				{
					Role:    openai.ChatMessageRoleUser,
					Content: "Tell me a dad joke",
				},
			}
			resp, err := client.CreateChatCompletion(context.Background(), openai.ChatCompletionRequest{
				Model:    openai.GPT4TurboPreview,
				Messages: message,
			})

			if err != nil {
				log.Printf("Error generating joke: %v", err)
				return "", err
			}

			if len(resp.Choices) == 0 {
				continue
			}
			reWhitespace := regexp.MustCompile(`[\s\n\t]+`)
			joke := reWhitespace.ReplaceAllString(resp.Choices[0].Message.Content, " ")
			return joke, nil
		}
	}
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
	} else if err != redis.Nil {
		log.Printf("Error retrieving joke from cache: %v", err)

		return Joke{}, err
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
func CacheJoke(rdb *redis.Client, joke *Joke) error {
	// Add the new joke to the cache
	jokeBytes, err := json.Marshal(joke)
	if err != nil {
		return fmt.Errorf("failed to marshal joke: %v", err)
	}
	_, err = rdb.Set(context.Background(), fmt.Sprintf("joke:%s", joke.ID), jokeBytes, constants.RedisTTL).Result()
	if err != nil {
		return fmt.Errorf("failed to set joke in cache: %v", err)
	}
	return nil
}

// Checks if a joke is similar to an existing joke
func IsSimilarJoke(joke1, joke2 string) bool {
	reWhitespace := regexp.MustCompile(`[\s\n\t]+`)
	rePunctuation := regexp.MustCompile(`[^\w\s]`)
	cleanJoke1 := strings.TrimSpace(reWhitespace.ReplaceAllString(strings.ToLower(joke1), " "))
	cleanJoke2 := strings.TrimSpace(reWhitespace.ReplaceAllString(strings.ToLower(joke2), " "))
	cleanJoke1 = rePunctuation.ReplaceAllString(cleanJoke1, "")
	cleanJoke2 = rePunctuation.ReplaceAllString(cleanJoke2, "")
	distance := levenshtein.DistanceForStrings([]rune(strings.ToLower(cleanJoke1)), []rune(strings.ToLower(cleanJoke2)), levenshtein.DefaultOptions)
	maxLength := max(len(joke1), len(joke2))
	similarity := 1 - float64(distance)/float64(maxLength)

	return similarity >= 0.5
}

// Returns the max of two integers
func max(a, b int) int {
	if a > b {
		return a
	}
	return b
}

// SaveRating saves a rating to the database
func SaveRating(ratingsCollection *mongo.Collection, rating *Rating) error {
	rating.RatedAt = time.Now()
	_, err := ratingsCollection.InsertOne(context.Background(), rating)
	if err != nil {
		log.Printf("Error saving rating: %v", err)
		return err
	}
	return nil
}

// GetJokeRating retrieves the average rating and count for a joke
func GetJokeRating(ratingsCollection *mongo.Collection, jokeID string) (JokeStats, error) {
	pipeline := mongo.Pipeline{
		{{"$match", bson.D{{"joke_id", jokeID}}}},
		{{"$group", bson.D{
			{"_id", "$joke_id"},
			{"avg_score", bson.D{{"$avg", "$score"}}},
			{"count", bson.D{{"$sum", 1}}},
		}}},
	}

	cursor, err := ratingsCollection.Aggregate(context.Background(), pipeline)
	if err != nil {
		return JokeStats{}, err
	}
	defer cursor.Close(context.Background())

	var stats JokeStats
	if cursor.Next(context.Background()) {
		err = cursor.Decode(&stats)
		if err != nil {
			return JokeStats{}, err
		}
		return stats, nil
	}

	// No ratings found
	return JokeStats{JokeID: jokeID, AverageScore: 0, Count: 0}, nil
}

// GetTopRatedJokes retrieves the top N rated jokes
func GetTopRatedJokes(ratingsCollection *mongo.Collection, jokesCollection *mongo.Collection, limit int) ([]map[string]interface{}, error) {
	pipeline := mongo.Pipeline{
		{{"$group", bson.D{
			{"_id", "$joke_id"},
			{"avg_score", bson.D{{"$avg", "$score"}}},
			{"count", bson.D{{"$sum", 1}}},
		}}},
		{{"$sort", bson.D{{"avg_score", -1}}}},
		{{"$limit", limit}},
	}

	cursor, err := ratingsCollection.Aggregate(context.Background(), pipeline)
	if err != nil {
		return nil, err
	}
	defer cursor.Close(context.Background())

	var results []map[string]interface{}
	for cursor.Next(context.Background()) {
		var stats JokeStats
		err = cursor.Decode(&stats)
		if err != nil {
			continue
		}

		// Fetch the joke text
		var joke Joke
		err = jokesCollection.FindOne(context.Background(), bson.M{"_id": stats.JokeID}).Decode(&joke)
		if err == nil {
			result := map[string]interface{}{
				"joke_id":       stats.JokeID,
				"text":          joke.Text,
				"average_score": stats.AverageScore,
				"rating_count":  stats.Count,
			}
			results = append(results, result)
		}
	}

	return results, nil
}

// GetJokeByID retrieves a joke by its ID
func GetJokeByID(jokesCollection *mongo.Collection, jokeID string) (Joke, error) {
	var joke Joke
	objID, err := primitive.ObjectIDFromHex(jokeID)
	if err != nil {
		return joke, err
	}

	err = jokesCollection.FindOne(context.Background(), bson.M{"_id": objID}).Decode(&joke)
	if err != nil {
		return joke, err
	}
	joke.ID = jokeID
	return joke, nil
}
