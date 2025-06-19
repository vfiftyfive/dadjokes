package health

import (
	"context"
	"time"

	"github.com/nats-io/nats.go"
	"github.com/redis/go-redis/v9"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/readpref"
)

// CheckResult represents the result of a health check
type CheckResult struct {
	Service string `json:"service"`
	Status  string `json:"status"`
	Error   string `json:"error,omitempty"`
}

// CheckMongoDB checks MongoDB connectivity
func CheckMongoDB(client *mongo.Client) CheckResult {
	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	defer cancel()

	err := client.Ping(ctx, readpref.Primary())
	if err != nil {
		return CheckResult{
			Service: "mongodb",
			Status:  "unhealthy",
			Error:   err.Error(),
		}
	}

	return CheckResult{
		Service: "mongodb",
		Status:  "healthy",
	}
}

// CheckRedis checks Redis connectivity
func CheckRedis(rdb *redis.Client) CheckResult {
	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	defer cancel()

	_, err := rdb.Ping(ctx).Result()
	if err != nil {
		return CheckResult{
			Service: "redis",
			Status:  "unhealthy",
			Error:   err.Error(),
		}
	}

	return CheckResult{
		Service: "redis",
		Status:  "healthy",
	}
}

// CheckNATS checks NATS connectivity
func CheckNATS(nc *nats.Conn) CheckResult {
	if nc == nil || nc.Status() != nats.CONNECTED {
		return CheckResult{
			Service: "nats",
			Status:  "unhealthy",
			Error:   "not connected",
		}
	}

	return CheckResult{
		Service: "nats",
		Status:  "healthy",
	}
}

// IsHealthy returns true if all services are healthy
func IsHealthy(checks []CheckResult) bool {
	for _, check := range checks {
		if check.Status != "healthy" {
			return false
		}
	}
	return true
}
