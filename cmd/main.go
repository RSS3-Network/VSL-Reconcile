/*
 * Sequencers reconcile for Optimism sequencers
 *
 * For detailed API specifications
 * please refer to https://docs.optimism.io/builders/node-operators/json-rpc#admin
 *
 * For primary / secondary sequencer switch flow
 * please refer to https://www.notion.so/rss3/RSS3-VSL-sequencer-fb202ab61fc04ca7baf70d9bae408b1f
 */

package main

import (
	"context"

	"github.com/rss3-network/vsl-reconcile/config"
	"github.com/rss3-network/vsl-reconcile/internal/safe"
	"github.com/rss3-network/vsl-reconcile/pkg/server"
	"github.com/rss3-network/vsl-reconcile/pkg/service/aggregator"
	"github.com/rss3-network/vsl-reconcile/pkg/service/heartbeat"
	"github.com/rss3-network/vsl-reconcile/pkg/service/http"
	"github.com/rss3-network/vsl-reconcile/pkg/service/label"
	"github.com/spf13/cobra"
	"go.uber.org/zap"
)

var (
	debug bool
)

var rootCmd = &cobra.Command{
	Use: "reconcile",
	RunE: func(_ *cobra.Command, _ []string) error {
		cfg, err := config.Setup()
		if err != nil {
			return err
		}

		providerAggregator := aggregator.New(
			cfg,
			&http.Service{},
			&label.Service{},
			&heartbeat.Service{},
		)

		routinesPool := safe.NewPool(context.Background())
		server := server.NewServer(providerAggregator, routinesPool)
		server.Start()

		server.Wait()

		return nil
	},
}

func init() {
	rootCmd.PersistentFlags().BoolVar(&debug, "debug", false, "enable debug mode")

	if debug {
		zap.ReplaceGlobals(zap.Must(zap.NewDevelopment()))
	} else {
		zap.ReplaceGlobals(zap.Must(zap.NewProduction()))
	}
}

func main() {
	if err := rootCmd.Execute(); err != nil {
		zap.L().Fatal("exec error:", zap.Error(err))
	}
}
