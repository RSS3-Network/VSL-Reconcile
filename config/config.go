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

type Config struct {
	DiscoverySTS string
	DiscoveryNS  string

	CheckInterval time.Duration
	MaxBlockTime  time.Duration
}

func Setup() (*Config, error) {
	// Read sequencers list from environment variable (comma-separated)
	discoverySTS := os.Getenv(EnvDiscoverySTS)
	if discoverySTS == "" {
		return nil, fmt.Errorf("statefulset name is not provided")
	}

	discoveryNS := os.Getenv(EnvDiscoveryNS)
	if discoveryNS == "" {
		discoveryNS = "default"
	}

	// Parse check interval
	checkIntervalStr := os.Getenv(EnvCheckInterval)
	if checkIntervalStr == "" {
		checkIntervalStr = DefaultCheckInterval
	}

	checkInterval, err := time.ParseDuration(checkIntervalStr)
	if err != nil {
		return nil, fmt.Errorf("failed to parse check interval str (%s): %w", checkIntervalStr, err)
	}

	// Parse max block time (how long can we tolerate if the block number doesn't increase)
	maxBlockTimeStr := os.Getenv(EnvMaxBlockTime)
	if maxBlockTimeStr == "" {
		maxBlockTimeStr = DefaultMaxBlockTime
	}

	maxBlockTime, err := time.ParseDuration(maxBlockTimeStr)
	if err != nil {
		return nil, fmt.Errorf("failed to parse max block time str (%s): %w", maxBlockTimeStr, err)
	}

	return &Config{
		DiscoverySTS:  discoverySTS,
		DiscoveryNS:   discoveryNS,
		CheckInterval: checkInterval,
		MaxBlockTime:  maxBlockTime,
	}, nil
}
