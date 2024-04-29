package config

import (
	"fmt"
	"os"
	"time"
)

const (
	DefaultCheckInterval = "60s"
	DefaultMaxBlockTime  = "5m"

	EnvDiscoverySTS  = "DISCOVERY_STS"
	EnvDiscoveryNS   = "DISCOVERY_NS"
	EnvCheckInterval = "CHECK_INTERVAL"
	EnvMaxBlockTime  = "MAX_BLOCK_TIME"
)

func InitializeConfigurations() ([]string, time.Duration, time.Duration, error) {
	// Read sequencers list from environment variable (comma-separated)
	discoverySTS := os.Getenv(EnvDiscoverySTS)
	if discoverySTS == "" {
		return nil, 0, 0, fmt.Errorf("statefulset name is not provided")
	}
	discoveryNS := os.Getenv(EnvDiscoveryNS)
	if discoveryNS == "" {
		discoveryNS = "default"
	}
	clientset, err := initKubeClient()
	if err != nil {
		return nil, 0, 0, fmt.Errorf("failed to initialize kubernetes client: %w", err)
	}
	sequencersList, err := DiscoverStsEndpoints(clientset, discoverySTS, discoveryNS)
	if err != nil {
		return nil, 0, 0, fmt.Errorf("failed to discover sequencers: %w", err)
	}

	// Parse check interval
	checkIntervalStr := os.Getenv(EnvCheckInterval)
	if checkIntervalStr == "" {
		checkIntervalStr = DefaultCheckInterval
	}

	checkInterval, err := time.ParseDuration(checkIntervalStr)
	if err != nil {
		return nil, 0, 0, fmt.Errorf("failed to parse check interval str (%s): %w", checkIntervalStr, err)
	}

	// Parse max block time (how long can we tolerate if the block number doesn't increase)
	maxBlockTimeStr := os.Getenv(EnvMaxBlockTime)
	if maxBlockTimeStr == "" {
		maxBlockTimeStr = DefaultMaxBlockTime
	}

	maxBlockTime, err := time.ParseDuration(maxBlockTimeStr)
	if err != nil {
		return nil, 0, 0, fmt.Errorf("failed to parse max block time str (%s): %w", maxBlockTimeStr, err)
	}

	return sequencersList, checkInterval, maxBlockTime, nil
}
