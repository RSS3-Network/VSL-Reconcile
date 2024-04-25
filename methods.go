package main

import (
	"bytes"
	"encoding/json"
	"fmt"
	"net/http"
	"time"
)

// jsonRPCCall: The function wraps method and params to JSON RPC call format, and then send to rpcEndpoint .
func jsonRPCCall[T any](method string, params []string, rpcEndpoint string) (*T, error) {

	reqData := JSONRPCRequestData{
		Version: "2.0",
		Method:  method,
		Params:  params,
		ID:      1, // Only important for WS-RPC calls.
	}

	reqDataBytes, err := json.Marshal(&reqData)
	if err != nil {
		return nil, fmt.Errorf("marshal request data: %w", err)
	}

	req, err := http.NewRequest("POST", rpcEndpoint, bytes.NewBuffer(reqDataBytes))
	if err != nil {
		return nil, fmt.Errorf("initialize request: %w", err)
	}

	req.Header.Set("Content-Type", "application/json")

	res, err := (&http.Client{
		Timeout: 1 * time.Second,
	}).Do(req)
	if err != nil {
		return nil, fmt.Errorf("execute request: %w", err)
	}

	var resObj JSONRPCResponse[T]

	err = json.NewDecoder(res.Body).Decode(&resObj)
	_ = res.Body.Close() // Close to prevent memory leak
	if err != nil {
		return nil, fmt.Errorf("decode response: %w", err)
	}

	if resObj.Error != nil {
		return nil, fmt.Errorf("request error %d: %s", resObj.Error.Code, resObj.Error.Message)
	}

	return resObj.Result, nil
}

// checkSequencerActive: Check if a sequencer is in active state
// {"jsonrpc":"2.0","id":1,"result":true} or {"jsonrpc":"2.0","id":1,"result":false}
// Sequencer can have some other status like just syncing as backup node, in which case it might print error like
// {"jsonrpc":"2.0","id":1,"error":{"code":-32601,"message":"the method admin_sequencerActive does not exist/is not available"}}
func checkSequencerActive(sequencer string) (bool, error) {
	isActive, err := jsonRPCCall[bool]("admin_sequencerActive", []string{}, sequencer)
	if err != nil {
		return false, fmt.Errorf("jsonrpc request failed: %w", err)
	} else if isActive == nil {
		return false, fmt.Errorf("unknown response nil")
	}

	return *isActive, nil
}

// activateSequencer: Activate a sequencer as primary sequencer.
// Seems like we don't care about the result if only there's no errors.
func activateSequencer(unsafeHash string, sequencer string) error {
	_, err := jsonRPCCall[any]("admin_startSequencer", []string{unsafeHash}, sequencer)
	if err != nil {
		return fmt.Errorf("jsonrpc request failed: %w", err)
	}

	return nil
}

// deactivateSequencer: Deactivate a sequencer and get current unsafe hash.
func deactivateSequencer(sequencer string) (string, error) {
	unsafeHash, err := jsonRPCCall[string]("admin_stopSequencer", []string{}, sequencer)
	if err != nil {
		return "", fmt.Errorf("jsonrpc request failed: %w", err)
	} else if unsafeHash == nil {
		return "", fmt.Errorf("unknown response nil")
	}

	return *unsafeHash, nil
}

// getUnsafeL2Status: Get unsafe L2 Head from op sync status.
// This shouldn't be common as we can get unsafe header from deactivation request,
// but sometimes deactivation can fail. So use this as a fallback.
func getUnsafeL2Status(sequencer string) (string, int64, error) {
	syncStatus, err := jsonRPCCall[struct { // Ignore irrelevant fields
		UnsafeL2 struct {
			Hash   string `json:"hash"`
			Number int64  `json:"number"`
		} `json:"unsafe_l2"`
	}]("optimism_syncStatus", []string{}, sequencer)
	if err != nil {
		return "", 0, fmt.Errorf("jsonrpc request failed: %w", err)
	} else if syncStatus == nil {
		return "", 0, fmt.Errorf("unknown response nil")
	}

	return syncStatus.UnsafeL2.Hash, syncStatus.UnsafeL2.Number, nil // unsafe hash
}
