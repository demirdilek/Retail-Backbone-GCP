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
		log.Fatalf("❌ Database connection failed: %v", err)
	}
	defer db.Close()

	// 2. Try to create the schema
	err = database.CreateSchema(db)
	if err != nil {
		log.Fatalf("❌ Schema creation failed: %v", err)
	}
        err = database.SeedDatabase(db, "inventory.json") // Adjust path if needed
        if err != nil {
                log.Fatalf("❌ Seeding failed: %v", err)
        }
	log.Println("🚀 SRE SUCCESS: Database is connected and schema is ready!")
        http.HandleFunc("/product", metricsMiddleware(handleGetProduct(db)))
        http.HandleFunc("/sell", metricsMiddleware(handleSell(db)))
        http.Handle("/", http.FileServer(http.Dir("./web")))
        // Replace the old ListenAndServe with this:
        certFile := "retail-server.tailb0ad6a.ts.net.crt"
        keyFile  := "retail-server.tailb0ad6a.ts.net.key"

        log.Println("🔐 Secure Server starting on https://retail-server.tailb0ad6a.ts.net")
        log.Fatal(http.ListenAndServeTLS(":443", certFile, keyFile, nil))
}
