package constants

import "time"

const (
	NatsURL         = "nats://localhost:4222"
	RedisAddr       = "localhost:6379"
	MongoURI        = "mongodb://localhost:27017"
	GetJokeSubject  = "joke.get"
	SaveJokeSubject = "joke.save"
	RedisTTL        = 10 * time.Minute
)
