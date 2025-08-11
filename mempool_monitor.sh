#!/bin/bash

# Allow users to pass in RPC_URL, defaulting to http://localhost:26657
RPC_URL="${1:-http://localhost:26657}"

# Monitor mempool transaction count every 1 second
while true; do
  # Get current timestamp (optional, for logging)
  timestamp=$(date +"%Y-%m-%d %H:%M:%S")
  
  # Fetch unconfirmed transaction count using curl and parse with jq
  response=$(curl -s "$RPC_URL/num_unconfirmed_txs" | jq .result.n_txs)
  
  # Print the result with timestamp
  echo "[$timestamp] Mempool transaction count: $response"
  
  # Wait for 1 second before next check
  sleep 1
done