#!/bin/bash

# SRE Automation: Refresh GCP Registry Token for k3s
echo "Step 1: Fetching fresh OAuth token from gcloud..."
TOKEN=$(gcloud auth print-access-token)

if [ -z "$TOKEN" ]; then
    echo "Error: Could not get gcloud token. Are you logged in?"
    exit 1
fi

echo "Step 2: Updating Kubernetes secret in Vagrant VM..."
vagrant ssh -c "sudo kubectl create secret docker-registry google-registry-key \
  --docker-server=europe-west3-docker.pkg.dev \
  --docker-username=oauth2accesstoken \
  --docker-password=$TOKEN \
  --dry-run=client -o yaml | sudo kubectl apply -f -"

echo "Step 3: Restarting Retail Edge pods to trigger fresh pull..."
vagrant ssh -c "sudo kubectl rollout restart deployment/retail-edge-app"

echo "Step 4: Watching pod status..."
vagrant ssh -c "sudo kubectl get pods -w"