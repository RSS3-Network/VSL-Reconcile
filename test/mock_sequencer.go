package test

import (
	"encoding/json"
	"fmt"
	"net"
	"net/http"
	"time"
)

/********************* Defile data types *********************/

type JSONRPCRequestData struct {
	Version string   `json:"jsonrpc"` // 2.0
	Method  string   `json:"method"`
	Params  []string `json:"params"`
	ID      uint     `json:"id"` // Request ID
}

type JSONRPCResponse[T any] struct {
	Version string                `json:"jsonrpc"` // 2.0
	ID      uint                  `json:"id"`      // Request ID
	Error   *JSONRPCResponseError `json:"error"`
	Result  *T                    `json:"result"`
}

type JSONRPCResponseError struct { // Possible error
	Code    int    `json:"code"`
	Message string `json:"message"`
}

type HeadL1Status struct {
	// Hash       string `json:"hash"`
	// Number     int    `json:"number"`
	// ParentHash string `json:"parentHash"`
	Timestamp int64 `json:"timestamp"` // For check if sequencer is ready to be activated ( 12s * 3 )
}

type UnsafeL2Status struct {
	Hash   string `json:"hash"`
	Number int64  `json:"number"`
	// Timestamp int64 `json:"timestamp"` // Not for isReady status reference
}

type OPSyncStatus struct { // Ignore irrelevant fields
	HeadL1   HeadL1Status   `json:"head_l1"`
	UnsafeL2 UnsafeL2Status `json:"unsafe_l2"`
}

/********************* Initialize mock sequencer *********************/

type MockSequencer struct {
	listener net.Listener
	server   http.Server

	isWithAdmin bool // Can be activated
	isActivated bool // Is now activated
	isReady     bool // Is sync with mainnet

	unsafeHash string // Unsafe L2 block hash
}

func NewMockSequencer() (*MockSequencer, string, error) {
	// Create a mock sequencer object
	ms := &MockSequencer{
		isActivated: false,
	}

	var err error

	// Listen with a random port
	ms.listener, err = net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		return nil, "", err
	}

	// Prepare handlers
	mockSequencerMux := http.NewServeMux()
	mockSequencerMux.HandleFunc("/", ms.handleJSONRPC)

	// Prepare server
	ms.server = http.Server{
		Handler:      mockSequencerMux,
		ReadTimeout:  5 * time.Second,
		WriteTimeout: 10 * time.Second,
		IdleTimeout:  120 * time.Second,
	}

	// Server start
	go func() {
		err := ms.server.Serve(ms.listener)
		if err != nil {
			_ = ms.listener.Close()
		}
	}()

	// Finish create
	return ms, fmt.Sprintf("http://%s/", ms.listener.Addr().String()), nil
}

/********************* Handle JSONRPC requests *********************/

func noSuchMethod(method string) string {
	return fmt.Sprintf("the method %s does not exist/is not available", method)
}

func (ms *MockSequencer) handleJSONRPC(w http.ResponseWriter, req *http.Request) {
	// Parse request
	var reqBody JSONRPCRequestData

	_ = json.NewDecoder(req.Body).Decode(&reqBody)

	// Prepare response
	var resBodyBytes []byte

	switch reqBody.Method {
	case "admin_sequencerActive":
		resBody := JSONRPCResponse[bool]{
			Version: reqBody.Version,
			ID:      reqBody.ID,
		}

		if ms.isWithAdmin {
			resBody.Result = &ms.isActivated
		} else {
			resBody.Error = &JSONRPCResponseError{
				-32601,
				noSuchMethod("admin_sequencerActive"),
			}
		}

		resBodyBytes, _ = json.Marshal(&resBody)

	case "admin_startSequencer":
		resBody := JSONRPCResponse[any]{
			Version: reqBody.Version,
			ID:      reqBody.ID,
		}

		if ms.isWithAdmin {
			if ms.isActivated {
				resBody.Error = &JSONRPCResponseError{
					-32000,
					"sequencer already running",
				}
			} else {
				ms.unsafeHash = reqBody.Params[0]
				ms.isActivated = true
			}
		} else {
			resBody.Error = &JSONRPCResponseError{
				-32601,
				noSuchMethod("admin_startSequencer"),
			}
		}

		resBodyBytes, _ = json.Marshal(&resBody)

	case "admin_stopSequencer":
		resBody := JSONRPCResponse[string]{
			Version: reqBody.Version,
			ID:      reqBody.ID,
		}

		if ms.isWithAdmin {
			if !ms.isActivated {
				resBody.Error = &JSONRPCResponseError{
					-32000,
					"sequencer not running",
				}
			} else {
				resBody.Result = &ms.unsafeHash
				ms.isActivated = false
			}
		} else {
			resBody.Error = &JSONRPCResponseError{
				-32601,
				noSuchMethod("admin_stopSequencer"),
			}
		}

		resBodyBytes, _ = json.Marshal(&resBody)

	case "optimism_syncStatus":
		resBody := JSONRPCResponse[OPSyncStatus]{
			Version: reqBody.Version,
			ID:      reqBody.ID,
		}

		resBody.Result = &OPSyncStatus{
			HeadL1: HeadL1Status{
				Timestamp: 0,
			},
			UnsafeL2: UnsafeL2Status{
				Hash: ms.unsafeHash,
			},
		}

		if ms.isReady {
			resBody.Result.HeadL1.Timestamp = time.Now().Unix()
		}

		resBodyBytes, _ = json.Marshal(&resBody)

	default:
		var resBody JSONRPCResponse[any]

		resBody.Error = &JSONRPCResponseError{
			-32601,
			noSuchMethod(reqBody.Method),
		}

		resBodyBytes, _ = json.Marshal(&resBody)
	}

	w.Header().Set("Content-Type", "application/json")
	_, _ = w.Write(resBodyBytes)
}

func (ms *MockSequencer) Close() {
	_ = ms.server.Close()
	_ = ms.listener.Close()
}

/********************* Manage mock sequencer status *********************/

func (ms *MockSequencer) SetIsWithAdmin(isWithAdmin bool) {
	ms.isWithAdmin = isWithAdmin
}

func (ms *MockSequencer) GetIsWithAdmin() bool {
	return ms.isWithAdmin
}

func (ms *MockSequencer) SetIsActivated(isActivated bool) {
	ms.isActivated = isActivated
}

func (ms *MockSequencer) GetIsActivated() bool {
	return ms.isActivated
}

func (ms *MockSequencer) SetIsReady(isReady bool) {
	ms.isReady = isReady
}

func (ms *MockSequencer) GetIsReady() bool {
	return ms.isReady
}

func (ms *MockSequencer) SetUnsafeHash(hash string) {
	ms.unsafeHash = hash
}

func (ms *MockSequencer) GetUnsafeHash() string {
	return ms.unsafeHash
}
