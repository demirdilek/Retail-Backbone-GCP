package main

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"log/slog"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	// Ensure this path matches your go.mod
	"github.com/demirdilek/Retail-Backbone-GCP/internal/database"

	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promauto"
	"github.com/prometheus/client_golang/prometheus/promhttp"
)

// --- SRE GOLDEN SIGNALS: METRICS ---
var (
	// LATENCY & TRAFFIC: Tracks request duration and total hits
	httpDuration = promauto.NewHistogramVec(prometheus.HistogramOpts{
		Name:    "retail_scanner_latency_seconds",
		Help:    "Latency of scanner operations in seconds.",
		Buckets: prometheus.DefBuckets,
	}, []string{"path", "status"})

	// ERRORS: Tracks failed operations
	errorCounter = promauto.NewCounterVec(prometheus.CounterOpts{
		Name: "retail_scanner_errors_total",
		Help: "Total number of failed retail operations.",
	}, []string{"path", "error_type"})
)

// Initialize a JSON logger for centralized observability
var logger = slog.New(slog.NewJSONHandler(os.Stdout, nil))

// --- MIDDLEWARE ---

type responseWriter struct {
	http.ResponseWriter
	status int
}

func (rw *responseWriter) WriteHeader(code int) {
	rw.status = code
	rw.ResponseWriter.WriteHeader(code)
}

// metricsMiddleware captures Golden Signals for every request
func metricsMiddleware(next http.HandlerFunc) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		start := time.Now()
		rw := &responseWriter{ResponseWriter: w, status: http.StatusOK}

		next.ServeHTTP(rw, r)

		duration := time.Since(start)

		// Record Latency & Traffic
		httpDuration.WithLabelValues(r.URL.Path, fmt.Sprint(rw.status)).Observe(duration.Seconds())

		// Record Errors if status code is 4xx or 5xx
		if rw.status >= 400 {
			errorCounter.WithLabelValues(r.URL.Path, "http_error").Inc()
		}

		logger.Info("http_request",
			"method", r.Method,
			"path", r.URL.Path,
			"status", rw.status,
			"lat_ms", duration.Milliseconds(),
			"ua", r.UserAgent(),
		)
	}
}

// --- HANDLERS ---

func handleSell(db *sql.DB) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
			return
		}

		ean := r.URL.Query().Get("ean")
		if ean == "" {
			http.Error(w, "Missing EAN parameter", http.StatusBadRequest)
			return
		}

		// Use the logic from your internal/database package
		newQty, err := database.ProcessSale(db, ean)
		if err != nil {
			logger.Error("Sale failed", "ean", ean, "error", err)
			errorCounter.WithLabelValues(r.URL.Path, "database_error").Inc()
			http.Error(w, err.Error(), http.StatusBadRequest)
			return
		}

		w.Header().Set("Content-Type", "application/json")
		fmt.Fprintf(w, `{"new_stock": %d}`, newQty)
	}
}

func handleGetProduct(db *sql.DB) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		ean := r.URL.Query().Get("ean")
		if ean == "" {
			http.Error(w, "Missing EAN parameter", http.StatusBadRequest)
			return
		}

		product, err := database.GetProductByEAN(db, ean)
		if err != nil {
			logger.Error("Database error", "ean", ean, "error", err)
			http.Error(w, "Internal server error", http.StatusInternalServerError)
			return
		}

		if product == nil {
			http.Error(w, "Product not found", http.StatusNotFound)
			return
		}

		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(product)
	}
}

func healthHandler(db *sql.DB) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		// Prüfe, ob die Datenbank erreichbar ist
		err := db.Ping()
		if err != nil {
			w.WriteHeader(http.StatusServiceUnavailable)
			w.Write([]byte("❌ Database unreachable"))
			return
		}

		w.WriteHeader(http.StatusOK)
		w.Write([]byte("✅ OK"))
	}
}

// --- MAIN ---

func main() {
	// Database configuration
	connStr := os.Getenv("DATABASE_URL")
	if connStr == "" {
		connStr = "postgres://retail_user:retail_password@localhost:5432/retail_backbone?sslmode=disable"
	}

	// 1. Database Setup
	db, err := database.InitDB(connStr)
	if err != nil {
		logger.Error("❌ DB connection failed", "err", err)
		os.Exit(1)
	}
	defer db.Close()

	if err := database.CreateSchema(db); err != nil {
		logger.Error("❌ Schema creation failed", "err", err)
		os.Exit(1)
	}

	if err := database.SeedDatabase(db, "inventory.json"); err != nil {
		logger.Warn("⚠️ Seeding failed or skipped", "err", err)
	}

	logger.Info("🚀 SRE SUCCESS: Database is ready!")

	// 2. Router Setup
	mux := http.NewServeMux()
	mux.HandleFunc("/product", metricsMiddleware(handleGetProduct(db)))
	mux.HandleFunc("/sell", metricsMiddleware(handleSell(db)))
	mux.Handle("/metrics", promhttp.Handler()) // Essential for Prometheus scraping
	mux.HandleFunc("/health", metricsMiddleware(healthHandler(db)))
	mux.Handle("/", http.FileServer(http.Dir("./web")))

	// 3. Server Configuration (TLS/Tailscale)
	certFile := os.Getenv("CERT_PATH")
	keyFile := os.Getenv("KEY_PATH")
	if certFile == "" || keyFile == "" {
		certFile = "/etc/ssl/certs/cert.pem" // HP Server Default
		keyFile = "/etc/ssl/private/key.pem"
	}

	server := &http.Server{
		Addr:    ":443",
		Handler: mux,
	}

	// 4. Graceful Shutdown Logic (SRE Standard)
	stop := make(chan os.Signal, 1)
	signal.Notify(stop, os.Interrupt, syscall.SIGTERM)

	go func() {
		logger.Info("🔐 Secure Server starting", "addr", server.Addr)
		if err := server.ListenAndServeTLS(certFile, keyFile); err != nil && err != http.ErrServerClosed {
			logger.Error("Server crashed", "err", err)
			os.Exit(1)
		}
	}()

	// Wait for termination signal
	<-stop
	logger.Info("Shutting down gracefully...")

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	if err := server.Shutdown(ctx); err != nil {
		logger.Error("Shutdown forced", "err", err)
	}

	logger.Info("👋 Server stopped.")
}
