package heartbeat

import (
	"context"
	"fmt"
	"time"

	"github.com/rss3-network/vsl-reconcile/config"
	"github.com/rss3-network/vsl-reconcile/internal/rpc"
	"github.com/rss3-network/vsl-reconcile/internal/safe"
	"github.com/rss3-network/vsl-reconcile/pkg/kube"
	"github.com/rss3-network/vsl-reconcile/pkg/service"
	"go.uber.org/zap"
)

var _ service.Service = (*Service)(nil)

type Service struct {
	sequencerList []string
	checkInterval time.Duration
	maxBlockTime  time.Duration
}

func (s *Service) Run(pool *safe.Pool) error {
	log := zap.L().With(zap.String("service", "heartbeat"))

	for id, sequencer := range s.sequencerList {
		log.Debug("sequencer found", zap.Int("id", id), zap.String("sequencer", sequencer))
	}

	// Bootstrap
	log.Debug("start bootstrap")

	primarySequencerID, err := Bootstrap(s.sequencerList)
	if err != nil {
		log.Error("failed to bootstrap", zap.Error(err))
	}

	// Start heartbeat loop
	log.Info("start heartbeat loop", zap.Int("primary_sequencer_id", primarySequencerID))

	pool.GoCtx(func(_ context.Context) {
		s.Loop(primarySequencerID)
	})

	return nil
}

func (s *Service) Init(cfg *config.Config) error {
	clientset, err := kube.Client()
	if err != nil {
		return fmt.Errorf("failed to initialize kubernetes client: %w", err)
	}

	sequencerList, err := DiscoverStsEndpoints(clientset, cfg.DiscoverySTS, cfg.DiscoveryNS)
	if err != nil {
		return fmt.Errorf("failed to discover sequencers: %w", err)
	}

	s.sequencerList = sequencerList
	s.checkInterval = cfg.CheckInterval
	s.maxBlockTime = cfg.MaxBlockTime

	return nil
}

func (s *Service) String() string {
	return "heartbeat"
}

// activateSequencerWithFirstID: Try to activate one of all sequencers from a specified ID.
// All sequencers are equal, but some sequencers are more equal than others.
func activateSequencerWithFirstID(firstID int, unsafeHash string, sequencersList []string) int {
	log := zap.L().With(zap.String("service", "heartbeat"))

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
			log.Error("failed to get unsafe hash from sequencer", zap.String("sequencer", sequencersList[id]), zap.Error(err))
			continue // Proceed to next sequencer
		}

		if !isSequencerReady || unsafeHashResponse == "" {
			// This sequencer is not ready to be activated
			log.Error("sequencer is not ready to be activated",
				zap.String("sequencer", sequencersList[id]))
			continue
		}

		// Update possible missing unsafe hash parameter
		if unsafeHashEnsure == "" {
			unsafeHashEnsure = unsafeHashResponse
		}

		err = rpc.ActivateSequencer(sequencersList[id], unsafeHashEnsure)
		if err != nil {
			log.Error("failed to activate sequencer",
				zap.String("sequencer", sequencersList[id]),
				zap.Error(err),
			)

			_, _ = rpc.DeactivateSequencer(sequencersList[id]) // Ensure this sequencer is deactivated even it failed to activate
		} else {
			return id // That's it, our new king
		}
	}

	return -1 // Everyone has tried, and they all failed
}

func Bootstrap(sequencersList []string) (int, error) {
	log := zap.L().With(zap.String("service", "heartbeat"))

	// Determine which sequencer is primary
	log.Debug("determine current primary sequencer")

	primarySequencerID := -1

	for id, sequencer := range sequencersList {
		isActive, err := rpc.CheckSequencerActive(sequencer)
		if err != nil {
			// Failed to get sequencer status
			log.Error("failed to get sequencer status",
				zap.Int("id", id),
				zap.String("sequencer", sequencer),
				zap.Error(err),
			)

			continue
		}

		if isActive {
			// Will this be the primary sequencer?
			if primarySequencerID == -1 {
				// Set as primary sequencer
				log.Info("found primary sequencer", zap.Int("id", id), zap.String("sequencer", sequencer))
				primarySequencerID = id
			} else {
				// Already have a primary sequencer, deactivate this to prevent conflict (poor optimism)
				log.Warn("another sequencer is already active, deactivating this sequencer", zap.Int("id", id), zap.String("sequencer", sequencer))
				// ignore unsafe hash
				if _, err = rpc.DeactivateSequencer(sequencer); err != nil {
					log.Error("failed to deactivate another active sequencer",
						zap.Int("id", id),
						zap.String("sequencer", sequencer),
						zap.Error(err),
					)
				}
			}
		}
	}

	// If neither of these sequencers are primary, try to promote one
	if primarySequencerID == -1 {
		log.Info("no primary sequencer found, start promote progress...")

		primarySequencerID = activateSequencerWithFirstID(0, "", sequencersList)
		if primarySequencerID == -1 {
			// All sequencer activate fail
			return -1, fmt.Errorf("failed to activate any sequencers")
		}

		log.Info("sequencer is now primary.", zap.Int("id", primarySequencerID), zap.String("sequencer", sequencersList[primarySequencerID]))
	}

	return primarySequencerID, nil
}

