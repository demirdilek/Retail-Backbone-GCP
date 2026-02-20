package main

import (
        "fmt"
        "encoding/json"
        "database/sql"
        "net/http"
	"log/slog"
        "os"
        "time"
	"github.com/demirdilek/Retail-Backbone-GCP/internal/database"
)

// Initialize a JSON logger
var logger = slog.New(slog.NewJSONHandler(os.Stdout, nil))

func metricsMiddleware(next http.HandlerFunc) http.HandlerFunc {
    return func(w http.ResponseWriter, r *http.Request) {
        start := time.Now()
        
        // Wrap the response writer to catch the status code (Errors signal)
        rw := &responseWriter{ResponseWriter: w, status: http.StatusOK}
        
        next.ServeHTTP(rw, r)

        // The 4 Golden Signals calculation
        duration := time.Since(start) // LATENCY
        
        logger.Info("http_request",
            "method",  r.Method,
            "path",    r.URL.Path,      // TRAFFIC
            "status",  rw.status,       // ERRORS
            "lat_ms",  duration.Milliseconds(),
            "ua",      r.UserAgent(),   // Identify if S25 or iPhone
        )
    }
}

type responseWriter struct {
    http.ResponseWriter
    status int
}

func (rw *responseWriter) WriteHeader(code int) {
    rw.status = code
    rw.ResponseWriter.WriteHeader(code)
}

func handleSell(db *sql.DB) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			http.Error(w, "Invalid method", 405)
			return
		}
		ean := r.URL.Query().Get("ean")
		newQty, err := database.ProcessSale(db, ean)
		if err != nil {
			http.Error(w, err.Error(), 400)
			return
		}
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
			http.Error(w, "Database error", http.StatusInternalServerError)
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

func main() {
	// Connection string matching your generic docker-compose settings
	connStr := "postgres://retail_user:retail_password@localhost:5432/retail_backbone?sslmode=disable"

	// 1. Try to connect
	db, err := database.InitDB(connStr)
	if err != nil {
		logger.Error("❌ Database connection failed: %v", err)
                os.Exit(1)
	}
	defer db.Close()

	// 2. Try to create the schema
	err = database.CreateSchema(db)
	if err != nil {
		logger.Error("❌ Schema creation failed: %v", err)
                os.Exit(1)
	}
        err = database.SeedDatabase(db, "inventory.json") // Adjust path if needed
        if err != nil {
                logger.Error("❌ Seeding failed: %v", err)
                os.Exit(1)
        }
	logger.Info("🚀 SRE SUCCESS: Database is connected and schema is ready!")
        http.HandleFunc("/product", metricsMiddleware(handleGetProduct(db)))
        http.HandleFunc("/sell", metricsMiddleware(handleSell(db)))
        http.Handle("/", http.FileServer(http.Dir("./web")))
        // Nutzt die Umgebungsvariable, falls gesetzt, sonst den Standardpfad
        certFile := os.Getenv("CERT_PATH")
        if certFile == "" {
            certFile = "/etc/ssl/certs/cert.pem" // Dein Standard auf dem HP
        }

        keyFile := os.Getenv("KEY_PATH")
        if keyFile == "" {
        keyFile = "/etc/ssl/private/key.pem"
        }       
        // Log the startup attempt
        logger.Info("🔐 Secure Server starting", "url", "https://retail-server.tailb0ad6a.ts.net")

        // Start the server synchronously to catch the error immediately
        err = http.ListenAndServeTLS(":443", certFile, keyFile, nil)
        if err != nil {
            // Critical: If the server fails to start, we must know why
            logger.Error("Server crashed during startup", "reason", err.Error())
    
            // Fallback: Print to standard error in case the logger fails
            os.Stderr.WriteString("CRITICAL: " + err.Error() + "\n")
            os.Exit(1)
        }
}
