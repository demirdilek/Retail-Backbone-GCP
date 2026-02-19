package main

import (
        "fmt"
        "encoding/json"
        "database/sql"
        "net/http"
	"log"
	"github.com/demirdilek/Retail-Backbone-GCP/internal/database"
)

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
        http.HandleFunc("/product", handleGetProduct(db))
        http.HandleFunc("/sell", handleSell(db))
        http.Handle("/", http.FileServer(http.Dir("./web")))
        // Replace the old ListenAndServe with this:
        certFile := "retail-server.tailb0ad6a.ts.net.crt"
        keyFile  := "retail-server.tailb0ad6a.ts.net.key"

        log.Println("🔐 Secure Server starting on https://retail-server.tailb0ad6a.ts.net")
        log.Fatal(http.ListenAndServeTLS(":443", certFile, keyFile, nil))
}
