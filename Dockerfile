# --- Stage 1: Build the Go binary ---
FROM golang:1.25-alpine AS builder

# Install git and certificates for secure connections
RUN apk add --no-cache git ca-certificates

WORKDIR /app

# Copy dependency files first (for better caching)
COPY go.mod go.sum ./
RUN go mod download

# Copy the rest of the source code
COPY . .

# Build the binary with optimizations for size and speed
RUN CGO_ENABLED=0 GOOS=linux go build -o scanner-service ./cmd/server/main.go

# --- Stage 2: Final minimal image ---
# Using 'scratch' or 'alpine' is an SRE standard to reduce attack surface
FROM alpine:latest

RUN apk add --no-cache ca-certificates

WORKDIR /root/

# Copy the binary from the builder stage
COPY --from=builder /app/scanner-service .
# Copy your web assets and inventory for the service
COPY --from=builder /app/web ./web
COPY --from=builder /app/inventory.json .

# Create the directory for certificates (Tailscale/Edge setup)
RUN mkdir -p /etc/ssl/certs /etc/ssl/private

# Expose the HTTPS port
EXPOSE 443

# Run the service
CMD ["./scanner-service"]