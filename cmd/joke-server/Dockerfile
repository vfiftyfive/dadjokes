FROM golang:1.20 AS builder
WORKDIR /app
COPY go.mod go.sum ./
RUN go mod download
COPY cmd/joke-server/main.go cmd/joke-server/
COPY internal/ internal/
RUN go build -o joke-server ./cmd/joke-server/main.go

FROM gcr.io/distroless/base-debian10
COPY --from=builder /app/joke-server /joke-server
EXPOSE 8080
ENTRYPOINT ["/joke-server"]