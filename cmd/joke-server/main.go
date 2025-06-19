package main

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"os"
	"strconv"
	"strings"
	"time"

	"github.com/nats-io/nats.go"
	"github.com/redis/go-redis/v9"
	"github.com/vfiftyfive/dadjokes/internal/constants"
	"github.com/vfiftyfive/dadjokes/internal/fault"
	"github.com/vfiftyfive/dadjokes/internal/health"
	"github.com/vfiftyfive/dadjokes/internal/joke"
	"github.com/vfiftyfive/dadjokes/internal/storage"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"
)

var (
	nc              *nats.Conn
	rdb             *redis.Client
	mongoClient     *mongo.Client
	jokesCollection *mongo.Collection
	localStorage    *storage.LocalStorage
	faultInjector   *fault.FaultInjector
	region          string
)

func main() {
	// Get region from environment
	region = os.Getenv("REGION")
	if region == "" {
		region = "unknown"
	}
	log.Printf("Starting joke-server in region: %s", region)

	// Initialize fault injector
	faultInjector = fault.NewFaultInjector()

	// Connect to services
	connectToServices()

	// Setup HTTP routes
	setupRoutes()

	// Start the HTTP server
	log.Println("Starting HTTP server on :8080")
	if err := http.ListenAndServe(":8080", nil); err != nil {
		log.Fatalf("Failed to start HTTP server: %v", err)
	}
}

func connectToServices() {
	var err error

	// Connect to NATS
	nc, err = nats.Connect(constants.NatsURL)
	if err != nil {
		log.Fatalf("Failed to connect to NATS: %v", err)
	}

	// Connect to Redis
	rdb = redis.NewClient(&redis.Options{
		Addr: constants.RedisURL,
	})

	// Connect to MongoDB
	mongoClient, err = mongo.Connect(context.Background(), options.Client().ApplyURI(constants.MongoURL))
	if err != nil {
		log.Fatalf("Failed to connect to MongoDB: %v", err)
	}

	jokesCollection = mongoClient.Database("jokesdb").Collection("jokes")

	// Initialize local storage for ratings
	storagePath := os.Getenv("RATINGS_STORAGE_PATH")
	if storagePath == "" {
		storagePath = "/data/ratings"
	}

	localStorage, err = storage.NewLocalStorage(storagePath)
	if err != nil {
		log.Fatalf("Failed to initialize local storage: %v", err)
	}
	log.Printf("Local ratings storage initialized at: %s", storagePath)
}

func setupRoutes() {
	// Health check endpoints
	http.HandleFunc("/startz", handleStartup)
	http.HandleFunc("/livez", handleLiveness)
	http.HandleFunc("/readyz", handleReadiness)

	// Joke endpoints
	http.HandleFunc("/joke", handleJoke)
	http.HandleFunc("/joke/", handleJokeByID)

	// Rating endpoints
	http.HandleFunc("/rating", handleRating)
	http.HandleFunc("/rating/", handleGetRating)
	http.HandleFunc("/ratings/top", handleTopRatings)
	http.HandleFunc("/ratings/storage", handleStorageInfo)

	// Fault injection endpoints
	http.HandleFunc("/inject/fault", handleInjectFault)
	http.HandleFunc("/inject/restore", handleRestoreFault)
}

// Health check handlers
func handleStartup(w http.ResponseWriter, r *http.Request) {
	w.WriteHeader(http.StatusOK)
	json.NewEncoder(w).Encode(map[string]string{
		"status": "ok",
		"region": region,
	})
}

func handleLiveness(w http.ResponseWriter, r *http.Request) {
	w.WriteHeader(http.StatusOK)
	json.NewEncoder(w).Encode(map[string]string{
		"status": "alive",
		"region": region,
	})
}

func handleReadiness(w http.ResponseWriter, r *http.Request) {
	checks := []health.CheckResult{
		health.CheckMongoDB(mongoClient),
		health.CheckRedis(rdb),
		health.CheckNATS(nc),
	}

	// Add local storage check
	storageInfo, err := localStorage.GetStorageInfo()
	storageCheck := health.CheckResult{
		Service: "local_storage",
		Status:  "healthy",
	}
	if err != nil {
		storageCheck.Status = "unhealthy"
		storageCheck.Error = err.Error()
	}
	checks = append(checks, storageCheck)

	response := map[string]interface{}{
		"region":       region,
		"checks":       checks,
		"ready":        health.IsHealthy(checks),
		"storage_info": storageInfo,
	}

	if health.IsHealthy(checks) {
		w.WriteHeader(http.StatusOK)
	} else {
		w.WriteHeader(http.StatusServiceUnavailable)
	}

	json.NewEncoder(w).Encode(response)
}

