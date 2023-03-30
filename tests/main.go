package main

import (
	"context"
	"fmt"
	"log"
	"os"

	openai "github.com/sashabaranov/go-openai"
)

func main() {
	// Get the OpenAI API key from the environment variable.
	apiKey := os.Getenv("OPENAI_API_KEY")
	if apiKey == "" {
		log.Fatalf("OPENAI_API_KEY environment variable is not set")
	}

	// Create a new OpenAI API client.
	client := openai.NewClient(apiKey)

	// Send a chat completion request to the OpenAI API.
	resp, err := client.CreateChatCompletion(
		context.Background(),
		openai.ChatCompletionRequest{
			Model: openai.GPT3Dot5Turbo,
			Messages: []openai.ChatCompletionMessage{
				{
					Role:    openai.ChatMessageRoleUser,
					Content: "Tell me 10 fun facts about Japan",
				},
			},
		},
	)
	if err != nil {
		log.Fatalf("Chat completion failed: %v", err)
	}

	// Get the joke from the chat completion response.
	if len(resp.Choices) == 0 {
		log.Fatalf("No joke found in response")
	}
	joke := resp.Choices[0].Message.Content

	// Print the joke to the console.
	fmt.Println(joke)
}
