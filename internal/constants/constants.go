package constants

import (
	"os"
	"time"
)

const (
	RedisTTL        = 10 * time.Minute
	GetJokeSubject  = "joke.get"
	SaveJokeSubject = "joke.save"
)

var (
	NatsURL  string
	MongoURL string
	RedisURL string
)

func init() {
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
