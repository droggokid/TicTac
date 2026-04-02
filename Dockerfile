# Build stage
FROM golang:1.25.1-alpine AS builder

WORKDIR /build

# Copy go mod files
COPY go.mod go.sum ./
RUN go mod download

# Copy source code
COPY cmd/server/ ./cmd/server/
COPY internal/server/ ./internal/server/
COPY internal/transport/ ./internal/transport/
COPY internal/game/ ./internal/game/

# Build the server binary
RUN CGO_ENABLED=0 GOOS=linux go build -a -installsuffix cgo -o server ./cmd/server

# Runtime stage
FROM alpine:latest

# Install ca-certificates for HTTPS
RUN apk --no-cache add ca-certificates

# Create non-root user
RUN addgroup -g 1000 tictac && \
    adduser -D -u 1000 -G tictac tictac

WORKDIR /app

# Copy binary from builder
COPY --from=builder /build/server .

# Change ownership
RUN chown -R tictac:tictac /app

# Switch to non-root user
USER tictac

# Expose WebSocket port
EXPOSE 8080

# Run the server (listen on all interfaces for k8s)
CMD ["./server", "-addr", "0.0.0.0:8080"]
