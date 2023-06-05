
# Set the current directory as the base directory for all the relative paths used in the Makefile.
BASE_DIR := $(shell pwd)

# Define the paths for the binaries, source code, and configuration files.
BIN_DIR := $(BASE_DIR)/bin
CMD_DIR := $(BASE_DIR)/cmd
DOCKER_COMPOSE_FILE := $(BASE_DIR)/deploy/docker/docker-compose.yaml

# Define the names for the binaries.
SERVER_BINARY := joke-server
WORKER_BINARY := joke-worker

# Define the registry variable and image names
REGISTRY := vfiftyfive
SERVER_IMAGE := $(REGISTRY)/$(SERVER_BINARY)
WORKER_IMAGE := $(REGISTRY)/$(WORKER_BINARY)

# Define the Docker Compose commands.
DOCKER_COMPOSE_UP := docker-compose -f $(DOCKER_COMPOSE_FILE) up -d
DOCKER_COMPOSE_DOWN := docker-compose -f $(DOCKER_COMPOSE_FILE) down
DOCKER_COMPOSE_BUILD_SERVER := docker build -t test-server -f cmd/joke-server/Dockerfile .
DOCKER_COMPOSE_BUILD_WORKER := docker build -t test-worker -f cmd/joke-worker/Dockerfile .


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

.PHONY: docker-push
docker-push:
	@echo "Pushing Docker images to Docker Hub..."
	@docker push $(SERVER_IMAGE)
	@docker push $(WORKER_IMAGE)
	
.PHONY: deploy
deploy: docker-build docker-up

.PHONY: clean
clean: docker-down
	@echo "Removing Docker images..."
	@docker rmi test-server:latest test-worker:latest || true
	@echo "Removing binaries..."
	@rm -rf $(BIN_DIR)
