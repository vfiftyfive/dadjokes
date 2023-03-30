package internal

import (
	"context"

	openai "github.com/sashabaranov/go-openai"
)

type Joke struct {
	ID   string
	Text string
}

func GenerateJoke(client *openai.Client) (string, error) {
	resp, err := client.CreateChatCompletion(
		context.Background(),
		openai.ChatCompletionRequest{
			Model: openai.GPT3Dot5Turbo,
			Messages: []openai.ChatCompletionMessage{
				{
					Role:    openai.ChatMessageRoleUser,
					Content: "Tell me dad joke",
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
