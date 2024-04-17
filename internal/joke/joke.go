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

	"github.com/aws/aws-sdk-go-v2/service/dynamodb"
	"github.com/aws/aws-sdk-go-v2/service/dynamodb/types"
	"github.com/aws/aws-sdk-go/aws"
	"github.com/redis/go-redis/v9"
	openai "github.com/sashabaranov/go-openai"
	"github.com/texttheater/golang-levenshtein/levenshtein"
)

// Joke represents a joke
type Joke struct {
	ID   string
	Text string
}

// Generates a joke using OpenAI's GPT-4 API
func GenerateJoke(client *openai.Client) (string, error) {
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	// Retry mechanism with timeout
	for {
		select {
		case <-ctx.Done():
			return "", errors.New("GPT-4 API call timed out")
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
func SaveJoke(ctx context.Context, svc *dynamodb.Client, joke *Joke) error {
	input := &dynamodb.PutItemInput{
		TableName: aws.String("JokesTable"),
		Item: map[string]types.AttributeValue{
			"ID":   &types.AttributeValueMemberS{Value: joke.ID},
			"Text": &types.AttributeValueMemberS{Value: joke.Text},
		},
	}

	_, err := svc.PutItem(ctx, input)
	if err != nil {
		log.Printf("Error saving joke to DynamoDB: %v", err)
		return fmt.Errorf("error saving joke to DynamoDB: %v", err)
	}
	return nil
}

// GetRandomJoke retrieves a random joke from the Redis cache
func GetRandomJoke(ctx context.Context, rdb *redis.Client) (Joke, error) {
	var joke Joke

	// Retrieve all joke IDs from Redis (from the 'jokeIDs' set)
	jokeIDs, err := rdb.SMembers(ctx, "jokeIDs").Result()
	if err != nil {
		log.Printf("Error retrieving joke IDs from cache: %v", err)
		return joke, err
	}

	if len(jokeIDs) == 0 {
		return joke, errors.New("no jokes available in cache")
	}

	// Select a random joke ID
	rand.Seed(time.Now().UnixNano())
	randomIndex := rand.Intn(len(jokeIDs))
	randomJokeID := jokeIDs[randomIndex]

	// Fetch the joke data from Redis using the selected joke ID
	jokeData, err := rdb.Get(ctx, "joke:"+randomJokeID).Bytes()
	if err != nil {
		log.Printf("Error retrieving joke from cache: %v", err)
		return joke, err
	}

	err = json.Unmarshal(jokeData, &joke)
	if err != nil {
		log.Printf("Error unmarshalling joke: %v", err)
		return joke, err
	}

	return joke, nil
}

// CacheJoke adds the new joke to the cache and tracks its ID for random retrieval
func CacheJoke(ctx context.Context, rdb *redis.Client, joke *Joke) error {
	jokeBytes, err := json.Marshal(joke)
	if err != nil {
		log.Printf("Failed to marshal joke: %v", err)
		return fmt.Errorf("failed to marshal joke: %v", err)
	}

	_, err = rdb.Set(ctx, fmt.Sprintf("joke:%s", joke.ID), jokeBytes, -1).Result() // Using -1 for no expiration
	if err != nil {
		log.Printf("Failed to set joke in cache: %v", err)
		return fmt.Errorf("failed to set joke in cache: %v", err)
	}

	// Add the joke ID to a set for random access
	_, err = rdb.SAdd(ctx, "jokeIDs", joke.ID).Result()
	if err != nil {
		log.Printf("Failed to add joke ID to set: %v", err)
		return fmt.Errorf("failed to add joke ID to set: %v", err)
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
