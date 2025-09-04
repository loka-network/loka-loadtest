package run

import (
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"strconv"

	limiterpkg "github.com/0glabs/evmchainbench/lib/limiter"
	"github.com/gorilla/websocket"
)

type BlockInfo struct {
	Time     int64
	TxCount  int64
	GasUsed  int64
	GasLimit int64
}

type EthereumListener struct {
	wsURL            string
	conn             *websocket.Conn
	limiter          *limiterpkg.RateLimiter
	blockStat        []BlockInfo
	quit             chan struct{}
	bestTPS          int64
	gasUsedAtBestTPS float64
}

func NewEthereumListener(wsURL string, limiter *limiterpkg.RateLimiter) *EthereumListener {
	return &EthereumListener{
		wsURL:   wsURL,
		limiter: limiter,
		quit:    make(chan struct{}),
	}
}

func (el *EthereumListener) Connect() error {
	conn, _, err := websocket.DefaultDialer.Dial(el.wsURL, http.Header{})
	if err != nil {
		return fmt.Errorf("dial error: %v", err)
	}
	el.conn = conn
	return nil
}

func (el *EthereumListener) SubscribeNewHeads() error {
	subscribeMsg := map[string]interface{}{
		"jsonrpc": "2.0",
		"id":      1,
		"method":  "eth_subscribe",
		"params":  []interface{}{"newHeads"},
	}
	err := el.conn.WriteJSON(subscribeMsg)
	if err != nil {
		return fmt.Errorf("subscribe error: %v", err)
	}

	go el.listenForMessages()

	return nil
}

func (el *EthereumListener) listenForMessages() {
	for {
		_, message, err := el.conn.ReadMessage()
		if err != nil {
			return
		}

		var response map[string]interface{}
		err = json.Unmarshal(message, &response)
		if err != nil {
			log.Println("unmarshal error:", err)
			continue
		}

		if method, ok := response["method"]; ok && method == "eth_subscription" {
			el.handleNewHead(response)
		} else {
			el.handleBlockResponse(response)
		}
	}
}

func (el *EthereumListener) handleNewHead(response map[string]interface{}) {
	params := response["params"].(map[string]interface{})
	result := params["result"].(map[string]interface{})
	blockNo := result["number"].(string)

	// Convert blockNo from hex string to decimal
	blockNoDec, err := strconv.ParseInt(blockNo[2:], 16, 64)
	if err != nil {
		log.Default().Println("Invalid block number:", blockNo)
	} else {
		log.Default().Println("Request block:", blockNoDec)
	}

	request := map[string]interface{}{
		"jsonrpc": "2.0",
		"id":      1,
		"method":  "eth_getBlockByNumber",
		"params":  []interface{}{blockNo, false},
	}
	err = el.conn.WriteJSON(request)
	if err != nil {
		log.Println("Failed to send block request:", err)
	}

	request = map[string]interface{}{
		"jsonrpc": "2.0",
		"id":      1,
		"method":  "eth_getLogs",
		"params": []interface{}{
			map[string]interface{}{
				"fromBlock": blockNo,
				"toBlock":   blockNo,
			},
		},
	}
	err = el.conn.WriteJSON(request)
	if err != nil {
		log.Println("Failed to send log request:", err)
	}
}

func (el *EthereumListener) handleBlockResponse(response map[string]interface{}) {
	if result, ok := response["result"].(map[string]interface{}); ok {
		if txns, ok := result["transactions"].([]interface{}); ok {
			el.limiter.IncreaseLimit(len(txns))
			ts, _ := strconv.ParseInt(result["timestamp"].(string)[2:], 16, 64)
			gasUsed, _ := strconv.ParseInt(result["gasUsed"].(string)[2:], 16, 64)
			gasLimit, _ := strconv.ParseInt(result["gasLimit"].(string)[2:], 16, 64)
			log.Default().Println("TxCount:", len(txns), "GasUsed:", gasUsed, "GasLimit:", gasLimit)
			el.blockStat = append(el.blockStat, BlockInfo{
				Time:     ts,
				TxCount:  int64(len(txns)),
				GasUsed:  gasUsed,
				GasLimit: gasLimit,
			})
			// keep only the last 60 seconds of blocks
			for {
				if len(el.blockStat) == 1 {
					break
				}
				if el.blockStat[len(el.blockStat)-1].Time-el.blockStat[0].Time > 60 {
					el.blockStat = el.blockStat[1:]
				} else {
					break
				}
			}
			// compute TPS over an active window that excludes early underfilled and trailing empty blocks
			if len(el.blockStat) >= 2 {
				startIdx := 0
				endIdx := len(el.blockStat) - 1

				// trim leading zero-tx blocks
				for startIdx < len(el.blockStat) && el.blockStat[startIdx].TxCount == 0 {
					startIdx++
				}
				// trim trailing zero-tx blocks
				for endIdx >= startIdx && el.blockStat[endIdx].TxCount == 0 {
					endIdx--
				}

				if endIdx > startIdx {
					// determine peak tx count in the active range
					peakTx := int64(0)
					for i := startIdx; i <= endIdx; i++ {
						if el.blockStat[i].TxCount > peakTx {
							peakTx = el.blockStat[i].TxCount
						}
					}
					// trim early underfilled blocks (<50% of peak)
					minFilled := peakTx / 2
					for startIdx < endIdx && el.blockStat[startIdx].TxCount < minFilled {
						startIdx++
					}

					timeSpan := el.blockStat[endIdx].Time - el.blockStat[startIdx].Time
					// require a sufficiently long active window
					if timeSpan > 20 {
						totalTxCount := int64(0)
						totalGasLimit := int64(0)
						totalGasUsed := int64(0)
						for i := startIdx; i <= endIdx; i++ {
							totalTxCount += el.blockStat[i].TxCount
							totalGasLimit += el.blockStat[i].GasLimit
							totalGasUsed += el.blockStat[i].GasUsed
						}
						tps := totalTxCount / timeSpan
						log.Default().Println("TimeSpan:", timeSpan, "TotalTxCount:", totalTxCount)
						gasUsedPercent := float64(totalGasUsed) / float64(totalGasLimit)
						if tps > el.bestTPS {
							el.bestTPS = tps
							el.gasUsedAtBestTPS = gasUsedPercent
						}
						fmt.Printf("TPS: %d GasUsed%%: %.2f%%\n", tps, gasUsedPercent*100)
						if totalTxCount < 100 {
							// exit if total tx count is less than 100
							fmt.Printf("Best TPS: %d GasUsed%%: %.2f%%\n", el.bestTPS, el.gasUsedAtBestTPS*100)
							el.Close()
							return
						}

						// to avoid waiting after the transmission is complete, close if the last 3 blocks are empty
						if len(el.blockStat) >= 3 {
							emptyTail := true
							for i := 1; i <= 3; i++ {
								if el.blockStat[len(el.blockStat)-i].TxCount != 0 {
									emptyTail = false
									break
								}
							}
							if emptyTail {
								fmt.Printf("Best TPS: %d GasUsed%%: %.2f%%\n", el.bestTPS, el.gasUsedAtBestTPS*100)
								el.Close()
							}
						}
					}
				}
			}
		} else {
			if result, ok := response["result"].([]interface{}); ok {
				if len(result) > 0 {
					fmt.Println("Logs:", len(result))
				}
			}
		}
	}
}

func (el *EthereumListener) Close() {
	if el.conn != nil {
		el.conn.Close()
	}
	close(el.quit)
}
