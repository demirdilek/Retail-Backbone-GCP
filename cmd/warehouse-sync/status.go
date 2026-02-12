package main

import (
	"database/sql"
	"fmt"
	_ "github.com/mattn/go-sqlite3"
	"log"
)

func main() {
	db, err := sql.Open("sqlite3", "./warehouse_edge.db")
	if err != nil {
		log.Fatal(err)
	}
	defer db.Close()

	var total, synced, pending int

	// Count total movements
	db.QueryRow("SELECT COUNT(*) FROM stock_movements").Scan(&total)
	// Count already synced
	db.QueryRow("SELECT COUNT(*) FROM stock_movements WHERE synced_to_gcp = 1").Scan(&synced)
	
	pending = total - synced

	fmt.Println("\n📊 BACKBONE NODE HEALTH REPORT")
	fmt.Println("================================")
	fmt.Printf("Total Events recorded:  %d\n", total)
	fmt.Printf("Status Synced (GCP):    %d ✅\n", synced)
	fmt.Printf("Status Pending:         %d ⏳\n", pending)
	
	if pending > 0 {
		fmt.Printf("Health: ⚠️  Backlog detected (%d items)\n", pending)
	} else {
		fmt.Println("Health: 🟢 All systems nominal. Edge and Cloud are in sync.")
	}
	fmt.Println("================================\n")
}

