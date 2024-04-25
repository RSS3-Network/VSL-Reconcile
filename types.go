package main

type JSONRPCRequestData struct {
	Version string   `json:"jsonrpc"` // 2.0
	Method  string   `json:"method"`
	Params  []string `json:"params"`
	ID      uint     `json:"id"` // Request ID
}

type JSONRPCResponse[T any] struct {
	Version string    `json:"jsonrpc"` // 2.0
	ID      uint      `json:"id"`      // Request ID
	Error   *struct { // Possible error
		Code    int    `json:"code"`
		Message string `json:"message"`
	} `json:"error"`
	Result *T `json:"result"`
}
