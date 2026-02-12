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
		synced_to_gcp BOOLEAN DEFAULT 0, -- New column for sync status
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

// SyncPendingEvents finds unsynced records and marks them as synced
func SyncPendingEvents(db *sql.DB) {
	fmt.Println("🔄 Sync-Manager: Checking for unsynced events...")

	// 1. Fetch all unsynced movements
	rows, err := db.Query("SELECT id, item_name, quantity FROM stock_movements WHERE synced_to_gcp = 0")
	if err != nil {
		log.Printf("Sync error: %v", err)
		return
	}
	defer rows.Close()

	for rows.Next() {
		var id int
		var item string
		var qty int
		rows.Scan(&id, &item, &qty)

		// 2. Simulate Cloud Upload
		fmt.Printf("☁️  Uploading: [ID: %d] %d x %s to GCP Pub/Sub...\n", id, qty, item)

		// 3. Mark as synced in local DB
		updateSQL := `UPDATE stock_movements SET synced_to_gcp = 1 WHERE id = ?`
		_, err := db.Exec(updateSQL, id)
		if err != nil {
			log.Printf("Failed to update sync status for ID %d: %v", id, err)
		} else {
			fmt.Printf("✅ Event %d marked as synced.\n", id)
		}
	}
}

