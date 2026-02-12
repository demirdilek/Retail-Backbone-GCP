package main

import (
	"database/sql"
	"fmt"
	_ "github.com/mattn/go-sqlite3"
	"log"
	"os"
)

func main() {
	// Define the local database path
	dbPath := "./warehouse_edge.db"

	fmt.Println("🚀 Retail-Backbone: Initializing Edge-Node...")

	// Open database connection
	db, err := sql.Open("sqlite3", dbPath)
	if err != nil {
		log.Fatalf("Failed to connect to database: %v", err)
	}
	defer db.Close()

	// Create table for stock movements if it doesn't exist
	// This represents our local buffer before syncing to GCP
	createTableSQL := `
	CREATE TABLE IF NOT EXISTS stock_movements (
		id INTEGER PRIMARY KEY AUTOINCREMENT,
		item_name TEXT,
		quantity INTEGER,
		created_at DATETIME DEFAULT CURRENT_TIMESTAMP
	);`

	_, err = db.Exec(createTableSQL)
	if err != nil {
		log.Fatalf("Failed to create table: %v", err)
	}

	// Insert a dummy record to simulate a warehouse scan
	// Example: A pallet of organic apples arrives
	itemName := "Organic Apples"
	quantity := 50

	insertSQL := `INSERT INTO stock_movements (item_name, quantity) VALUES (?, ?)`
	_, err = db.Exec(insertSQL, itemName, quantity)
	if err != nil {
		log.Fatalf("Failed to insert data: %v", err)
	os.Exit(1)
	}

	fmt.Printf("✅ Success: Recorded movement of %d %s\n", quantity, itemName)
}

