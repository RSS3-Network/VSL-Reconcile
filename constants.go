package main

const (
	// MaxMainnetBlockTimestampLateTolerance : Mainnet is 12 seconds per block, and an active sequencer's sync status are allowed to left behind 3 blocks maximum
	MaxMainnetBlockTimestampLateTolerance = 3 * 12
)
