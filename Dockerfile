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

# Expose the HTTPS port
EXPOSE 443