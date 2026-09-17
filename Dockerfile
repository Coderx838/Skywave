# Build stage
FROM golang:1.24-alpine AS builder

WORKDIR /app

# Install git and build essentials
RUN apk add --no-cache git

# Copy dependencies
COPY go.mod go.sum ./
RUN go mod download

# Copy source code
COPY . .

# Build static binary for server
RUN CGO_ENABLED=0 GOOS=linux go build -ldflags="-s -w" -o /app/skywave-server ./cmd/skywave-server

# Final lightweight scratch stage
FROM alpine:3.20

RUN apk --no-cache add ca-certificates tzdata

WORKDIR /app

COPY --from=builder /app/skywave-server /app/skywave-server

# Default port
EXPOSE 8080

# Run server with zero-log default or custom params (picks up $PORT automatically if provided)
ENTRYPOINT ["/app/skywave-server"]
CMD ["-host", "0.0.0.0", "-name", "Skywave Community Node"]
