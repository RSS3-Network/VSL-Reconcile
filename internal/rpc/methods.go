package rpc

import (
	"bytes"
	"encoding/json"
	"fmt"
	"net/http"
	"time"
)

// jsonRPCCall: The function wraps method and params to JSON RPC call format, and then send to rpcEndpoint .
func jsonRPCCall[T any](method string, params []string, rpcEndpoint string) (*T, error) {
	var failCount = 0
	var returnErr error

	for failCount < JSONRPCCallFailRetry {
		if failCount > 0 {
			time.Sleep(JSONRPCCallRequestTimeout)
		}

		failCount++

		reqData := JSONRPCRequestData{
			Version: "2.0",
			Method:  method,
			Params:  params,
			ID:      1, // Only important for WS-RPC calls.
		}

		reqDataBytes, err := json.Marshal(&reqData)
		if err != nil {
			returnErr = fmt.Errorf("marshal request data: %w", err)
			continue
		}

		req, err := http.NewRequest("POST", rpcEndpoint, bytes.NewBuffer(reqDataBytes))
		if err != nil {
			returnErr = fmt.Errorf("initialize request: %w", err)
			continue
		}

		req.Header.Set("Content-Type", "application/json")

		res, err := (&http.Client{
			Timeout: JSONRPCCallRequestTimeout,
		}).Do(req)
		if err != nil {
			returnErr = fmt.Errorf("execute request: %w", err)
			continue
		}

		var resObj JSONRPCResponse[T]

		err = json.NewDecoder(res.Body).Decode(&resObj)
		_ = res.Body.Close() // Close to prevent memory leak
		if err != nil {
			returnErr = fmt.Errorf("decode response: %w", err)
		}

		if resObj.Error != nil {
			returnErr = fmt.Errorf("request error %d: %s", resObj.Error.Code, resObj.Error.Message)
		}

		// Success
		return resObj.Result, nil
	}

	return nil, returnErr
}

// CheckSequencerActive : Check if a sequencer is in active state
// {"jsonrpc":"2.0","id":1,"result":true} or {"jsonrpc":"2.0","id":1,"result":false}
// Sequencer can have some other status like just syncing as backup node, in which case it might print error like
// {"jsonrpc":"2.0","id":1,"error":{"code":-32601,"message":"the method admin_sequencerActive does not exist/is not available"}}
func CheckSequencerActive(sequencer string) (bool, error) {
	isActive, err := jsonRPCCall[bool]("admin_sequencerActive", []string{}, sequencer)
	if err != nil {
		return false, fmt.Errorf("jsonrpc request failed: %w", err)
	} else if isActive == nil {
		return false, fmt.Errorf("unknown response nil")
	}

	return *isActive, nil
}

// ActivateSequencer : Activate a sequencer as primary sequencer.
// Seems like we don't care about the result if only there's no errors.
func ActivateSequencer(sequencer string, unsafeHash string) error {
	_, err := jsonRPCCall[any]("admin_startSequencer", []string{unsafeHash}, sequencer)
	if err != nil {
		return fmt.Errorf("jsonrpc request failed: %w", err)
	}

	return nil
}

// DeactivateSequencer : Deactivate a sequencer and get current unsafe hash.
func DeactivateSequencer(sequencer string) (string, error) {
	unsafeHash, err := jsonRPCCall[string]("admin_stopSequencer", []string{}, sequencer)
	if err != nil {
		return "", fmt.Errorf("jsonrpc request failed: %w", err)
	} else if unsafeHash == nil {
		return "", fmt.Errorf("unknown response nil")
	}

	return *unsafeHash, nil
}

// GetOPSyncStatus : Get unsafe L2 Head from op sync status.
// This shouldn't be common as we can get unsafe header from deactivation request,
// but sometimes deactivation can fail. So use this as a fallback.
func GetOPSyncStatus(sequencer string) (string, int64, bool, error) {
	syncStatus, err := jsonRPCCall[struct { // Ignore irrelevant fields
		HeadL1 struct {
			// Hash       string `json:"hash"`
			// Number     int    `json:"number"`
			// ParentHash string `json:"parentHash"`
			Timestamp int64 `json:"timestamp"` // For check if sequencer is ready to be activated ( 12s * 3 )
		} `json:"head_l1"`
		UnsafeL2 struct {
			Hash   string `json:"hash"`
			Number int64  `json:"number"`
			//Timestamp int64 `json:"timestamp"` // Not for isReady status reference
		} `json:"unsafe_l2"`
	}]("optimism_syncStatus", []string{}, sequencer)
	if err != nil {
		return "", 0, false, fmt.Errorf("jsonrpc request failed: %w", err)
	} else if syncStatus == nil {
		return "", 0, false, fmt.Errorf("unknown response nil")
	}

	return syncStatus.UnsafeL2.Hash, syncStatus.UnsafeL2.Number, // unsafe hash
		time.Now().Unix()-syncStatus.HeadL1.Timestamp < MaxMainnetBlockTimestampLateTolerance, // is sequencer sync with mainnet (max tolerance 3 blocks behind) and ready to be activated
		nil
}
