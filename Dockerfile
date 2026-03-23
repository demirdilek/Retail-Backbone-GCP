# --- Stage 1: Build ---
FROM golang:1.25-alpine AS builder
RUN apk add --no-cache git ca-certificates
WORKDIR /app

# 1. Abhängigkeiten laden (Optimiertes Caching)
COPY go.mod go.sum ./
RUN go mod download

# 2. EXPLIZIT: Daten-Ordner kopieren, damit er im Builder existiert
COPY data/ ./data/

# 3. Restlichen Code kopieren
COPY . .

# 4. Binary bauen
RUN CGO_ENABLED=0 GOOS=linux go build -ldflags="-s -w" -o bin/retail-syncworker ./cmd/server/main.go

# --- Stage 2: Final (Distroless) ---
FROM gcr.io/distroless/static-debian12:latest
WORKDIR /app

# 5. Binary und Daten aus dem Builder in das finale Image übertragen
COPY --from=builder /app/bin/retail-syncworker ./retail-syncworker
COPY --from=builder /app/data ./data

# Falls du keinen 'web' Ordner hast, lösche diese Zeile unbedingt!
# COPY --from=builder /app/web ./web

USER nonroot:nonroot
EXPOSE 8080
ENTRYPOINT ["./retail-syncworker"]