func (s *Service) Loop(primarySequencerID int) {
	log := zap.L().With(zap.String("service", "heartbeat"))

	currentBlockTime := time.Now()
	currentBlockHeight := int64(0)

	for {
		time.Sleep(s.checkInterval)

		// Check for sequencer status
		isActive, err := rpc.CheckSequencerActive(s.sequencerList[primarySequencerID])

		switch {
		case err != nil:
			log.Error("failed to check primary sequencer status", zap.Error(err))
		case !isActive:
			log.Info("primary sequencer is not active, switching...")
		default:
			// Primary sequencer is active, let's check the block height
			log.Debug("start check current block height")

			_, blockHeight, _, err := rpc.GetOPSyncStatus(s.sequencerList[primarySequencerID])
			if err != nil {
				// Then see this sequencer as working abnormally, proceed to restart it
				log.Error("failed to get unsafe L2 status from primary sequencer", zap.Int("id", primarySequencerID), zap.String("sequencer", s.sequencerList[primarySequencerID]), zap.Error(err))
			} else {
				// Regard this request as successful and blockHeight is real
				log.Info("block height get, start compare", zap.Int64("block_height", blockHeight), zap.Int64("current_block_height", currentBlockHeight), zap.Duration("block_time", time.Since(currentBlockTime)), zap.Duration("max_block_time", s.maxBlockTime))

				if blockHeight > currentBlockHeight {
					// Say hi to our new block
					log.Info("new block height found, reset tolerate timer", zap.Int64("block_height", blockHeight), zap.Int64("current_block_height", currentBlockHeight), zap.Duration("block_time", time.Since(currentBlockTime)), zap.Duration("max_block_time", s.maxBlockTime))

					currentBlockTime = time.Now()
					currentBlockHeight = blockHeight

					continue // nothing else to do, wait for next round as this sequencer should be fine
				}

				// equal or even less than, check max block delay
				if time.Since(currentBlockTime) <= s.maxBlockTime {
					// Within acceptable limit, proceed too
					log.Warn("still old blocks, but it's fine", zap.Int64("block_height", blockHeight), zap.Int64("current_block_height", currentBlockHeight), zap.Duration("block_time", time.Since(currentBlockTime)), zap.Duration("max_block_time", s.maxBlockTime))
					continue
				}

				// we can't tolerate this, gear up and let's restart the sequencer!
				log.Warn("block time exceeds maximal tolerance, this sequencer might working abnormally, trying to restart it...")
			}
		}

		// If current sequencer is working abnormally, try to restart it before promote next sequencer as primary
		log.Info("for some reason the current primary sequencer (#%d %s) is not working, we have to promote a new primary.",
			zap.Int("id", primarySequencerID),
			zap.String("sequencer", s.sequencerList[primarySequencerID]),
		)
		// 1. deactivate this sequencer
		log.Info("first let's try to shutdown it")

		unsafeHash, err := rpc.DeactivateSequencer(s.sequencerList[primarySequencerID])
		if err != nil {
			log.Error("failed to deactivate sequencer (%d): %v",
				zap.Int("id", primarySequencerID), zap.Error(err))
		}

		// 2. activate a new sequencer
		log.Info("then let's find it's successor")

		primarySequencerID = activateSequencerWithFirstID(primarySequencerID, unsafeHash, s.sequencerList)
		if primarySequencerID == -1 {
			log.Fatal("failed to activate any sequencer")
		}

		log.Info("sequencer is now primary.", zap.Int("id", primarySequencerID), zap.String("sequencer", s.sequencerList[primarySequencerID]))
	}
}
