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

// activateSequencerByID: Try to activate one of all sequencers from a specified ID.
// All sequencers are equal, but some sequencers are "more equal" than others.
func activateSequencerByID(id int, unsafeHash string, sequencersList []string) int {
	log := zap.L().With(zap.String("service", "heartbeat"))

	for i := 0; i < len(sequencersList); i++ {
		// index is the absolute position of sequencer in the list
		index := (i + id) % len(sequencersList)

		// Activates sequencer and handles possible failures internally
		if activated, err := activateSequencer(sequencersList[index], unsafeHash); activated {
			return index // Return the ID of the activated sequencer
		} else if err != nil {
			log.Error("Failed to activate sequencer",
				zap.String("sequencer", sequencersList[index]),
				zap.Error(err),
			)
		}
	}
	// No sequencer could be activated
	return -1
}

// activateSequencer: Activate a sequencer and return whether it was successful
func activateSequencer(sequencer string, unsafeHash string) (bool, error) {
	unsafeHashResponse, _, isReady, err := rpc.GetOPSyncStatus(sequencer)
	if err != nil {
		return false, err
	}

	if !isReady || unsafeHashResponse == "" {
		return false, fmt.Errorf("sequencer %s is not ready", sequencer)
	}

	// Use unsafeHash from the response if initial unsafeHash is empty
	if unsafeHash == "" {
		unsafeHash = unsafeHashResponse
	}

	err = rpc.ActivateSequencer(sequencer, unsafeHash)
	if err != nil {
		// Ensure this sequencer is deactivated even it failed to activate
		_, _ = rpc.DeactivateSequencer(sequencer)
		return false, err
	}

	return true, nil
}

func Bootstrap(sequencersList []string) (int, error) {
	log := zap.L().With(zap.String("service", "heartbeat"))

	log.Debug("Determining current primary sequencer")
	primarySequencerID, err := findActivePrimary(sequencersList, log)

	if err != nil {
		return -1, err // Propagate error upwards
	}

	// Attempt to promote a new primary if no active primary was found
	if primarySequencerID == -1 {
		log.Info("No primary sequencer found, starting promotion process...")

		primarySequencerID, err = promoteNewPrimary(sequencersList)
		if err != nil {
			return -1, err // Promotion failed, propagate error
		}
	}

	log.Info("Primary sequencer is active.", zap.Int("id", primarySequencerID), zap.String("sequencer", sequencersList[primarySequencerID]))

	return primarySequencerID, nil
}

// findActivePrimary finds the active primary sequencer which is processing blocks
func findActivePrimary(sequencersList []string, log *zap.Logger) (int, error) {
	for id, sequencer := range sequencersList {
		isActive, err := rpc.CheckSequencerActive(sequencer)
		if err != nil {
			log.Error("Failed to get sequencer status", zap.Int("id", id), zap.String("sequencer", sequencer), zap.Error(err))
			continue
		}

		if isActive {
			log.Info("Found active primary sequencer", zap.Int("id", id), zap.String("sequencer", sequencer))
			deactivateExtraSequencers(id, sequencersList, log)

			return id, nil
		}
	}

	return -1, nil
}

// deactivateExtraSequencers deactivates all sequencers except the active primary
func deactivateExtraSequencers(primaryID int, sequencersList []string, log *zap.Logger) {
	for id, sequencer := range sequencersList {
		if id != primaryID {
			if _, err := rpc.DeactivateSequencer(sequencer); err != nil {
				log.Error("Failed to deactivate sequencer", zap.Int("id", id), zap.String("sequencer", sequencer), zap.Error(err))
			}
		}
	}
}

func promoteNewPrimary(sequencersList []string) (int, error) {
	primarySequencerID := activateSequencerByID(0, "", sequencersList)
	if primarySequencerID == -1 {
		return -1, fmt.Errorf("failed to activate any sequencers")
	}

	return primarySequencerID, nil
}

// Loop is the main heartbeat loop, which monitors the status of the primary sequencer
func (s *Service) Loop(primarySequencerID int) {
	log := zap.L().With(zap.String("service", "heartbeat"))

	currentBlockTime := time.Now()
	currentBlockHeight := int64(0)

	// begin the heartbeat loop
	for {
		time.Sleep(s.checkInterval)

		isActive, err := s.checkPrimarySequencerStatus(primarySequencerID, log)
		if err != nil {
			// primary sequencer is active, do nothing
			continue
		}

		if !isActive {
			log.Info("Primary sequencer is not active, switching...")
			primarySequencerID = s.handleSequencerFailure(primarySequencerID, "", log)

			continue
		}

		blockHeight, err := s.checkBlockHeight(primarySequencerID, log, currentBlockHeight, currentBlockTime)
		if err != nil || blockHeight == currentBlockHeight {
			// blockHeight is correct, do nothing
			continue
		}

		currentBlockHeight = blockHeight
		currentBlockTime = time.Now()
	}
}

func (s *Service) checkPrimarySequencerStatus(primarySequencerID int, log *zap.Logger) (bool, error) {
	isActive, err := rpc.CheckSequencerActive(s.sequencerList[primarySequencerID])

	if err != nil {
		log.Error("Failed to check primary sequencer status", zap.Error(err))
		return false, err
	}

	return isActive, nil
}

// checkBlockHeight checks the current block height of the primary sequencer
func (s *Service) checkBlockHeight(primarySequencerID int, log *zap.Logger, currentBlockHeight int64, currentBlockTime time.Time) (int64, error) {
	log.Debug("Start checking current block height")

	_, blockHeight, _, err := rpc.GetOPSyncStatus(s.sequencerList[primarySequencerID])

	if err != nil {
		log.Error("Failed to get block status from primary sequencer", zap.Error(err))

		return currentBlockHeight, err
	}

	if blockHeight > currentBlockHeight {
		log.Info("New block height found", zap.Int64("new_block_height", blockHeight))

		return blockHeight, nil
	}

	if time.Since(currentBlockTime) > s.maxBlockTime {
		log.Warn("Block time exceeds maximum tolerance, attempting to restart sequencer...")
		s.handleSequencerFailure(primarySequencerID, "", log)
	}

	return currentBlockHeight, nil
}

func (s *Service) handleSequencerFailure(currentSequencerID int, unsafeHash string, log *zap.Logger) int {
	log.Info("Handling failure of the primary sequencer", zap.Int("sequencer_id", currentSequencerID))

	_, err := rpc.DeactivateSequencer(s.sequencerList[currentSequencerID])

	if err != nil {
		log.Error("Failed to deactivate sequencer", zap.Error(err))
	}

	newPrimaryID := activateSequencerByID(currentSequencerID, unsafeHash, s.sequencerList)

	if newPrimaryID == -1 {
		log.Fatal("Failed to activate any sequencer")
	}

	log.Info("New primary sequencer activated.", zap.Int("new_primary_id", newPrimaryID))

	return newPrimaryID
}
