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

// activateSequencerWithFirstID: Try to activate one of all sequencers from a specified ID.
// All sequencers are equal, but some sequencers are more equal than others.
func activateSequencerWithFirstID(firstID int, unsafeHash string, sequencersList []string) int {
	sequencersListLen := len(sequencersList)
	unsafeHashEnsure := unsafeHash
	for i := 0; i < sequencersListLen; i++ {
		// Calculate absolute ID
		id := i + firstID
		if id >= sequencersListLen {
			id -= sequencersListLen
		}

		var err error

		// Check if under any circumstance the unsafe hash from previously deactivated sequencer could be empty (e.g. it's offline)
		if unsafeHashEnsure == "" {
			// Try to get a valid unsafe hash
			unsafeHashEnsure, _, err = getUnsafeL2Status(sequencersList[id])
			if err != nil {
				log.Printf("failed to get unsafe hash from sequencer (%d): %v", id, err)
				unsafeHashEnsure = "" // Ensure this is cleared
				continue              // Proceed to next sequencer
			}
		}

		err = activateSequencer(sequencersList[id], unsafeHash)
		if err != nil {
			log.Printf("failed to activate sequencer (%d): %v", id, err)
			_, _ = deactivateSequencer(sequencersList[id]) // Ensure this sequencer is deactivated even it failed to activate
		} else {
			return id // That's it, our new king
		}

	}

	return -1 // Everyone has tried, and they all failed
}

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
		checkIntervalStr = "1s" // Default set as +1s
	}

	checkInterval, err := time.ParseDuration(checkIntervalStr)
	if err != nil {
		log.Fatalf("failed to parse check interval str (%s): %v", checkIntervalStr, err)
	}

	// Parse max block time (how long can we tolerate if the block number doesn't increase)
	maxBlockTimeStr := os.Getenv("MAX_BLOCK_TIME")
	if maxBlockTimeStr == "" {
		maxBlockTimeStr = "30s" // Default set as 30s
	}

	maxBlockTime, err := time.ParseDuration(maxBlockTimeStr)
	if err != nil {
		log.Fatalf("failed to parse max block time str (%s): %v", maxBlockTimeStr, err)
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

	currentBlockTime := time.Now()
	currentBlockHeight := int64(0)

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
			// Let's check the block height
			_, blockHeight, err := getUnsafeL2Status(sequencersList[primarySequencerID])
			if err != nil {
				log.Printf("failed to get unsafe L2 status from primary sequencer: %v", err)
			}

			if blockHeight > currentBlockHeight {
				// Say hi to our new block
				currentBlockTime = time.Now()
				currentBlockHeight = blockHeight
				continue
			} else {
				// equal or even less than, check max block delay
				if time.Now().Sub(currentBlockTime) <= maxBlockTime {
					// Within acceptable limit, proceed
					continue
				} // else we can't tolerate this, gear up and let's restart the sequencer!
			}
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
