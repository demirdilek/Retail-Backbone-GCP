# --- Stage 1: Build the Go binary ---
FROM golang:1.25-alpine AS builder

RUN apk add --no-cache git ca-certificates
WORKDIR /app

COPY go.mod go.sum ./
RUN go mod download

COPY . .

# Build the binary as 'retail-edge'
RUN CGO_ENABLED=0 GOOS=linux go build -o retail-edge ./cmd/server/main.go

# --- Stage 2: Final minimal image ---
FROM alpine:latest
RUN apk add --no-cache ca-certificates
WORKDIR /root/

# Copy the binary with the new name
COPY --from=builder /app/retail-edge .

# Copy assets to the correct working directory
COPY --from=builder /app/web ./web
COPY --from=builder /app/retail-edge-lab/inventory.json ./inventory.json

# Create certs directory
RUN mkdir -p /root/certs

EXPOSE 443 8080

# Start the service using the standardized name
CMD ["./retail-edge"]