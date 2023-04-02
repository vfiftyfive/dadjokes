
# Set the current directory as the base directory for all the relative paths used in the Makefile.
BASE_DIR := $(shell pwd)

# Define the paths for the binaries, source code, and configuration files.
BIN_DIR := $(BASE_DIR)/bin
CMD_DIR := $(BASE_DIR)/cmd
DOCKER_COMPOSE_FILE := $(BASE_DIR)/docker/docker-compose.yml

# Define the names for the binaries.
SERVER_BINARY := joke-server
WORKER_BINARY := joke-worker

# Define the port mappings for the Docker Compose and Kubernetes files.
export NATS_PORT := 4222
export MONGO_PORT := 27017
export REDIS_PORT := 6379
export SERVER_PORT := 8080

# Define the Docker Compose commands.
DOCKER_COMPOSE_UP := docker-compose -f docker/docker-compose.yaml up -d
DOCKER_COMPOSE_DOWN := docker-compose -f docker/docker-compose.yaml down
DOCKER_COMPOSE_BUILD_SERVER := cd cmd/joke-server && docker build -t joke-server -f Dockerfile ../..
DOCKER_COMPOSE_BUILD_WORKER := cd cmd/joke-worker && docker build -t joke-worker -f Dockerfile ../..

# Define the Go build commands.
GO_BUILD_SERVER := go build -o $(BIN_DIR)/$(SERVER_BINARY) $(CMD_DIR)/joke-server/main.go
GO_BUILD_WORKER := go build -o $(BIN_DIR)/$(WORKER_BINARY) $(CMD_DIR)/joke-worker/main.go

# Define the default target.
.PHONY: all
all: build docker-build

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
	@$(DOCKER_COMPOSE_BUILD_SERVER)
	@$(DOCKER_COMPOSE_BUILD_WORKER)

.PHONY: docker-up
docker-up:
	@echo "Starting Docker containers..."
	@$(DOCKER_COMPOSE_UP)
.PHONY: docker-down
docker-down:
	@echo "Stopping Docker containers..."
	@$(DOCKER_COMPOSE_DOWN)


