package run

import (
	"context"
	"fmt"
	"log"
	"time"

	"github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/ethclient"

	limiterpkg "github.com/0glabs/evmchainbench/lib/limiter"
)

type Transmitter struct {
	RpcUrl  string
	limiter *limiterpkg.RateLimiter
}

func NewTransmitter(rpcUrl string, limiter *limiterpkg.RateLimiter) (*Transmitter, error) {
	return &Transmitter{
		RpcUrl:  rpcUrl,
		limiter: limiter,
	}, nil
}

func (t *Transmitter) Broadcast(txsMap map[int]types.Transactions) error {
	ch := make(chan error)

	for _, txs := range txsMap {
		go func(txs []*types.Transaction) {
			client, err := ethclient.Dial(t.RpcUrl)
			if err != nil {
				// Retry connection up to 4 times
				for retry := 0; retry < 4; retry++ {
					log.Printf("Failed to connect to RPC, retrying %d/4: %v", retry+1, err)
					time.Sleep(time.Duration(retry+1) * 100 * time.Millisecond)
					client, err = ethclient.Dial(t.RpcUrl)
					if err == nil {
						break
					}
				}
				if err != nil {
					log.Printf("Failed to connect to RPC after 4 retries, skipping batch: %v", err)
					ch <- nil // Don't exit, just skip this batch
					return
				}
			}

			for _, tx := range txs {
				for {
					if t.limiter == nil || t.limiter.AllowRequest() {
						err := broadcastWithRetry(client, tx)
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