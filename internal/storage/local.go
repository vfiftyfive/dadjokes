package storage

import (
	"encoding/json"
	"fmt"
	"io/ioutil"
	"os"
	"path/filepath"
	"sync"
	"time"
)

// LocalRating represents a rating stored locally
type LocalRating struct {
	JokeID    string    `json:"joke_id"`
	Score     int       `json:"score"`
	Timestamp time.Time `json:"timestamp"`
}

// LocalStorage manages file-based storage for ratings
type LocalStorage struct {
	mu       sync.RWMutex
	basePath string
}

// NewLocalStorage creates a new local storage instance
func NewLocalStorage(basePath string) (*LocalStorage, error) {
	// Create base directory if it doesn't exist
	if err := os.MkdirAll(basePath, 0755); err != nil {
		return nil, fmt.Errorf("failed to create storage directory: %v", err)
	}

	return &LocalStorage{
		basePath: basePath,
	}, nil
}

// SaveRating saves a rating to local storage
func (ls *LocalStorage) SaveRating(jokeID string, score int) error {
	ls.mu.Lock()
	defer ls.mu.Unlock()

	rating := LocalRating{
		JokeID:    jokeID,
		Score:     score,
		Timestamp: time.Now(),
	}

	// Create joke-specific directory
	jokeDir := filepath.Join(ls.basePath, jokeID)
	if err := os.MkdirAll(jokeDir, 0755); err != nil {
		return fmt.Errorf("failed to create joke directory: %v", err)
	}

	// Generate unique filename based on timestamp
	filename := fmt.Sprintf("%d.json", time.Now().UnixNano())
	filepath := filepath.Join(jokeDir, filename)

	// Marshal rating to JSON
	data, err := json.Marshal(rating)
	if err != nil {
		return fmt.Errorf("failed to marshal rating: %v", err)
	}

	// Write to file
	if err := ioutil.WriteFile(filepath, data, 0644); err != nil {
		return fmt.Errorf("failed to write rating file: %v", err)
	}

	return nil
}

// GetRatings retrieves all ratings for a joke
func (ls *LocalStorage) GetRatings(jokeID string) ([]LocalRating, error) {
	ls.mu.RLock()
	defer ls.mu.RUnlock()

	jokeDir := filepath.Join(ls.basePath, jokeID)

	// Check if directory exists
	if _, err := os.Stat(jokeDir); os.IsNotExist(err) {
		return []LocalRating{}, nil
	}

	// Read all rating files
	files, err := ioutil.ReadDir(jokeDir)
	if err != nil {
		return nil, fmt.Errorf("failed to read ratings directory: %v", err)
	}

	var ratings []LocalRating
	for _, file := range files {
		if filepath.Ext(file.Name()) != ".json" {
			continue
		}

		data, err := ioutil.ReadFile(filepath.Join(jokeDir, file.Name()))
		if err != nil {
			continue
		}

		var rating LocalRating
		if err := json.Unmarshal(data, &rating); err != nil {
			continue
		}

		ratings = append(ratings, rating)
	}

	return ratings, nil
}

// GetAllRatings retrieves all ratings from local storage
func (ls *LocalStorage) GetAllRatings() (map[string][]LocalRating, error) {
	ls.mu.RLock()
	defer ls.mu.RUnlock()

	allRatings := make(map[string][]LocalRating)

	// Read all joke directories
	entries, err := ioutil.ReadDir(ls.basePath)
	if err != nil {
		return nil, fmt.Errorf("failed to read storage directory: %v", err)
	}

	for _, entry := range entries {
		if !entry.IsDir() {
			continue
		}

		jokeID := entry.Name()
		ratings, err := ls.GetRatings(jokeID)
		if err != nil {
			continue
		}

		if len(ratings) > 0 {
			allRatings[jokeID] = ratings
		}
	}

	return allRatings, nil
}

// GetStats calculates statistics for a joke's ratings
func (ls *LocalStorage) GetStats(jokeID string) (avgScore float64, count int, err error) {
	ratings, err := ls.GetRatings(jokeID)
	if err != nil {
		return 0, 0, err
	}

	if len(ratings) == 0 {
		return 0, 0, nil
	}

	var total int
	for _, rating := range ratings {
		total += rating.Score
	}

	return float64(total) / float64(len(ratings)), len(ratings), nil
}

// GetStorageInfo returns information about the storage
func (ls *LocalStorage) GetStorageInfo() (map[string]interface{}, error) {
	ls.mu.RLock()
	defer ls.mu.RUnlock()

	info := make(map[string]interface{})

	// Count total ratings
	allRatings, err := ls.GetAllRatings()
	if err != nil {
		return nil, err
	}

	totalRatings := 0
	for _, ratings := range allRatings {
		totalRatings += len(ratings)
	}

	info["total_ratings"] = totalRatings
	info["total_jokes_rated"] = len(allRatings)
	info["storage_path"] = ls.basePath

	// Get disk usage
	var size int64
	err = filepath.Walk(ls.basePath, func(_ string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}
		if !info.IsDir() {
			size += info.Size()
		}
		return nil
	})

	if err == nil {
		info["storage_size_bytes"] = size
	}

	return info, nil
}
