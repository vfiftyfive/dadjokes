# Set the base image
FROM golang:1.20

# Set the working directory
WORKDIR /app

# Copy go.mod and go.sum files
COPY go.mod go.sum ./

# Download dependencies
RUN go mod download

# Copy the source code
COPY cmd/joke-worker/main.go cmd/joke-worker/
COPY internal/ internal/

# Build the binary
RUN go build -o joke-worker ./cmd/joke-worker/main.go

# Run the binary
CMD ["./joke-worker"]