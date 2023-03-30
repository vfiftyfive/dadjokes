
# Set the current directory as the base directory for all the relative paths used in the Makefile.
BASE_DIR := $(shell pwd)

# Define the paths for the binaries, source code, and configuration files.
BIN_DIR := $(BASE_DIR)/bin
CMD_DIR := $(BASE_DIR)/cmd

# Define the names for the binaries.
SERVER_BINARY := joke-server
WORKER_BINARY := joke-worker

# Define the port mappings for the Docker Compose and Kubernetes files.
NATS_PORT := 4222
MONGO_PORT := 27017
REDIS_PORT := 6379
SERVER_PORT := 8080

# Define the Docker Compose commands.
DOCKER_COMPOSE_UP := docker-compose up -d
DOCKER_COMPOSE_DOWN := docker-compose down


# Define the Go build commands.
GO_BUILD_SERVER := go build -o $(BIN_DIR)/$(SERVER_BINARY) $(CMD_DIR)/joke-server/main.go
GO_BUILD_WORKER := go build -o $(BIN_DIR)/$(WORKER_BINARY) $(CMD_DIR)/joke-worker/main.go

# Define the default target.
.PHONY: all
all: build

# Define the build targets.
.PHONY: build
build: build-server build-worker

.PHONY: build-server
build-server:
	@echo "Building $(SERVER_BINARY) binary..."
	@mkdir -p $(BIN_DIR)
	@$(GO_BUILD_SERVER)

.PHONY: build-worker
build-worker:
	@echo "Building $(WORKER_BINARY) binary..."
	@mkdir -p $(BIN_DIR)
	@$(GO_BUILD_WORKER)

# Define the Docker targets.
.PHONY: docker-build
docker-build:
	@echo "Building Docker images..."
	@docker-compose build

.PHONY: docker-up
docker-up:
	@echo "Starting Docker containers..."
	@docker-compose up -d

.PHONY: docker-down
docker-down:
	@echo "Stopping Docker containers..."
	@docker-compose down

