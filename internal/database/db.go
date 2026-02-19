package database

import (
    "database/sql"
    "fmt"
    "log"
    "encoding/json"
    "os"
    _ "github.com/lib/pq" // PostgreSQL driver
)

// Product represents our Master Data structure
type Product struct {
        EAN             string  `json:"ean"`
        Name            string  `json:"name"`
        PriceEuro       float64 `json:"price_euro"`
        StockQuantity   int     `json:"stock_quantity"`
}

// InitDB initializes the Postgres connectio and creates tables
func InitDB(dataSourceName string) (*sql.DB, error) {
        db, err := sql.Open("postgres", dataSourceName)
        if err != nil {
                return nil, err
        }

        //Verify the connection is alive
        if err = db.Ping(); err != nil {
                return nil, err
        }
        
        log.Println("Successfully connected to PostgresSQL")
        return db, nil
}

//CreateSchema creates the necessary tables if the don't exist
func CreateSchema(db *sql.DB) error {
        schema := `
        CREATE TABLE IF NOT EXISTS products (
                ean TEXT PRIMARY KEY,
                name TEXT NOT NULL,
                price_euro DOUBLE PRECISION NOT NULL,
                stock_quantity INTEGER NOT NULL
        );

        CREATE TABLE IF NOT EXISTS transactions (
            id SERIAL PRIMARY KEY,
            ean TEXT REFERENCES products(ean),
            timestamp TIMESTAMP DEFAULT CURRENT_TIMESTAMP
        );

        CREATE TABLE IF NOT EXISTS sales (
            id SERIAL PRIMARY KEY,
            transaction_id UUID NOT NULL,
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
func GetProductByEAN(db *sql.DB, ean string) (*Product, error) {
	p := &Product{}
	query := `SELECT ean, name, price_euro, stock_quantity FROM products WHERE ean = $1`
	err := db.QueryRow(query, ean).Scan(&p.EAN, &p.Name, &p.PriceEuro, &p.StockQuantity)
	if err != nil {
		if err == sql.ErrNoRows {
			return nil, nil // Not found
		}
		return nil, err
	}
	return p, nil
}
func ProcessSale(db *sql.DB, ean string) (int, error) {
	tx, err := db.Begin()
	if err != nil {
		return 0, err
	}
	defer tx.Rollback()

	// 1. Decrease stock and get the NEW quantity back
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

	// 2. Log the sale
	_, err = tx.Exec(`
		INSERT INTO sales (transaction_id, ean, quantity, sold_price)
		SELECT gen_random_uuid(), ean, 1, price_euro 
		FROM products WHERE ean = $1`, ean)
	if err != nil {
		return 0, err
	}

	return newQuantity, tx.Commit()
}
