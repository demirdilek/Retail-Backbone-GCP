package main

import (
	"bytes"
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"log/slog"
	"net/http"
	"os"
	"os/signal"
	"strings"
	"syscall"
	"time"

	"github.com/demirdilek/Retail-Backbone-GCP/internal/database"
	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promauto"
	"github.com/prometheus/client_golang/prometheus/promhttp"
)

// --- SRE GOLDEN SIGNALS: METRICS ---
var (
	// Jetzt als GaugeVec, um nach Filiale (Namespace) zu filtern
	unsyncedSalesGauge = promauto.NewGaugeVec(prometheus.GaugeOpts{
		Name: "retail_edge_unsynced_sales_count",
		Help: "Number of sales records pending synchronization to GCP.",
	}, []string{"namespace"}) // Wichtig für die Filial-Trennung

	httpDuration = promauto.NewHistogramVec(prometheus.HistogramOpts{
		Name:    "retail_edge_latency_seconds",
		Help:    "Latency of retail operations in seconds.",
		Buckets: prometheus.DefBuckets,
	}, []string{"path", "status"})

	errorCounter = promauto.NewCounterVec(prometheus.CounterOpts{
		Name: "retail_edge_errors_total",
		Help: "Total number of failed retail operations.",
	}, []string{"path", "error_type"})

	logger = slog.New(slog.NewJSONHandler(os.Stdout, nil))
)

// StartSyncWorker initiates an asynchronous background routine to synchronize
// local sales data with the GCP Backbone. This ensures offline-first resilience.
func StartSyncWorker(db *sql.DB, backboneURL string, edgeNodeID string) {
	if backboneURL == "" {
		logger.Warn("SRE-Sync: No GCP_BACKBONE_URL provided. Background sync is disabled.")
		return
	}

	namespace := os.Getenv("K8S_NAMESPACE")
	if namespace == "" {
		namespace = "unknown"
	}

	// Ticker defines the synchronization interval (Golden Signal: Freshness)
	ticker := time.NewTicker(30 * time.Second)

	go func() {
		logger.Info("SRE-Sync: background worker started", "target", backboneURL)
		for range ticker.C {
			// anonymous function to ensure 'defer' triggers at the end of each tick
			func() {
				// Final count update at the end of the tick for better performance
				defer func() {
					var finalCount float64
					if err := db.QueryRow("SELECT COUNT(*) FROM sales WHERE synced_at IS NULL").Scan(&finalCount); err == nil {
						unsyncedSalesGauge.WithLabelValues(namespace).Set(finalCount)
					}
				}()

				ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
				defer cancel()

				// Retrieve unsynced sales records
				rows, err := db.QueryContext(ctx, `
                    SELECT id, transaction_id, ean, quantity, sold_price 
                    FROM sales 
                    WHERE synced_at IS NULL 
                    LIMIT 10`)
				if err != nil {
					logger.Error("SRE-Sync: database query failed", "error", err)
					return // ends current tick function
				}
				defer rows.Close()

				for rows.Next() {
					var id int
					var txID, ean string
					var qty int
					var price float64

					if err := rows.Scan(&id, &txID, &ean, &qty, &price); err != nil {
						continue
					}

					payload := map[string]interface{}{
						"transaction_id": txID,
						"ean":            ean,
						"quantity":       qty,
						"price":          price,
						"edge_node":      edgeNodeID,
					}

					// Attempt to send data to GCP Backbone
					if err := sendToGCP(payload, backboneURL); err != nil {
						logger.Warn("SRE-Sync: GCP Backbone unreachable", "error", err)
						break // exit the row-processing for this tick (simple backoff)
					}

					// Reconcile local record
					_, err = db.Exec("UPDATE sales SET synced_at = NOW() WHERE id = $1", id)
					if err == nil {
						logger.Info("SRE-Sync: transaction reconciled", "tx_id", txID)
					}
				}
			}()
		}
	}()
}

func sendToGCP(data interface{}, url string) error {
	jsonData, _ := json.Marshal(data)
	client := &http.Client{Timeout: 5 * time.Second}

	// Korrektur: Nur einmal /api/v1/sync anhängen
	cleanURL := strings.TrimSuffix(url, "/") + "/api/v1/sync"

	resp, err := client.Post(cleanURL, "application/json", bytes.NewBuffer(jsonData))
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	if resp.StatusCode >= 400 {
		return fmt.Errorf("backbone returned status: %d", resp.StatusCode)
	}
	return nil
}

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
			w.Write([]byte("Service Unavailable"))
			return
		}

		w.WriteHeader(http.StatusOK)
		w.Write([]byte("OK"))
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
func initializeSelfMetrics() {
	//Read the namespace from environment variable, default to "local-dev" if not set
	namespace := os.Getenv("K8S_NAMESPACE")
	if namespace == "" {
		namespace = "local-dev"
	}

	//Initialize the unsynced sales gauge for this namespace to 0 at startup
	unsyncedSalesGauge.WithLabelValues(namespace).Set(0)

	logger.Info("SRE-Metrics: Registered self in Prometheus", "namespace", namespace)
}

// --- MAIN ---

