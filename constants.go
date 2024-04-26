package main

import "time"

const (
	DefaultCheckInterval = "60s"
	DefaultMaxBlockTime  = "5m"
)

const (
	// MaxMainnetBlockTimestampLateTolerance : Mainnet is 12 seconds per block, and an active sequencer's sync status are allowed to left behind 3 blocks maximum
	MaxMainnetBlockTimestampLateTolerance = 3 * 12

	// JSONRPCCallRequestTimeout : JSON-RPC Calls timeout
	JSONRPCCallRequestTimeout = 1 * time.Minute

	// JSONRPCCallFailRetry : How many times we can retry when JSON-RPC Calls fails
	JSONRPCCallFailRetry = 3
)
