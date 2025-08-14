# Loka Blockchain Load Testing Tool

This project is primarily designed as a load testing tool for the **Loka blockchain**, which is a fully EVM-compatible chain utilizing **NeoBFT** consensus and **Block-STM** parallel execution. The tool enables performance benchmarking and stress testing of Loka under various transaction workloads.

## Features

- Supports native token transfers, simple smart contract calls, and ERC20 token transfers.
- Compatible with Loka's NeoBFT + Block-STM architecture.
- Multi-process and multi-account workload generation.
- Includes scripts for mempool monitoring and parallel client execution.

## Loka Chain Benchmark

[Benchmark Document](benchmark.md) provides an overview of the benchmark tests conducted on the Loka blockchain, including test setups, results, and configurations used.
- Provides a summary table of test results for different transaction types.
- Includes detailed configuration settings for the testing environment.
- Describes the steps for conducting high throughput tests and monitoring TPS (Transactions Per Second).
- Highlights the expected results and performance metrics for the Loka chain.

## Build Instructions

To compile the project, ensure you have the necessary dependencies installed (e.g., Go, Make).
Then run:

```sh
git clone https://github.com/loka-network/loka-loadtest.git
cd loka-loadtest

make
```

This will build all necessary binaries, including `lokabenchcli` and supporting tools.

## Usage

### Basic Benchmark

Run a benchmark with default parameters:

```sh
./bin/lokabenchcli run --http-rpc <RPC_URL> --ws-rpc <WS_URL> --tx-count <COUNT> --sender-count <NUM_SENDERS>
```

- `--http-rpc`: HTTP RPC endpoint of your Loka node.
- `--ws-rpc`: WebSocket RPC endpoint.
- `--tx-count`: Number of transactions to send.
- `--sender-count`: Number of concurrent senders.

### Multi-Account Parallel Benchmark

Use the provided script to generate multiple accounts, fund them, and start parallel clients:

```sh
./test-multi-acc-from-root.sh <root_private_key> <num_accounts> [sender_count] <amount_per_account_eth> [rpc_url] [ws_url] [type]
```

- `<root_private_key>`: Private key of the funding account.
- `<num_accounts>`: Number of test accounts to create.
- `[sender_count]`: Number of concurrent senders per client (default: 4).
- `<amount_per_account_eth>`: Amount of ETH to fund each account.
- `[rpc_url]`: HTTP RPC endpoint (default: http://127.0.0.1:8545).
- `[ws_url]`: WebSocket RPC endpoint (default: ws://127.0.0.1:8546).
- `[type]`: Workload type (`simple`, `erc20`, or `uniswap`).

See [`test-multi-acc-from-root.sh`](test-multi-acc-from-root.sh) for details.

### Mempool Monitoring

Monitor the mempool transaction count in real time:

```sh
./mempool_monitor.sh http://localhost:26657
```

This script queries the Loka node's RPC endpoint (default: `http://localhost:26657`) every second and prints the current mempool transaction count.

See [`mempool_monitor.sh`](mempool_monitor.sh) for details.

## Scripts

- [test-multi-acc-from-root.sh](test-multi-acc-from-root.sh): Generates multiple accounts, funds them, and launches parallel benchmark clients.
- [mempool_monitor.sh](mempool_monitor.sh): Monitors the mempool transaction count via RPC.