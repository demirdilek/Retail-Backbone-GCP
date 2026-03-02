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

	"github.com/demirdilek/Retail-Backbone-GCP/internal/database"
	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promauto"
	"github.com/prometheus/client_golang/prometheus/promhttp"
)

// --- SRE GOLDEN SIGNALS: METRICS ---
var (
	httpDuration = promauto.NewHistogramVec(prometheus.HistogramOpts{
		Name:    "retail_edge_latency_seconds",
		Help:    "Latency of retail operations in seconds.",
		Buckets: prometheus.DefBuckets,
	}, []string{"path", "status"})

	errorCounter = promauto.NewCounterVec(prometheus.CounterOpts{
		Name: "retail_edge_errors_total",
		Help: "Total number of failed retail operations.",
	}, []string{"path", "error_type"})
)

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

func metricsMiddleware(next http.HandlerFunc) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		start := time.Now()
		rw := &responseWriter{ResponseWriter: w, status: http.StatusOK}

		next.ServeHTTP(rw, r)

		duration := time.Since(start)
		httpDuration.WithLabelValues(r.URL.Path, fmt.Sprint(rw.status)).Observe(duration.Seconds())

		if rw.status >= 400 {
			errorCounter.WithLabelValues(r.URL.Path, "http_error").Inc()
		}

		logger.Info("http_request",
			"method", r.Method,
			"path", r.URL.Path,
			"status", rw.status,
			"lat_ms", duration.Milliseconds(),
		)
	}
}

// --- HANDLERS ---

func handleGetProduct(db *sql.DB) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		ean := r.URL.Query().Get("ean")
		if ean == "" {
			http.Error(w, "Missing EAN parameter", http.StatusBadRequest)
			return
		}

		product, err := database.GetProductByEAN(db, ean)
		if err != nil {
			logger.Error("DB lookup failed", "ean", ean, "error", err)
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

// handleCheckout processes the final basket from the frontend
func handleCheckout(db *sql.DB) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
			return
		}

		var req struct {
			Items []string `json:"items"` // Expects array of EANs from frontend
		}

		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			logger.Error("Invalid checkout body", "error", err)
			http.Error(w, "Invalid request body", http.StatusBadRequest)
			return
		}

		// SRE Logic: Iterate through all items and process them as sales
		for _, ean := range req.Items {
			_, err := database.ProcessSale(db, ean)
			if err != nil {
				logger.Error("Checkout item failed", "ean", ean, "error", err)
				errorCounter.WithLabelValues(r.URL.Path, "database_error").Inc()
				http.Error(w, "Checkout failed for item: "+ean, http.StatusInternalServerError)
				return
			}
		}

		logger.Info("Checkout successful", "items_count", len(req.Items))
		w.WriteHeader(http.StatusOK)
		w.Write([]byte(`{"status": "success"}`))
	}
}

func handleRestock(db *sql.DB) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
			return
		}

		var req struct {
			EAN    string `json:"ean"`
			Amount int    `json:"add_amount"`
		}

		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			http.Error(w, "Invalid request body", http.StatusBadRequest)
			return
		}

		err := database.UpdateStock(db, req.EAN, req.Amount)
		if err != nil {
			logger.Error("Restock failed", "ean", req.EAN, "error", err)
			http.Error(w, "Update failed", http.StatusInternalServerError)
			return
		}

		logger.Info("Stock updated", "ean", req.EAN, "added", req.Amount)
		w.WriteHeader(http.StatusOK)
	}
}

// This should be at the top level of your file
func healthHandler(db *sql.DB) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		ctx, cancel := context.WithTimeout(r.Context(), 2*time.Second)
		defer cancel()

		if err := db.PingContext(ctx); err != nil {
			// Ensure logger is defined or use fmt/log for now
			w.WriteHeader(http.StatusServiceUnavailable)
			w.Write([]byte("❌ Service Unavailable"))
			return
		}

		w.WriteHeader(http.StatusOK)
		w.Write([]byte("✅ OK"))
	}
}

func setupInitialMetrics() {
	endpoints := []string{"/product", "/checkout", "/restock"}

	for _, path := range endpoints {
		// Initialisiert den Traffic-Counter (Teil des Histograms) auf 0
		// Wir nutzen status "200", da dies der Normalzustand ist
		httpDuration.WithLabelValues(path, "200").Observe(0)

		// Initialisiert den Error-Counter auf 0
		// Wir nutzen einen generischen Label-Wert, um die Metrik zu registrieren
		errorCounter.WithLabelValues(path, "none").Add(0)
	}
}

// --- MAIN ---

func main() {
	connStr := os.Getenv("DATABASE_URL")
	if connStr == "" {
		connStr = "postgres://retail_user:retail_password@db:5432/retail_backbone?sslmode=disable"
	}

	var db *sql.DB
	var err error
	for i := 0; i < 5; i++ {
		db, err = database.InitDB(connStr)
		if err == nil {
			if err = db.Ping(); err == nil {
				break
			}
		}
		logger.Warn("Database not ready, retrying in 2s...", "attempt", i+1)
		time.Sleep(2 * time.Second)
	}

	if err != nil {
		logger.Error("❌ Critical: DB connection failed", "err", err)
		os.Exit(1)
	}
	defer db.Close()

	// --- DATABASE INITIALIZATION ---

	// Step 1: Ensure the database schema (tables) exists
	if err := database.CreateSchema(db); err != nil {
		logger.Error("❌ Schema creation failed", "err", err)
		os.Exit(1)
	}

	// Step 2: Seed master data from inventory.json
	// We use a warning instead of a fatal error so the server still starts
	// even if the seed file is missing in the container.
	if err := database.SeedDatabase(db, "inventory.json"); err != nil {
		logger.Warn("⚠️ Seeding skipped or failed", "reason", err.Error())
	} else {
		logger.Info("✅ Database master data synchronized")
	}

	setupInitialMetrics()

	// 2. Router Setup
	mux := http.NewServeMux()
	mux.HandleFunc("/product", metricsMiddleware(handleGetProduct(db)))
	mux.HandleFunc("/checkout", metricsMiddleware(handleCheckout(db))) // NEW: Checkout Route
	mux.HandleFunc("/restock", metricsMiddleware(handleRestock(db)))
	mux.HandleFunc("/health", healthHandler(db))
	mux.Handle("/metrics", promhttp.Handler())
	mux.Handle("/", http.FileServer(http.Dir("./web")))

	// 3. Server Configuration (TLS)
	certFile := os.Getenv("CERT_PATH")
	keyFile := os.Getenv("KEY_PATH")

	if certFile == "" || keyFile == "" {
		certFile = "cert.crt"
		keyFile = "cert.key"
	}

	if _, err := os.Stat(certFile); os.IsNotExist(err) {
		logger.Error("❌ Critical: TLS certificates missing", "path", certFile)
		os.Exit(1)
	}

	server := &http.Server{
		Addr:    ":443",
		Handler: mux,
	}

	// 4. Graceful Shutdown
	stop := make(chan os.Signal, 1)
	signal.Notify(stop, os.Interrupt, syscall.SIGTERM)

	go func() {
		logger.Info("🔐 Secure Retail Edge Node starting", "addr", server.Addr)
		if err := server.ListenAndServeTLS(certFile, keyFile); err != nil && err != http.ErrServerClosed {
			logger.Error("Server crash", "err", err)
			os.Exit(1)
		}
	}()

	<-stop
	logger.Info("Graceful shutdown initiated...")

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	if err := server.Shutdown(ctx); err != nil {
		logger.Error("Forced shutdown", "err", err)
	}

	logger.Info("👋 Retail Edge Node stopped.")
}
