package rpc

import (
	"encoding/json"
	"testing"
)

func TestMarshalOPSyncStatus(t *testing.T) {
	t.Parallel()

	rpcResponse := `
{
  "jsonrpc": "2.0",
  "id": 1,
  "result": {
    "current_l1": {
      "hash": "0x34e812ada03b9c4ca54999bd8dd62ed75af9829e8e857c64844189c8d3483bbe",
      "number": 5780555,
      "parentHash": "0x0cfd9ae6a62ffea510c11f22b98a622cce6b9765fd0e73741b284ba4bbb7e2b0",
      "timestamp": 1714122156
    },
    "current_l1_finalized": {
      "hash": "0xd4513892a7857981bb39723e68dbc3095f24e5c2d734301ee2284b35c2e5a43a",
      "number": 5780476,
      "parentHash": "0x5db35fcb0906dca02b61c5a054211ee1a5961c3b4fada4a5170505d45b5139ee",
      "timestamp": 1714121184
    },
    "head_l1": {
      "hash": "0x3b9763836a33d3642f1c70700e8e471d5855276504d976cd192e6b6db5a18c47",
      "number": 5780571,
      "parentHash": "0xb99ae979f48cc0e3044ee5185dea3b9ae060d099c9f253fcb3a791b4f70a9e72",
      "timestamp": 1714122348
    },
    "safe_l1": {
      "hash": "0x3a759b456173862e9b2a236188143e827cabc068d616260202b53ffd357c6a58",
      "number": 5780507,
      "parentHash": "0x4640c6aac130a6a7358b80971f00d0e991c7beb197a4569043f10140ce101a0c",
      "timestamp": 1714121568
    },
    "finalized_l1": {
      "hash": "0xd4513892a7857981bb39723e68dbc3095f24e5c2d734301ee2284b35c2e5a43a",
      "number": 5780476,
      "parentHash": "0x5db35fcb0906dca02b61c5a054211ee1a5961c3b4fada4a5170505d45b5139ee",
      "timestamp": 1714121184
    },
    "unsafe_l2": {
      "hash": "0x28b450267a12e6441fcfe02b9cf5cfa58f13b12404ab3e93430dbc24df3f174d",
      "number": 2730980,
      "parentHash": "0x0a42424df42d4f357ed069795820d286ad9be5e052361ea7f09108910d11072d",
      "timestamp": 1714122280,
      "l1origin": {
        "hash": "0x639e84b0ee445618f66952ce2bbbfb90b540cc9b3b4d1024e83f4068186eb9cf",
        "number": 5780549
      },
      "sequenceNumber": 5
    },
    "safe_l2": {
      "hash": "0x2278382317b5ce7812ad4516d1bd419314942859056579d0a026823418bb6ae9",
      "number": 2730824,
      "parentHash": "0xd53c3a7b5279f1dee41fe06f6936c20463d3ef28814fbbb3cd2e9468b4f74589",
      "timestamp": 1714121968,
      "l1origin": {
        "hash": "0xd44573969548fc8ea3a7242612ba554f360f21960eeb5d285e01d1fa3fb7051c",
        "number": 5780524
      },
      "sequenceNumber": 1
    },
    "finalized_l2": {
      "hash": "0xaea2074fe7029061296bd7b3bc4e65a87ec43b9bd2a58a0b6a55bbad96602db0",
      "number": 2730317,
      "parentHash": "0xe453a6ab0cebc07c68fe2ba985ebd5912e48b66e1d2f9e83427564ccd88fa00c",
      "timestamp": 1714120954,
      "l1origin": {
        "hash": "0x8921c17862d8ead2371f9828d31f88ac8663bbe6b2d5781f1fdafaf037838513",
        "number": 5780440
      },
      "sequenceNumber": 4
    },
    "pending_safe_l2": {
      "hash": "0x2278382317b5ce7812ad4516d1bd419314942859056579d0a026823418bb6ae9",
      "number": 2730824,
      "parentHash": "0xd53c3a7b5279f1dee41fe06f6936c20463d3ef28814fbbb3cd2e9468b4f74589",
      "timestamp": 1714121968,
      "l1origin": {
        "hash": "0xd44573969548fc8ea3a7242612ba554f360f21960eeb5d285e01d1fa3fb7051c",
        "number": 5780524
      },
      "sequenceNumber": 1
    }
  }
}
`

	var opSyncStatus JSONRPCResponse[struct { // Ignore irrelevant fields
		HeadL1 struct {
			Hash       string `json:"hash"`
			Number     int    `json:"number"`
			ParentHash string `json:"parentHash"`
			Timestamp  int64  `json:"timestamp"` // For check if sequencer is ready to be activated ( 12s * 3 )
		} `json:"head_l1"`
		UnsafeL2 struct {
			Hash   string `json:"hash"`
			Number int64  `json:"number"`
			// Timestamp int64 `json:"timestamp"` // Not for isReady status reference
		} `json:"unsafe_l2"`
	}]

	err := json.Unmarshal([]byte(rpcResponse), &opSyncStatus)
	if err != nil {
		t.Fatal(err)
	}

	t.Log(opSyncStatus.Result)
}
