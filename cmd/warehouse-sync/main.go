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
	SyncPendingEvents(db)
}
func SyncPendingEvents(db *sql.DB) {
    fmt.Println("🔄 Sync-Manager: Checking for unsynced events...")

    // 1. Collect IDs of unsynced records
    rows, err := db.Query("SELECT id, item_name, quantity FROM stock_movements WHERE synced_to_gcp = 0")
    if err != nil {
        log.Printf("Query error: %v", err)
        return
    }

    type movement struct {
        id   int
        item string
        qty  int
    }
    var toSync []movement

    for rows.Next() {
        var m movement
        if err := rows.Scan(&m.id, &m.item, &m.qty); err != nil {
            continue
        }
        toSync = append(toSync, m)
    }
    rows.Close() // Explicitly close the reader before starting writers!

    // 2. Iterate over the collected records and sync them
    for _, m := range toSync {
        fmt.Printf("☁️  Uploading: [ID: %d] %d x %s to GCP Pub/Sub...\n", m.id, m.qty, m.item)
        
        // Simulating network success...
        _, err := db.Exec("UPDATE stock_movements SET synced_to_gcp = 1 WHERE id = ?", m.id)
        if err != nil {
            log.Printf("Failed to update ID %d: %v", m.id, err)
        } else {
            fmt.Printf("✅ Event %d marked as synced.\n", m.id)
        }
    }
}
	
