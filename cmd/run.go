/*
Copyright © 2024 NAME HERE <EMAIL ADDRESS>
*/
package cmd

import (
	"github.com/0glabs/evmchainbench/cmd/option"
	"github.com/0glabs/evmchainbench/lib/cmd/run"
	"github.com/spf13/cobra"
)

// runCmd represents the run command
var runCmd = &cobra.Command{
	Use:   "run",
	Short: "To run the benchmark",
	Long:  "To run the benchmark",
	Run: func(cmd *cobra.Command, args []string) {
		httpRpc, _ := cmd.Flags().GetString("http-rpc")
		wsRpc, _ := cmd.Flags().GetString("ws-rpc")
		faucetPrivateKey, _ := cmd.Flags().GetString("faucet-private-key")
		senderCount, _ := cmd.Flags().GetInt("sender-count")
		txCount, _ := cmd.Flags().GetInt("tx-count")
		txType, _ := cmd.Flags().GetString("tx-type")
		mempool, _ := cmd.Flags().GetInt("mempool")
		poolSize, _ := cmd.Flags().GetInt("client-pool-size")

		run.Run(httpRpc, wsRpc, faucetPrivateKey, senderCount, txCount, txType, mempool, poolSize)
	},
}

func init() {
	rootCmd.AddCommand(runCmd)
	option.OptionsForGeneration(runCmd)
	runCmd.Flags().Int("client-pool-size", 800, "HTTP client pool size for broadcasting (default 800)")
}
