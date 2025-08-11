#!/bin/bash

# make first to ensure all dependencies are built
make

# This script requires the 'cast' tool from Foundry.
# Install with: curl -L https://foundry.paradigm.xyz | bash && foundryup

# Usage: ./test-multi-acc-from-root.sh <root_private_key> <num_accounts> [sender_count] <amount_per_account_eth> [rpc_url] [ws_url] [type]
ROOT_PRIVKEY="$1"
NUM_ACCS="$2"
SENDER_COUNT="${3:-4}" # default 4
AMOUNT="$4"
RPC_URL="$5"
WS_URL="$6"
TYPE="${7:-simple}" # default to simple if not provided

if [ -z "$ROOT_PRIVKEY" ] || [ -z "$NUM_ACCS" ] || [ -z "$AMOUNT" ]; then
  echo "Usage: $0 <root_private_key> <num_accounts> [sender_count] <amount_per_account> [rpc_url] [ws_url] [type]"
  exit 1
fi

# Use default RPC URL if not provided
if [ -z "$RPC_URL" ]; then
  RPC_URL="http://127.0.0.1:8545"
fi

# Use default WS URL if not provided
if [ -z "$WS_URL" ]; then
  WS_URL="ws://127.0.0.1:8546"
fi

# Generate random private keys and addresses
ACCS=()
for ((i=0;i<NUM_ACCS;i++)); do
  PRIV=$(openssl rand -hex 32)
  ADDR=$(cast wallet address $PRIV) # Requires foundry/cast tool
  ACCS+=("$PRIV:$ADDR")
done

# Transfer from root private key to each account
for entry in "${ACCS[@]}"; do
  PRIV=$(echo $entry | cut -d: -f1)
  ADDR=$(echo $entry | cut -d: -f2)
  # Use cast send to transfer (or replace with your own transfer command)
  cast send --private-key $ROOT_PRIVKEY --rpc-url $RPC_URL $ADDR --value "${AMOUNT}ether"
done

# Start evmchainbench in parallel
for entry in "${ACCS[@]}"; do
  PRIV=$(echo $entry | cut -d: -f1)
  if [[ "$TYPE" == "erc20" || "$TYPE" == "uniswap" ]]; then
    ./bin/evmchainbench run --faucet-private-key $PRIV --tx-count 20000 --mempool $((100000/NUM_ACCS)) -p "$TYPE" --http-rpc $RPC_URL --ws-rpc $WS_URL --sender-count $SENDER_COUNT &
  else
    ./bin/evmchainbench run --faucet-private-key $PRIV --tx-count 20000 --mempool $((100000/NUM_ACCS)) --http-rpc $RPC_URL --ws-rpc $WS_URL --sender-count $SENDER_COUNT &
  fi
done

wait
