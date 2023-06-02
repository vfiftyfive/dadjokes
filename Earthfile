#Earthfile version
VERSION 0.7

FROM golang:latest

WORKDIR /app

server:
    # Copy go.mod and go.sum files
    COPY go.mod go.sum ./
    # Download dependencies
    RUN go mod download
    # Copy the source code
    COPY cmd/joke-server/main.go cmd/joke-server/
    COPY internal/ internal/
    # Build the binary
    RUN go build -o joke-server ./cmd/joke-server/main.go
    SAVE ARTIFACT joke-server

docker-server:
    # Copy the binary
    COPY +server/joke-server /app/
    # Expose the port
    EXPOSE 8080
    # Run the binary
    CMD ["./joke-server"]
    SAVE IMAGE joke-server:latest

worker:
    # Copy go.mod and go.sum files
    COPY go.mod go.sum ./
    # Download dependencies
    RUN go mod download
    # Copy the source code
    COPY cmd/joke-worker/main.go cmd/joke-worker/
    COPY internal/ internal/
    # Build the binary
    RUN go build -o joke-worker ./cmd/joke-worker/main.go
    SAVE ARTIFACT joke-worker

docker-worker:
    # Copy the binary
    COPY +worker/joke-worker /app/
    # Run the binary
    CMD ["./joke-worker"]
    SAVE IMAGE joke-worker:latest

docker-compose:
    FROM earthly/dind:alpine
    WORKDIR /test
    # Copy the docker-compose file
    COPY deploy/docker/docker-compose.yaml ./
    # Copy the env file
    COPY deploy/docker/.env ./
    WITH DOCKER \
        --compose docker-compose.yaml \
        --load test-worker:latest=+docker-worker \
        --load test-server:latest=+docker-server 
    RUN docker run --network host curlimages/curl:latest -s http://localhost:8080/joke | grep "Joke"
    END

# Build everything
all:
    BUILD +server
    BUILD +worker
    BUILD +docker-server
    BUILD +docker-worker
    BUILD +docker-compose