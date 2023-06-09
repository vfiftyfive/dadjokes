FROM golang:1.20 AS builder
WORKDIR /app
COPY go.mod go.sum ./
RUN go mod download
COPY cmd/joke-worker/main.go cmd/joke-worker/
COPY internal/ internal/
RUN go build -o joke-worker ./cmd/joke-worker/main.go

FROM gcr.io/distroless/base-debian10
COPY --from=builder /app/joke-worker /joke-worker
ENTRYPOINT ["/joke-worker"]