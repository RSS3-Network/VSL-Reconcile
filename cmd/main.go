/*
 * Sequencers watchdog for Optimism sequencers
 *
 * For detailed API specifications
 * please refer to https://docs.optimism.io/builders/node-operators/json-rpc#admin
 *
 * For primary / secondary sequencer switch flow
 * please refer to https://www.notion.so/rss3/RSS3-VSL-sequencer-fb202ab61fc04ca7baf70d9bae408b1f
 */

package main

import (
	"fmt"
	"log"
	"sequencer-watchdog/config"
	"sequencer-watchdog/internal/rpc"
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
		// Try to get a valid unsafe hash
		var (
			isSequencerReady   bool
			unsafeHashResponse string
		)
		unsafeHashResponse, _, isSequencerReady, err = rpc.GetOPSyncStatus(sequencersList[id])
		if err != nil {
			log.Printf("failed to get unsafe hash from sequencer (%d): %v", id, err)
			continue // Proceed to next sequencer
		}
		if !isSequencerReady || unsafeHashResponse == "" {
			// This sequencer is not ready to be activated
			log.Printf("sequencer (%d) is not ready to be activated: %v", id, err)
			continue
		}

		// Update possible missing unsafe hash parameter
		if unsafeHashEnsure == "" {
			unsafeHashEnsure = unsafeHashResponse
		}

		err = rpc.ActivateSequencer(sequencersList[id], unsafeHashEnsure)
		if err != nil {
			log.Printf("failed to activate sequencer (%d): %v", id, err)
			_, _ = rpc.DeactivateSequencer(sequencersList[id]) // Ensure this sequencer is deactivated even it failed to activate
		} else {
			return id // That's it, our new king
		}

	}

	return -1 // Everyone has tried, and they all failed
}

func Bootstrap(sequencersList []string) (int, error) {
	// Determine which sequencer is primary
	log.Printf("determine current primary sequencer")
	primarySequencerID := -1
	for id, sequencer := range sequencersList {
		isActive, err := rpc.CheckSequencerActive(sequencer)
		if err != nil {
			// Failed to get sequencer status
			log.Printf("failed to get sequencer (#%d %s) status: %v", id, sequencer, err)
			continue
		}

		if isActive {
			// Will this be the primary sequencer?
			if primarySequencerID == -1 {
				// Set as primary sequencer
				log.Printf("found sequencer (#%d %s) as primary", id, sequencer)
				primarySequencerID = id
			} else {
				// Already have a primary sequencer, deactivate this to prevent conflict (poor optimism)
				log.Printf("another sequencer is already active, deactivating this sequencer (#%d %s)...", id, sequencer)
				_, err = rpc.DeactivateSequencer(sequencer) // ignore unsafe hash
				if err != nil {
					log.Printf("failed to deactivate another active sequencer (#%d %s): %v", id, sequencer, err)
				}
			}
		}
	}

	// If neither of these sequencers are primary, try to promote one
	if primarySequencerID == -1 {
		log.Printf("no primary sequencer found, start promote progress...")
		primarySequencerID = activateSequencerWithFirstID(0, "", sequencersList)
		if primarySequencerID == -1 {
			// All sequencer activate fail
			return -1, fmt.Errorf("failed to activate any sequencers")
		} else {
			log.Printf("sequencer (#%d %s) is now primary.", primarySequencerID, sequencersList[primarySequencerID])
		}
	}

	return primarySequencerID, nil
}

func HeartbeatLoop(sequencersList []string, primarySequencerID int, checkInterval time.Duration, maxBlockTime time.Duration) {
	currentBlockTime := time.Now()
	currentBlockHeight := int64(0)

	for {
		time.Sleep(checkInterval)

		// Check for sequencer status
		isActive, err := rpc.CheckSequencerActive(sequencersList[primarySequencerID])
		if err != nil {
			log.Printf("failed to check primary sequencer status: %v", err)
		} else if !isActive {
			log.Printf("primary sequencer is not active, switching...")
		} else {
			// Primary sequencer is active, let's check the block height
			log.Printf("start check current block height")
			_, blockHeight, _, err := rpc.GetOPSyncStatus(sequencersList[primarySequencerID])
			if err != nil {
				log.Printf("failed to get unsafe L2 status from primary sequencer (#%d %s): %v", primarySequencerID, sequencersList[primarySequencerID], err)
				// Then see this sequencer as working abnormally, proceed to restart it
			} else {
				// Regard this request as successful and blockHeight is real
				log.Printf("block height get, start compare")
				if blockHeight > currentBlockHeight {
					// Say hi to our new block
					log.Printf("new block height found, reset tolerate timer")
					currentBlockTime = time.Now()
					currentBlockHeight = blockHeight
					continue // nothing else to do, wait for next round as this sequencer should be fine
				} else {
					// equal or even less than, check max block delay
					if time.Now().Sub(currentBlockTime) <= maxBlockTime {
						// Within acceptable limit, proceed too
						log.Printf("still old blocks, but it's fine")
						continue
					} else {
						// we can't tolerate this, gear up and let's restart the sequencer!
						log.Printf("block time exceeds maximal tolerance, this sequencer might working abnormally, trying to restart it...")
					}
				}
			}

		}

		// If current sequencer is working abnormally, try to restart it before promote next sequencer as primary
		log.Printf("for some reason the current primary sequencer (#%d %s) is not working, we have to promote a new primary.", primarySequencerID, sequencersList[primarySequencerID])
		// 1. deactivate this sequencer
		log.Printf("first let's try to shutdown it")
		unsafeHash, err := rpc.DeactivateSequencer(sequencersList[primarySequencerID])
		if err != nil {
			log.Printf("failed to deactivate sequencer (%d): %v", primarySequencerID, err)
		}

		// 2. activate a new sequencer
		log.Printf("then let's find it's successor")
		primarySequencerID = activateSequencerWithFirstID(primarySequencerID, unsafeHash, sequencersList)
		if primarySequencerID == -1 {
			log.Fatalf("failed to activate any sequencer")
		} else {
			log.Printf("sequencer (#%d %s) is now primary.", primarySequencerID, sequencersList[primarySequencerID])
		}
	}
}

func main() {
	// Initialize
	log.Printf("start initialize")
	sequencersList, checkInterval, maxBlockTime, err := config.InitializeConfigurations()
	if err != nil {
		log.Fatalf("initialize failed: %v", err)
	}

	log.Printf("start with %d sequencers, heartbeat interval %ds, timeout %ds", len(sequencersList), checkInterval/time.Second, maxBlockTime/time.Second)
	for id, sequencer := range sequencersList {
		log.Printf("sequencer %d: %s", id, sequencer)
	}

	// Bootstrap
	log.Printf("start bootstrap")
	primarySequencerID, err := Bootstrap(sequencersList)
	if err != nil {
		log.Fatalf("bootstrap failed: %v", err)
	}

	// Start heartbeat loop
	log.Printf("start heartbeat loop")
	HeartbeatLoop(sequencersList, primarySequencerID, checkInterval, maxBlockTime)
}