func main() {
	pwd, _ := os.Getwd()
	logger.Info("SRE-Debug", "working_dir", pwd)

	if _, err := os.Stat("web/index.html"); os.IsNotExist(err) {
		logger.Error("Critical: index.html not found in ./web/")
	}

	connStr := os.Getenv("DB_DSN")
	var db *sql.DB
	var err error
	isDemoMode := false

	if connStr == "" || connStr == "mock" {
		logger.Warn("SRE-Alert: DB_DSN not set or 'mock'. Starting in DEMO MODE (In-Memory).")
		isDemoMode = true
	} else {
		// Normal DB connection with retries
		for i := 0; i < 5; i++ { // 5 times is enough for a startup retry
			db, err = database.InitDB(connStr)
			if err == nil {
				if err = db.Ping(); err == nil {
					logger.Info("Database connection established successfully")
					break
				}
			}
			logger.Warn("Database not ready, retrying in 2s...", "attempt", i+1)
			time.Sleep(2 * time.Second)
		}
	}

	// Wenn nach den Versuchen kein DB-Connect zustande kam
	if err != nil {
		logger.Error("DB connection failed, falling back to DEMO MODE", "err", err)
		isDemoMode = true
	}

	if !isDemoMode {
		defer db.Close()
	}

	// --- DATABASE INITIALIZATION ---

	// Step 1: Ensure the database schema (tables) exists
	if err := database.CreateSchema(db); err != nil {
		logger.Error("Schema creation failed", "err", err)
		os.Exit(1)
	}

	// Step 2: Seed master data from inventory.json
	// We use a warning instead of a fatal error so the server still starts
	// even if the seed file is missing in the container.
	if err := database.SeedDatabase(db, "/data/inventory.json"); err != nil {
		logger.Warn("Seeding skipped or failed", "reason", err.Error())
	} else {
		logger.Info("Database master data synchronized")
	}

	// START SYNC WORKER: Decouple edge sales from cloud persistence
	backboneURL := os.Getenv("GCP_BACKBONE_URL")
	if backboneURL == "" {
		logger.Warn("Backbone URL is missing, falling back to unknown")
		backboneURL = "unknown-url"
	}

	// Fetch the Edge Node ID from the environment, provide fallback
	edgeNodeID := os.Getenv("EDGE_NODE_ID")
	if edgeNodeID == "" {
		logger.Warn("EDGE_NODE_ID is missing, falling back to unknown")
		edgeNodeID = "unknown-edge-node"
	}

	// Initialize self-metrics before starting the worker to ensure the namespace label is set
	initializeSelfMetrics()

	// Pass the ID and URL to the worker
	StartSyncWorker(db, backboneURL, edgeNodeID)

	setupInitialMetrics()

	// 2. Router Setup
	mux := http.NewServeMux()
	mux.HandleFunc("/product", metricsMiddleware(handleGetProduct(db)))
	mux.HandleFunc("/checkout", metricsMiddleware(handleCheckout(db))) // NEW: Checkout Route
	mux.HandleFunc("/restock", metricsMiddleware(handleRestock(db)))
	mux.HandleFunc("/healthz", healthHandler(db))
	mux.Handle("/metrics", promhttp.Handler())
	mux.Handle("/", http.FileServer(http.Dir("./web")))

	//This endpoint simulates a CPU stress test by performing meaningless calculations for 5 seconds. It can be used to test how the system behaves under high CPU load and to validate the effectiveness of monitoring and alerting based on CPU usage.
	mux.HandleFunc("/debug/stress-cpu", func(w http.ResponseWriter, r *http.Request) {
		// Berechnet für 5 Sekunden sinnlose Primzahlen, um CPU zu fressen
		done := time.After(5 * time.Second)
		for {
			select {
			case <-done:
				w.Write([]byte("CPU Stress finished"))
				return
			default:
				_ = 1234567 * 7654321
			}
		}
	})

	//This endpoint simulates a memory leak by allocating 50MB of RAM and keeping it in memory. Each call to this endpoint will increase the memory usage of the server, which can be useful for testing how the system behaves under memory pressure.
	var leak [][]byte
	mux.HandleFunc("/debug/stress-mem", func(w http.ResponseWriter, r *http.Request) {
		// Reserviert ca. 50MB RAM und behält sie im Speicher
		s := make([]byte, 50*1024*1024)
		leak = append(leak, s)
		w.Write([]byte("Allocated 50MB of Heap"))
	})

	// 3. Server Configuration (TLS)
	certFile := os.Getenv("CERT_PATH")
	keyFile := os.Getenv("KEY_PATH")

	if certFile == "" || keyFile == "" {
		certFile = "cert.crt"
		keyFile = "cert.key"
	}

	if _, err := os.Stat(certFile); os.IsNotExist(err) {
		logger.Error("Critical: TLS certificates missing", "path", certFile)
	}
	server := &http.Server{
		Handler: mux,
	}

	stop := make(chan os.Signal, 1)
	signal.Notify(stop, os.Interrupt, syscall.SIGTERM)

	go func() {
		logger.Info("SRE-Operations: SyncWorker online",
			"node_id", edgeNodeID,
			"port", "8080",
			"mode", "insecure/local",
		)

		server.Addr = ":8080"
		if err := server.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			logger.Error("HTTP Server crash", "err", err)
		}
	}()

	<-stop
	logger.Info("Graceful shutdown initiated...")

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	if err := server.Shutdown(ctx); err != nil {
		logger.Error("Forced shutdown", "err", err)
	}

	logger.Info("Retail Edge Node stopped.")
}
