.PHONY: run ship

# Lokal starten mit Umgebungsvariablen
run:
	sudo CERT_PATH="./retail-server.tailb0ad6a.ts.net.crt" \
	     KEY_PATH="./retail-server.tailb0ad6a.ts.net.key" \
	     /usr/local/go/bin/go run cmd/server/main.go
# Den SRE-Workflow automatisieren
ship:
	go build ./cmd/server/main.go
	git add .
	git commit -m "refactor: use env vars for config and update docs"
	git push origin main
