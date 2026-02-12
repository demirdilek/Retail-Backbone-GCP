package main

import (
	"database/sql"
	"fmt"
	_ "github.com/mattn/go-sqlite3" // SQLite Treiber
)

// SetupDatabase erstellt die lokale Tabelle für die Warenbestands-Events
func SetupDatabase(dbPath string) (*sql.DB, error) {
	db, err := sql.Open("sqlite3", dbPath)
	if err != nil {
		return nil, err
	}

	// Eine Tabelle für Bestandsänderungen (z.B. Palette Äpfel eingetroffen)
	statement := `
	CREATE TABLE IF NOT EXISTS stock_events (
		id INTEGER PRIMARY KEY AUTOINCREMENT,
		item_name TEXT,
		quantity INTEGER,
		timestamp DATETIME DEFAULT CURRENT_TIMESTAMP,
		synced_to_gcp BOOLEAN DEFAULT 0
	);`

	_, err = db.Exec(statement)
	return db, err
}
