/*
 * Sequencers watchdog for Optimism nodes
 *
 * For detailed API specifications
 * please refer to https://docs.optimism.io/builders/node-operators/json-rpc#admin
 *
 * For primary / secondary sequencer switch flow
 * please refer to https://www.notion.so/rss3/RSS3-VSL-sequencer-fb202ab61fc04ca7baf70d9bae408b1f
 */

package main

import (
	"log"
	"os"
	"strings"
	"time"
)

func main() {
	// Read sequencers list from environment variable (comma-separated)
	sequencersListStr := os.Getenv("SEQUENCERS_LIST")
	if sequencersListStr == "" {
		// No sequencers specified, panic
		log.Fatalf("no sequencers specified")
	}

	sequencersList := strings.Split(sequencersListStr, ",")

	// Parse check interval
	checkIntervalStr := os.Getenv("CHECK_INTERVAL")
	if checkIntervalStr == "" {
		checkIntervalStr = "1s" // Default set as 1 seconds
	}

	checkInterval, err := time.ParseDuration(checkIntervalStr)
	if err != nil {
		log.Fatalf("failed to parse check interval str (%s): %v", checkIntervalStr, err)
	}

	// Determine which sequencer is primary
	primarySequencerID := -1
	for id, sequencer := range sequencersList {
		isActive, err := checkSequencerActive(sequencer)
		if err != nil {
			// Failed to get sequencer status
			log.Printf("failed to get sequencer status: %v", err)
			continue
		}

		if isActive {
			// Will this be the primary sequencer?
			if primarySequencerID == -1 {
				// Set as primary sequencer
				primarySequencerID = id
			} else {
				// Already have a primary sequencer, deactivate this to prevent conflict (poor optimism)
				_, err := deactivateSequencer(sequencer) // ignore unsafe hash
				if err != nil {
					log.Printf("failed to deactivate another active sequencer: %v", err)
				}
			}
		}
	}

	// If neither of these sequencers are primary, try to promote one
	if primarySequencerID == -1 {
		primarySequencerID = activateSequencerWithFirstID(0, "", sequencersList)
		if primarySequencerID == -1 {
			// All sequencer activate fail
			log.Fatalf("failed to activate any sequencer")
		}
	}

	// Start routine
	t := time.NewTicker(checkInterval)
	for {
		// Wait for ticker
		<-t.C

		// Check for sequencer status
		isActive, err := checkSequencerActive(sequencersList[primarySequencerID])
		if err != nil {
			log.Printf("failed to check primary sequencer status: %v", err)
		} else if !isActive {
			log.Printf("primary sequencer is not active, switching...")
		} else {
			// It's fine
			continue
		}

		// If current sequencer is working abnormally, promote next sequencer as primary
		// 1. deactivate this sequencer
		unsafeHash, err := deactivateSequencer(sequencersList[primarySequencerID])
		if err != nil {
			log.Printf("failed to deactivate sequencer (%d): %v", primarySequencerID, err)
		}

		// 2. activate a new sequencer
		primarySequencerID = activateSequencerWithFirstID(primarySequencerID+1, unsafeHash, sequencersList)
		if primarySequencerID == -1 {
			log.Fatalf("failed to activate any sequencer")
		}

	}

}
