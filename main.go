/*
 * Sequencers watchdog for Optimism nodes
 *
 * For detailed API specifications
 * please refer to https://docs.optimism.io/builders/node-operators/json-rpc#admin
 */

package main

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"os"
	"strings"
	"time"
)

type jsonRPCRequestData struct {
	Version string   `json:"jsonrpc"` // 2.0
	Method  string   `json:"method"`
	Params  []string `json:"params"`
	ID      uint     `json:"id"` // Request ID
}

var requestIDCounter uint

func init() {
	requestIDCounter = 0
}

func jsonRPCSend(method string, params []string, rpcEndpoint string) ([]byte, int, error) {
	requestIDCounter++

	reqData := jsonRPCRequestData{
		Version: "2.0",
		Method:  method,
		Params:  params,
		ID:      requestIDCounter,
	}

	reqDataBytes, err := json.Marshal(&reqData)
	if err != nil {
		return nil, 0, fmt.Errorf("marshal request data: %w", err)
	}

	req, err := http.NewRequest("POST", rpcEndpoint, bytes.NewBuffer(reqDataBytes))
	if err != nil {
		return nil, 0, fmt.Errorf("initialize request: %w", err)
	}

	req.Header.Set("Content-Type", "application/json")

	res, err := (&http.Client{}).Do(req)
	if err != nil {
		return nil, 0, fmt.Errorf("execute request: %w", err)
	}

	resBytes, err := io.ReadAll(res.Body)

	_ = res.Body.Close()

	return resBytes, res.StatusCode, err
}

func checkSequencerActive(sequencer string) (bool, error) {
	res, status, err := jsonRPCSend("admin_sequencerActive", []string{}, sequencer)
	if err != nil {
		return false, fmt.Errorf("jsonrpc request failed: %w", err)
	}
	if status != http.StatusOK {
		return false, fmt.Errorf("unrecognized http status code: %d", status)
	}

	// TODO: parse res

	return res.isActive, nil
}

func activateSequencer(sequencer string) error {
	_, status, err := jsonRPCSend("admin_stopSequencer", []string{}, sequencer)
	if err != nil {
		return fmt.Errorf("jsonrpc request failed: %w", err)
	}
	if status != http.StatusOK {
		return fmt.Errorf("unrecognized http status code: %d", status)
	}

	return nil
}

func deactivateSequencer(sequencer string) error {
	_, status, err := jsonRPCSend("admin_startSequencer", []string{}, sequencer)
	if err != nil {
		return fmt.Errorf("jsonrpc request failed: %w", err)
	}
	if status != http.StatusOK {
		return fmt.Errorf("unrecognized http status code: %d", status)
	}

	return nil
}

func activateSequencerWithFirstID(firstID int, sequencersList []string) int {
	sequencersListLen := len(sequencersList)
	for i := 0; i < sequencersListLen; i++ {
		// Calculate absolute ID
		id := i + firstID
		if id >= sequencersListLen {
			id -= sequencersListLen
		}

		err := activateSequencer(sequencersList[id])
		if err != nil {
			log.Printf("failed to activate sequencer (%d): %v", id, err)
			continue
		}

		return id
	}

	return -1
}

func main() {
	// Read sequencers list from environment variable (comma-separated)
	sequencersList := strings.Split(os.Getenv("SEQUENCERS_LIST"), ",")

	if len(sequencersList) == 0 {
		// No sequencers specified, panic
		log.Fatalf("no sequencers specified")
	}

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
				err = deactivateSequencer(sequencer)
				if err != nil {
					log.Printf("failed to deactivate another active sequencer: %v", err)
				}
			}
		}
	}

	// If neither of these sequencers are primary, try to promote one
	if primarySequencerID == -1 {
		primarySequencerID = activateSequencerWithFirstID(0, sequencersList)
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
		// TODO

		// If current sequencer is working abnormally, promote next sequencer as primary
		// 1. deactivate this sequencer
		err = deactivateSequencer(sequencersList[primarySequencerID])
		if err != nil {
			log.Printf("failed to deactivate sequencer (%d): %v", primarySequencerID, err)
		}
		// 2. activate a new sequencer
		primarySequencerID = activateSequencerWithFirstID(primarySequencerID, sequencersList)
		if primarySequencerID == -1 {
			log.Fatalf("failed to activate any sequencer")
		}

	}

}