// Joke handlers
func handleJoke(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	// Check for fault injection
	if faultInjector.ShouldFail(fault.ComponentNATS) {
		http.Error(w, "NATS service unavailable (fault injected)", http.StatusServiceUnavailable)
		return
	}

	if delay := faultInjector.GetDelay(fault.ComponentNATS); delay > 0 {
		time.Sleep(delay)
	}

	// Request a joke from the joke-worker
	resp, err := nc.Request(constants.GetJokeSubject, nil, 15*time.Second)
	if err != nil {
		log.Printf("Error getting joke: %v", err)
		http.Error(w, "Error getting joke", http.StatusInternalServerError)
		return
	}

	var jokeObj joke.Joke
	err = json.Unmarshal(resp.Data, &jokeObj)
	if err != nil {
		log.Printf("Error unmarshalling joke: %v", err)
		http.Error(w, "Error unmarshalling joke", http.StatusInternalServerError)
		return
	}

	// Return JSON response
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]interface{}{
		"id":     jokeObj.ID,
		"text":   jokeObj.Text,
		"region": region,
	})

	// Publish the joke to save
	nc.Publish(constants.SaveJokeSubject, resp.Data)
}

func handleJokeByID(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	// Extract joke ID from path
	parts := strings.Split(r.URL.Path, "/")
	if len(parts) < 3 {
		http.Error(w, "Invalid joke ID", http.StatusBadRequest)
		return
	}
	jokeID := parts[2]

	// Check for fault injection
	if faultInjector.ShouldFail(fault.ComponentMongo) {
		http.Error(w, "MongoDB service unavailable (fault injected)", http.StatusServiceUnavailable)
		return
	}

	// Try Redis cache first
	if !faultInjector.ShouldFail(fault.ComponentRedis) {
		cacheKey := fmt.Sprintf("joke:%s", jokeID)
		jokeBytes, err := rdb.Get(context.Background(), cacheKey).Bytes()
		if err == nil {
			var cachedJoke joke.Joke
			if json.Unmarshal(jokeBytes, &cachedJoke) == nil {
				w.Header().Set("Content-Type", "application/json")
				w.Header().Set("X-Cache", "HIT")
				json.NewEncoder(w).Encode(map[string]interface{}{
					"id":     cachedJoke.ID,
					"text":   cachedJoke.Text,
					"region": region,
				})
				return
			}
		}
	}

	// Get from MongoDB
	jokeObj, err := joke.GetJokeByID(jokesCollection, jokeID)
	if err != nil {
		if err == mongo.ErrNoDocuments {
			http.Error(w, "Joke not found", http.StatusNotFound)
		} else {
			http.Error(w, "Error retrieving joke", http.StatusInternalServerError)
		}
		return
	}

	w.Header().Set("Content-Type", "application/json")
	w.Header().Set("X-Cache", "MISS")
	json.NewEncoder(w).Encode(map[string]interface{}{
		"id":     jokeObj.ID,
		"text":   jokeObj.Text,
		"region": region,
	})
}

// Rating handlers
func handleRating(w http.ResponseWriter, r *http.Request) {
	switch r.Method {
	case http.MethodPost:
		handlePostRating(w, r)
	default:
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
	}
}

func handlePostRating(w http.ResponseWriter, r *http.Request) {
	var req struct {
		ID    string `json:"id"`
		Score int    `json:"score"`
	}

	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "Invalid request body", http.StatusBadRequest)
		return
	}

	if req.Score < 1 || req.Score > 5 {
		http.Error(w, "Score must be between 1 and 5", http.StatusBadRequest)
		return
	}

	// Save to local storage
	if err := localStorage.SaveRating(req.ID, req.Score); err != nil {
		log.Printf("Error saving rating: %v", err)
		http.Error(w, "Error saving rating", http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]interface{}{
		"status":  "success",
		"region":  region,
		"storage": "local",
	})
}

func handleGetRating(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	// Extract joke ID from path
	parts := strings.Split(r.URL.Path, "/")
	if len(parts) < 3 {
		http.Error(w, "Invalid joke ID", http.StatusBadRequest)
		return
	}
	jokeID := parts[2]

	// Get from local storage
	avgScore, count, err := localStorage.GetStats(jokeID)
	if err != nil {
		log.Printf("Error retrieving rating: %v", err)
		http.Error(w, "Error retrieving rating", http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]interface{}{
		"id":      jokeID,
		"avg":     avgScore,
		"cnt":     count,
		"region":  region,
		"storage": "local",
	})
}

