# --- Stage 1: Build the Go binary ---
FROM golang:1.25-alpine AS builder

RUN apk add --no-cache git ca-certificates

WORKDIR /app

COPY go.mod go.sum ./
RUN go mod download

COPY . .

RUN CGO_ENABLED=0 GOOS=linux go build -o scanner-service ./cmd/server/main.go

# --- Stage 2: Final minimal image ---
FROM alpine:latest

RUN apk add --no-cache ca-certificates

WORKDIR /root/

# Copy the binary from the builder stage
COPY --from=builder /app/scanner-service .

# Copy web assets
COPY --from=builder /app/web ./web

# FIX: Point to the correct path where inventory.json now lives
COPY --from=builder /app/retail-edge-lab/inventory.json .

# Create certs directory (standard for SRE setups)
RUN mkdir -p /root/certs

EXPOSE 443
EXPOSE 8080

# Start the service
CMD ["./scanner-service"]