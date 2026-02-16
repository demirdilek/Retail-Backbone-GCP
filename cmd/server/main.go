package main

import (
	"fmt"
	"io"
	"log"
	"net/http"
	"time"
)

func ingestHandler(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	// Read the EAN from the request body
	body, err := io.ReadAll(r.Body)
	if err != nil {
		http.Error(w, "Failed to read body", http.StatusInternalServerError)
		return
	}
	ean := string(body)

	// Logging with the "Round Timestamp" format
	timestamp := time.Now().Format("2006-01-02 15:04:05")
	fmt.Printf("[%s] 📥 Received EAN from Edge: %s\n", timestamp, ean)

	// Here we will later add the logic to store it in the PC's database
	// For now, we just acknowledge receipt
	w.WriteHeader(http.StatusOK)
	fmt.Fprint(w, "Scan received")
}

func main() {
	http.HandleFunc("/ingest", ingestHandler)

	port := ":8080"
	fmt.Printf("🚀 Central Server (K3s candidate) listening on %s\n", port)
	log.Fatal(http.ListenAndServe(port, nil))
}

