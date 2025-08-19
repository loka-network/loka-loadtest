package run

import (
	"context"
	"fmt"
	"log"
	"sync"
	"sync/atomic"
	"time"

	"github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/ethclient"

	limiterpkg "github.com/0glabs/evmchainbench/lib/limiter"
)

type Transmitter struct {
	RpcUrl  string
	limiter *limiterpkg.RateLimiter

	pool      []*ethclient.Client
	poolOnce  sync.Once
	poolErr   error
	poolIndex uint64
	poolSize  int
}

func NewTransmitter(rpcUrl string, limiter *limiterpkg.RateLimiter, poolSize int) (*Transmitter, error) {
	return &Transmitter{
		RpcUrl:   rpcUrl,
		limiter:  limiter,
		poolSize: poolSize,
	}, nil
}

func (t *Transmitter) getClientFromPool() (*ethclient.Client, error) {
	t.poolOnce.Do(func() {
		ps := t.poolSize
		if ps <= 0 {
			ps = 800
		}
		pool := make([]*ethclient.Client, 0, ps)
		for i := 0; i < ps; i++ {
			var cli *ethclient.Client
			var err error
			for retry := 0; retry < 4; retry++ {
				cli, err = ethclient.Dial(t.RpcUrl)
				if err == nil {
					break
				}
				log.Printf("[pool] Failed to connect (slot %d), retrying %d/4: %v", i, retry+1, err)
				time.Sleep(time.Duration(retry+1) * 100 * time.Millisecond)
			}
			if err != nil {
				t.poolErr = fmt.Errorf("failed to initialize client pool at slot %d: %w", i, err)
				return
			}
			pool = append(pool, cli)
		}
		t.pool = pool
	})
	if t.poolErr != nil {
		return nil, t.poolErr
	}
	if len(t.pool) == 0 {
		return nil, fmt.Errorf("client pool is empty")
	}
	idx := atomic.AddUint64(&t.poolIndex, 1)
	return t.pool[idx%uint64(len(t.pool))], nil
}

func (t *Transmitter) Broadcast(txsMap map[int]types.Transactions) error {
	ch := make(chan error)

	// Ensure pool initialized early to catch any fatal errors
	if _, err := t.getClientFromPool(); err != nil {
		return fmt.Errorf("failed to initialize RPC client pool: %w", err)
	}

	for _, txs := range txsMap {
		go func(txs []*types.Transaction) {
			for _, tx := range txs {
				for {
					if t.limiter == nil || t.limiter.AllowRequest() {
						client, err := t.getClientFromPool()
						if err != nil {
							log.Printf("Client pool error: %v", err)
							time.Sleep(10 * time.Millisecond)
							continue
						}
						err = broadcastWithRetry(client, tx)
						if err != nil {
							log.Printf("Failed to broadcast transaction %s after retries: %v", tx.Hash().Hex(), err)
							// Continue with next transaction instead of exiting
							break
						}
						break
					} else {
						time.Sleep(10 * time.Millisecond)
					}
				}
			}
			ch <- nil
		}(txs)
	}

	senderCount := len(txsMap)
	for i := 0; i < senderCount; i++ {
		err := <-ch
		if err != nil {
			log.Printf("Batch processing error: %v", err)
			// Continue processing other batches instead of returning immediately
		}
	}

	return nil
}

func broadcastWithRetry(client *ethclient.Client, tx *types.Transaction) error {
	const maxRetries = 4

	for retry := 0; retry < maxRetries; retry++ {
		err := broadcast(client, tx)
		if err == nil {
			return nil
		}

		// Log retry attempt
		if retry < maxRetries-1 {
			log.Printf("Broadcast failed for tx %s, retrying %d/%d: %v",
				tx.Hash().Hex(), retry+1, maxRetries, err)
			// Shorter backoff for stress testing: 100ms, 200ms, 400ms, 800ms
			backoff := time.Duration(1<<retry) * 100 * time.Millisecond
			time.Sleep(backoff)
		}
	}

	// All retries failed
	return fmt.Errorf("failed to broadcast transaction %s after %d retries", tx.Hash().Hex(), maxRetries)
}

func broadcast(client *ethclient.Client, tx *types.Transaction) error {
	err := client.SendTransaction(context.Background(), tx)
	if err != nil {
		return err
	}

	// Check tx hash
	// the hash can be obtained: tx.Hash().Hex()
	return nil
}