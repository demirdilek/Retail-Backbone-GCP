# --- Stage 1: Build ---
FROM golang:1.25-alpine AS builder
RUN apk add --no-cache git ca-certificates
WORKDIR /app
COPY go.mod go.sum ./
RUN go mod download
COPY . .

# Build with the name defined in your Makefile
RUN CGO_ENABLED=0 GOOS=linux go build -ldflags="-s -w" -o bin/retail-syncworker ./cmd/server/main.go

# --- Stage 2: Final ---
FROM gcr.io/distroless/static-debian12:latest
WORKDIR /app

# Copy binary from the specific 'bin' folder created in Stage 1
COPY --from=builder /app/bin/retail-syncworker ./retail-syncworker

# Copy your data and web assets
# Note: Ensure these directories exist in your repo root!
COPY --from=builder /app/data ./data
COPY --from=builder /app/web ./web

USER nonroot:nonroot
EXPOSE 8080
ENTRYPOINT ["./retail-syncworker"]