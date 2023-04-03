package constants

import (
	"log"
	"os"
	"time"
)

const (
	RedisTTL        = 10 * time.Minute
	GetJokeSubject  = "joke.get"
	SaveJokeSubject = "joke.save"
)

var (
	ApiKey   string
	NatsURL  string
	MongoURL string
	RedisURL string
)

func init() {
	//ApiKey = os.Getenv("OPENAI_API_KEY")
	ApiKey = "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"
	if ApiKey == "" {
		log.Fatalf("OPENAI_API_KEY environment variable is not set")
	}

	NatsURL = os.Getenv("NATS_URL")
	if NatsURL == "" {
		NatsURL = "nats://localhost:4222"
	}

	MongoURL = os.Getenv("MONGO_URL")
	if MongoURL == "" {
		MongoURL = "mongodb://localhost:27017"
	}

	RedisEnvURL := os.Getenv("REDIS_URL")
	if RedisEnvURL == "" {
		RedisURL = "redis:6379"
	} else {
		RedisURL = RedisEnvURL
	}

}
