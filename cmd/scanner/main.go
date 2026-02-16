package main

import (
	"fmt"
	"net/http"
	"os/exec"
	"strings"
	"time"
)

// Configuration: Replace with your PC's Tailscale IP
const pcEndpoint = "http://100.x.y.z:8080/ingest"

func main() {
	fmt.Println("[SRE-LOG] 📡 Scanner mode active. Forwarding data to PC...")
	lastClipboard := ""

	for {
		// 1. Fetch current clipboard content via Termux API
		out, err := exec.Command("/data/data/com.termux/files/usr/bin/termux-clipboard-get").Output()
		if err != nil {
			// Quietly retry if the API bridge is temporarily unavailable
			time.Sleep(2 * time.Second)
			continue
		}

		ean := strings.TrimSpace(string(out))

		// 2. Validate: Only send if it's a new scan and meets minimum EAN length
		if ean != "" && ean != lastClipboard && len(ean) >= 8 {
			lastClipboard = ean
			
			// 3. Transmit data to the central PC server
			resp, err := http.Post(pcEndpoint, "text/plain", strings.NewReader(ean))
			
			timestamp := time.Now().Format("15:04:05")
			if err == nil && resp.StatusCode == http.StatusOK {
				fmt.Printf("[%s] ✅ Success: EAN %s transmitted to server\n", timestamp, ean)
			} else {
				// Fallback: In a production SRE scenario, we would implement local buffering here
				fmt.Printf("[%s] ❌ Error: Server unreachable. EAN: %s\n", timestamp, ean)
			}
		}
		
		// 4. Polling interval to balance responsiveness and battery life
		time.Sleep(1 * time.Second)
	}
}

