package database

import (
	"database/sql"
	"encoding/json"
	"fmt"
	"log"
	"os"

	_ "github.com/lib/pq" // PostgreSQL driver
)

// Product represents our Master Data structure for Retail Edge
type Product struct {
	EAN           string  `json:"ean"`
	Name          string  `json:"name"`
	PriceEuro     float64 `json:"price_euro"`
	StockQuantity int     `json:"stock_quantity"`
}

// InitDB initializes the Postgres connection and verifies connectivity
func InitDB(dataSourceName string) (*sql.DB, error) {
	db, err := sql.Open("postgres", dataSourceName)
	if err != nil {
		return nil, err
	}

	// SRE Check: Verify the connection is alive
	if err = db.Ping(); err != nil {
		return nil, err
	}

	log.Println("Successfully connected to PostgreSQL")
	return db, nil
}

// CreateSchema ensures all necessary tables exist in the database
func CreateSchema(db *sql.DB) error {
	schema := `
	CREATE TABLE IF NOT EXISTS products (
		ean TEXT PRIMARY KEY,
		name TEXT NOT NULL,
		price_euro DOUBLE PRECISION NOT NULL,
		stock_quantity INTEGER NOT NULL
	);

	CREATE TABLE IF NOT EXISTS sales (
		id SERIAL PRIMARY KEY,
		transaction_id UUID DEFAULT gen_random_uuid(),
		ean TEXT REFERENCES products(ean),
		quantity INTEGER NOT NULL,
		sold_price DOUBLE PRECISION NOT NULL,
		created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP
	);`

	_, err := db.Exec(schema)
	if err != nil {
		return fmt.Errorf("could not create schema: %w", err)
	}

	log.Println("Database schema is up to date")
	return nil
}

// SeedDatabase populates the DB with initial product data from a JSON file
func SeedDatabase(db *sql.DB, filename string) error {
	data, err := os.ReadFile(filename)
	if err != nil {
		return err
	}

	var products []Product
	if err := json.Unmarshal(data, &products); err != nil {
		return err
	}

	for _, p := range products {
		_, err := db.Exec(`
			INSERT INTO products (ean, name, price_euro, stock_quantity)
			VALUES ($1, $2, $3, $4)
			ON CONFLICT (ean) DO UPDATE SET
				name = EXCLUDED.name,
				price_euro = EXCLUDED.price_euro,
				stock_quantity = EXCLUDED.stock_quantity`,
			p.EAN, p.Name, p.PriceEuro, p.StockQuantity)
		if err != nil {
			return err
		}
	}
	log.Printf("✅ Seeded %d products into the database", len(products))
	return nil
}

// GetProductByEAN retrieves a single product by its EAN code
func GetProductByEAN(db *sql.DB, ean string) (*Product, error) {
	p := &Product{}
	query := `SELECT ean, name, price_euro, stock_quantity FROM products WHERE ean = $1`
	err := db.QueryRow(query, ean).Scan(&p.EAN, &p.Name, &p.PriceEuro, &p.StockQuantity)
	if err != nil {
		if err == sql.ErrNoRows {
			return nil, nil // Product not found
		}
		return nil, err
	}
	return p, nil
}

// UpdateStock increases the stock quantity (used for Receiving/Inbound Goods)
func UpdateStock(db *sql.DB, ean string, amount int) error {
	query := `
		UPDATE products 
		SET stock_quantity = stock_quantity + $1 
		WHERE ean = $2`

	result, err := db.Exec(query, amount, ean)
	if err != nil {
		return fmt.Errorf("could not update stock: %v", err)
	}

	rowsAffected, _ := result.RowsAffected()
	if rowsAffected == 0 {
		return fmt.Errorf("no product found with EAN %s", ean)
	}

	return nil
}

// ProcessSale handles the checkout logic: decreases stock and logs the sale
func ProcessSale(db *sql.DB, ean string) (int, error) {
	tx, err := db.Begin()
	if err != nil {
		return 0, err
	}
	defer tx.Rollback()

	// 1. Decrease stock and get the NEW quantity back (atomic)
	var newQuantity int
	err = tx.QueryRow(`
		UPDATE products 
		SET stock_quantity = stock_quantity - 1 
		WHERE ean = $1 AND stock_quantity > 0
		RETURNING stock_quantity`, ean).Scan(&newQuantity)

	if err != nil {
		if err == sql.ErrNoRows {
			return 0, fmt.Errorf("out of stock")
		}
		return 0, err
	}

	// 2. Log the sale record
	_, err = tx.Exec(`
		INSERT INTO sales (ean, quantity, sold_price)
		SELECT ean, 1, price_euro 
		FROM products WHERE ean = $1`, ean)
	if err != nil {
		return 0, err
	}

	return newQuantity, tx.Commit()
}