func handleTopRatings(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	// Parse query parameter
	n := 10 // default
	if nStr := r.URL.Query().Get("n"); nStr != "" {
		if parsed, err := strconv.Atoi(nStr); err == nil && parsed > 0 {
			n = parsed
		}
	}

	// Get all ratings from local storage
	allRatings, err := localStorage.GetAllRatings()
	if err != nil {
		log.Printf("Error retrieving ratings: %v", err)
		http.Error(w, "Error retrieving ratings", http.StatusInternalServerError)
		return
	}

	// Calculate averages and sort
	type jokeScore struct {
		JokeID   string
		AvgScore float64
		Count    int
		Text     string
	}

	var scores []jokeScore
	for jokeID, ratings := range allRatings {
		if len(ratings) == 0 {
			continue
		}

		total := 0
		for _, rating := range ratings {
			total += rating.Score
		}

		avg := float64(total) / float64(len(ratings))

		// Get joke text from MongoDB
		jokeObj, err := joke.GetJokeByID(jokesCollection, jokeID)
		jokeText := ""
		if err == nil {
			jokeText = jokeObj.Text
		}

		scores = append(scores, jokeScore{
			JokeID:   jokeID,
			AvgScore: avg,
			Count:    len(ratings),
			Text:     jokeText,
		})
	}

	// Sort by average score
	for i := 0; i < len(scores)-1; i++ {
		for j := i + 1; j < len(scores); j++ {
			if scores[j].AvgScore > scores[i].AvgScore {
				scores[i], scores[j] = scores[j], scores[i]
			}
		}
	}

	// Limit to top N
	if len(scores) > n {
		scores = scores[:n]
	}

	// Format response
	var topJokes []map[string]interface{}
	for _, score := range scores {
		topJokes = append(topJokes, map[string]interface{}{
			"joke_id":       score.JokeID,
			"text":          score.Text,
			"average_score": score.AvgScore,
			"rating_count":  score.Count,
		})
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]interface{}{
		"jokes":   topJokes,
		"region":  region,
		"storage": "local",
	})
}

func handleStorageInfo(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	info, err := localStorage.GetStorageInfo()
	if err != nil {
		http.Error(w, "Error retrieving storage info", http.StatusInternalServerError)
		return
	}

	info["region"] = region

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(info)
}

// Fault injection handlers
func handleInjectFault(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	component := r.URL.Query().Get("component")
	faultType := r.URL.Query().Get("type")

	var comp fault.Component
	switch component {
	case "mongo":
		comp = fault.ComponentMongo
	case "redis":
		comp = fault.ComponentRedis
	case "nats":
		comp = fault.ComponentNATS
	case "openai":
		comp = fault.ComponentOpenAI
	default:
		http.Error(w, "Invalid component", http.StatusBadRequest)
		return
	}

	var ft fault.FaultType
	switch faultType {
	case "delay":
		ft = fault.FaultDelay
	case "error":
		ft = fault.FaultError
	case "partition":
		ft = fault.FaultPartition
	default:
		http.Error(w, "Invalid fault type", http.StatusBadRequest)
		return
	}

	faultInjector.InjectFault(comp, ft, 30*time.Second)

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]interface{}{
		"status":    "fault injected",
		"component": component,
		"type":      faultType,
		"duration":  "30s",
		"region":    region,
	})
}

func handleRestoreFault(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	component := r.URL.Query().Get("component")

	var comp fault.Component
	switch component {
	case "mongo":
		comp = fault.ComponentMongo
	case "redis":
		comp = fault.ComponentRedis
	case "nats":
		comp = fault.ComponentNATS
	case "openai":
		comp = fault.ComponentOpenAI
	case "": // Restore all
		faultInjector.RestoreFault(fault.ComponentMongo)
		faultInjector.RestoreFault(fault.ComponentRedis)
		faultInjector.RestoreFault(fault.ComponentNATS)
		faultInjector.RestoreFault(fault.ComponentOpenAI)
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(map[string]interface{}{
			"status": "all faults restored",
			"region": region,
		})
		return
	default:
		http.Error(w, "Invalid component", http.StatusBadRequest)
		return
	}

	faultInjector.RestoreFault(comp)

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]interface{}{
		"status":    "fault restored",
		"component": component,
		"region":    region,
	})
}
