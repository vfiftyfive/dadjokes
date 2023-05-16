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

	RedisPassword      string
	RedisUsername      string
	NatsUsername       string
	NatsPassword       string
	MongoUsername      string
	MongoPassword      string
	MongoAuthMechanism string
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

	RedisPassword = os.Getenv("REDIS_PASSWORD")
	RedisUsername = os.Getenv("REDIS_USERNAME")

	NatsUsername = os.Getenv("NATS_USERNAME")
	NatsPassword = os.Getenv("NATS_PASSWORD")

	MongoUsername = os.Getenv("MONGO_USERNAME")
	MongoPassword = os.Getenv("MONGO_PASSWORD")
	MongoAuthMechanism = os.Getenv("MONGO_AUTH_MECHANISM")

}
