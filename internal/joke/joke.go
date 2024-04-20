package joke

import (
	"context"
	"errors"
	"fmt"
	"log"
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
type Joke string

// Generates a joke using OpenAI's GPT-4 API
func GenerateJoke(client *openai.Client) (Joke, error) {
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
			resp, err := client.CreateChatCompletion(ctx, openai.ChatCompletionRequest{
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
			clearJoke := reWhitespace.ReplaceAllString(resp.Choices[0].Message.Content, " ")
			return Joke(clearJoke), nil
		}
	}
}

// Saves a joke to the database
func SaveJoke(ctx context.Context, svc *dynamodb.Client, joke Joke) error {
	input := &dynamodb.PutItemInput{
		TableName: aws.String("JokesTable"),
		Item: map[string]types.AttributeValue{
			"Joke": &types.AttributeValueMemberS{Value: string(joke)},
		},
	}

	_, err := svc.PutItem(ctx, input)
	if err != nil {
		return fmt.Errorf("error saving joke to DynamoDB: %v", err)
	}
	return nil
}

// GetRandomJoke retrieves a random joke from the Redis cache
func GetRandomJoke(ctx context.Context, rdb *redis.Client) (Joke, error) {
	var joke Joke

	// Retrieve a random joke ID directly from Redis
	randomJokeID, err := rdb.SRandMember(ctx, "jokes").Result()
	if err != nil {
		log.Printf("Error retrieving random joke ID from cache: %v", err)
		return joke, err
	}

	if randomJokeID == "" {
		return joke, errors.New("no jokes available in cache")
	}

	joke = Joke(randomJokeID)
	return joke, nil
}

// CacheJoke adds the new joke to the cache
func CacheJoke(ctx context.Context, rdb *redis.Client, joke Joke) error {
	_, err := rdb.SAdd(ctx, "jokes", string(joke)).Result()
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

	return similarity >= 0.8
}

// Returns the max of two integers
func max(a, b int) int {
	if a > b {
		return a
	}
	return b
}
