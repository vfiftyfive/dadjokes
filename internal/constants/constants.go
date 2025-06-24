package constants

import (
	"fmt"
	"os"
	"strings"
	"time"
)

const (
	RedisTTL       = 10 * time.Minute
	GetJokeSubject = "joke.get"
)

var (
	NatsURL  string
	MongoURL string
	RedisURL string
)

// buildMongoURL constructs MongoDB connection string from individual components
func buildMongoURL() string {
	// Check if MONGO_URL is explicitly provided (for backward compatibility)
	if url := os.Getenv("MONGO_URL"); url != "" {
		return url
	}

	// Build connection string from components
	username := os.Getenv("MONGO_USERNAME")
	password := os.Getenv("MONGO_PASSWORD")
	hosts := os.Getenv("MONGO_HOSTS")
	database := os.Getenv("MONGO_DATABASE")
	replicaSet := os.Getenv("MONGO_REPLICA_SET")
	readPreference := os.Getenv("MONGO_READ_PREFERENCE")
	maxStaleness := os.Getenv("MONGO_MAX_STALENESS_SECONDS")
	ssl := os.Getenv("MONGO_SSL")
	authMechanism := os.Getenv("MONGO_AUTH_MECHANISM")

	// Set defaults
	if username == "" {
		username = "demo"
	}
	if password == "" {
		password = "spectrocloud"
	}
	if hosts == "" {
		hosts = "localhost:27017"
	}
	if database == "" {
		database = "admin"
	}
	if replicaSet == "" {
		replicaSet = "mongodb"
	}
	if readPreference == "" {
		readPreference = "primaryPreferred"
	}
	if maxStaleness == "" {
		maxStaleness = "90"
	}
	if ssl == "" {
		ssl = "false"
	}

	// Build connection string
	var params []string
	params = append(params, fmt.Sprintf("replicaSet=%s", replicaSet))
	params = append(params, fmt.Sprintf("ssl=%s", ssl))
	params = append(params, fmt.Sprintf("readPreference=%s", readPreference))

	if readPreference == "primaryPreferred" || readPreference == "secondaryPreferred" {
		params = append(params, fmt.Sprintf("maxStalenessSeconds=%s", maxStaleness))
	}

	if authMechanism != "" {
		params = append(params, fmt.Sprintf("authMechanism=%s", authMechanism))
	}

	// Always add authSource=admin since users are created in admin database
	params = append(params, "authSource=admin")

	return fmt.Sprintf("mongodb://%s:%s@%s/%s?%s",
		username, password, hosts, database, strings.Join(params, "&"))
}

func init() {
	NatsURL = os.Getenv("NATS_URL")
	if NatsURL == "" {
		NatsURL = "nats://localhost:4222"
	}

	MongoURL = buildMongoURL()

	RedisEnvURL := os.Getenv("REDIS_URL")
	if RedisEnvURL == "" {
		RedisURL = "redis:6379"
	} else {
		RedisURL = RedisEnvURL
	}

}
