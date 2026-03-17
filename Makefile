# Retail Edge - SRE Infrastructure Makefile
# Guiding Principle: Events over Timeouts

.DEFAULT_GOAL := help

# Configuration variables
CLUSTER_NAME = retail-cluster
STORES = store-betzdorf store-remagen store-koeln

.PHONY: demo cluster-up mesh-up deploy-stores clean deploy-monitoring status help warmup tunnels open-stores

# --- Documentation ---

help:
	@echo "Retail Edge - SRE Deployment Menu"
	@echo "---------------------------------"
	@echo "make demo              - Full automated setup with readiness checks"
	@echo "make cluster-up        - Create k3d cluster and Gateway API"
	@echo "make mesh-up           - Install Linkerd service mesh"
	@echo "make deploy-stores     - Deploy stores and wait for readiness"
	@echo "make deploy-monitoring - Deploy observability stack"
	@echo "make status            - Check Golden Signals"
	@echo "make clean             - Full teardown"

# --- Main Workflows ---

demo:
	@start_time=$$(date +%s); \
	$(MAKE) cluster-up; \
	$(MAKE) mesh-up; \
	$(MAKE) deploy-stores; \
	$(MAKE) deploy-monitoring; \
	$(MAKE) tunnels; \
	echo "SRE Check: Validating tunnel stability..."; \
	for i in $$(seq 1 10); do \
		if timeout 1 bash -c 'cat < /dev/null > /dev/tcp/localhost/8081' 2>/dev/null; then \
			echo "Tunnels established and verified."; \
			break; \
		fi; \
		if [ $$i -eq 10 ]; then echo "Tunnels failed to stabilize"; exit 1; fi; \
		echo "Waiting for connectivity ($$i/10)..."; \
		sleep 1; \
	done; \
	$(MAKE) warmup; \
	$(MAKE) open-stores; \
	echo "Setup complete. Starting Linkerd dashboard on Port 5050..."; \
	linkerd viz dashboard --port 5050 & \
	duration=$$(( $$(date +%s) - start_time )); \
	echo "------------------------------------------------"; \
	echo "RETAIL EDGE READY IN $$duration SECONDS"; \
	echo "------------------------------------------------"

warmup:
	@echo "SRE Operations: Warming up store caches..."
	@port=8081; \
	for store in $(STORES); do \
		echo "Hitting $$store on port $$port..."; \
		curl -s "http://localhost:$$port/product?ean=4012345678901" > /dev/null || true; \
		port=$$((port + 1)); \
	done

# --- Infrastructure Components ---

cluster-up:
	@k3d cluster get $(CLUSTER_NAME) >/dev/null 2>&1 || \
	k3d cluster create $(CLUSTER_NAME) \
		-p "8443:443@loadbalancer" \
		-p "3000:3000@loadbalancer" \
		-p "8080:80@loadbalancer"
	@k3d kubeconfig merge $(CLUSTER_NAME) --kubeconfig-switch-context --kubeconfig-merge-default
	@kubectl apply -f https://github.com/kubernetes-sigs/gateway-api/releases/download/v1.0.0/standard-install.yaml
	@echo "Waiting for Cluster Nodes..."
	@kubectl wait --for=condition=Ready nodes --all --timeout=60s

mesh-up:
	@kubectl get namespace linkerd >/dev/null 2>&1 || ( \
		linkerd install --crds | kubectl apply -f - && \
		linkerd install | kubectl apply -f - && \
		linkerd viz install | kubectl apply -f - \
	)
	@kubectl wait --for=condition=available deployment --all -n linkerd-viz --timeout=120s

deploy-stores:
	@echo "Building application and importing into k3d..."
	@docker build -t retail-edge:latest .
	@k3d image import retail-edge:latest -c $(CLUSTER_NAME)
	@kubectl apply -f k8s/gateway.yaml
	@for store in $(STORES); do \
		echo "Deploying $$store..."; \
		kubectl create namespace $$store --dry-run=client -o yaml | kubectl apply -f -; \
		kubectl annotate namespace $$store linkerd.io/inject=enabled --overwrite; \
		kubectl apply -f k8s/db.yaml -n $$store; \
		kubectl apply -f k8s/app.yaml -n $$store; \
		sed "s/STORE_NAME/$$store/g" k8s/routes.yaml | kubectl apply -f - -n $$store; \
		kubectl wait --for=condition=available deployment/retail-edge -n $$store --timeout=60s; \
	done

deploy-monitoring:
	@kubectl create namespace monitoring --dry-run=client -o yaml | kubectl apply -f -
	@kubectl create configmap retail-edge-dashboard \
		--from-file=monitoring/grafana/provisioning/dashboards/retail-edge.json \
		-n monitoring --dry-run=client -o yaml | kubectl apply -f -
	@kubectl apply -f k8s/prometheus.yaml -n monitoring
	@kubectl apply -f k8s/grafana.yaml -n monitoring
	@echo "Waiting for Observability Readiness..."
	@kubectl wait --for=condition=available deployment/prometheus -n monitoring --timeout=60s
	@kubectl wait --for=condition=available deployment/grafana -n monitoring --timeout=60s
	@-pkill -f "port-forward svc/grafana" > /dev/null 2>&1 || true
	@nohup kubectl port-forward svc/grafana 3000:3000 -n monitoring > /dev/null 2>&1 &

tunnels:
	@echo "SRE Operations: Resetting and opening store tunnels..."
	@-pkill -f "port-forward" > /dev/null 2>&1 || true
	@sleep 1
	@port=8081; \
	for store in $(STORES); do \
		echo "Mapping $$store -> port $$port"; \
		nohup kubectl port-forward svc/retail-edge -n $$store $$port:8080 >/dev/null 2>&1 & \
		port=$$((port + 1)); \
	done; \
	nohup kubectl port-forward svc/grafana 3000:3000 -n monitoring >/dev/null 2>&1 & \
	echo "Tunnels initiated in background."

open-stores:
	@echo "SRE Operations: Launching all interfaces..."
	@echo "Opening Grafana at http://localhost:3000"
	@(xdg-open http://localhost:3000 || open http://localhost:3000) > /dev/null 2>&1
	@sleep 1
	@port=8081; \
	for store in $(STORES); do \
		url="http://localhost:$$port"; \
		echo "Opening $$store at $$url"; \
		(xdg-open $$url || open $$url) > /dev/null 2>&1; \
		port=$$((port + 1)); \
		sleep 0.5; \
	done

clean:
	@echo "SRE Cleanup: Purging environment..."
	@-pkill -f "port-forward" > /dev/null 2>&1 || true
	@k3d cluster delete $(CLUSTER_NAME)