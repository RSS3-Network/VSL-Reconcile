package rpc

import (
	"testing"

	"github.com/rss3-network/vsl-reconcile/test"
)

func TestCheckSequencerActive(t *testing.T) {
	t.Parallel()

	// Prepare mock sequencer
	ms, endpoint, err := test.NewMockSequencer()

	if err != nil {
		t.Fatal(err)
	}

	defer ms.Close()

	// Situation 1: sequencer disable admin endpoints

	ms.SetIsWithAdmin(false)

	isSequencerActive, err := CheckSequencerActive(endpoint)

	if err == nil {
		t.Log("should be error")
		t.Fail()
	}

	if isSequencerActive {
		t.Log("should not active")
		t.Fail()
	}

	// Situation 2: sequencer enable admin endpoints, but not activated

	ms.SetIsWithAdmin(true)

	ms.SetIsActivated(false)

	isSequencerActive, err = CheckSequencerActive(endpoint)

	if err != nil {
		t.Log("should no error", err)
		t.Fail()
	}

	if isSequencerActive {
		t.Log("should not active")
		t.Fail()
	}

	// Situation 3: sequencer enable admin endpoints, and activated

	ms.SetIsWithAdmin(true)

	ms.SetIsActivated(true)

	isSequencerActive, err = CheckSequencerActive(endpoint)

	if err != nil {
		t.Log("should no error", err)
		t.Fail()
	}

	if !isSequencerActive {
		t.Log("should be active")
		t.Fail()
	}
}

func TestActivateSequencer(t *testing.T) {
	t.Parallel()

	// Prepare mock sequencer

	ms, endpoint, err := test.NewMockSequencer()

	if err != nil {
		t.Fatal(err)
	}

	defer ms.Close()

	// Situation 1: sequencer disable admin endpoints

	ms.SetIsWithAdmin(false)

	err = ActivateSequencer(endpoint, "unsafe-hash")

	if err == nil {
		t.Log("should be error")
		t.Fail()
	}

	// Situation 2: sequencer enable admin endpoints, but not activated

	ms.SetIsWithAdmin(true)

	ms.SetIsActivated(false)

	err = ActivateSequencer(endpoint, "unsafe-hash")

	if err != nil {
		t.Log("should no error", err)
		t.Fail()
	}

	// Situation 3: sequencer enable admin endpoints, and activated

	ms.SetIsWithAdmin(true)

	ms.SetIsActivated(true)

	err = ActivateSequencer(endpoint, "unsafe-hash")

	if err == nil {
		t.Log("should be error")
		t.Fail()
	}
}

func TestDeactivateSequencer(t *testing.T) {
	t.Parallel()

	// Prepare mock sequencer

	ms, endpoint, err := test.NewMockSequencer()

	if err != nil {
		t.Fatal(err)
	}

	defer ms.Close()

	// Situation 1: sequencer disable admin endpoints

	ms.SetIsWithAdmin(false)

	ms.SetUnsafeHash("unsafe-hash-1")

	unsafeHash, err := DeactivateSequencer(endpoint)

	if err == nil {
		t.Log("should be error")
		t.Fail()
	}

	if unsafeHash != "" {
		t.Log("unsafeHash should be empty")
		t.Fail()
	}

	// Situation 2: sequencer enable admin endpoints, but not activated

	ms.SetIsWithAdmin(true)

	ms.SetIsActivated(false)

	ms.SetUnsafeHash("unsafe-hash-2")

	unsafeHash, err = DeactivateSequencer(endpoint)

	if err == nil {
		t.Log("should be error")
		t.Fail()
	}

	if unsafeHash != "" {
		t.Log("unsafeHash should be empty")
		t.Fail()
	}

	// Situation 3: sequencer enable admin endpoints, and activated

	ms.SetIsWithAdmin(true)

	ms.SetIsActivated(true)

	ms.SetUnsafeHash("unsafe-hash-3")

	unsafeHash, err = DeactivateSequencer(endpoint)

	if err != nil {
		t.Log("should no error", err)
		t.Fail()
	}

	if unsafeHash != "unsafe-hash-3" {
		t.Log("unsafeHash mismatch", unsafeHash)
		t.Fail()
	}
}

func TestGetOPSyncStatus(t *testing.T) {
	t.Parallel()

	// Prepare mock sequencer

	ms, endpoint, err := test.NewMockSequencer()

	if err != nil {
		t.Fatal(err)
	}

	defer ms.Close()

	// Situation 1: sequencer not ready

	ms.SetIsReady(false)

	ms.SetUnsafeHash("unsafe-hash-1")

	unsafeHash, _, isReady, err := GetOPSyncStatus(endpoint)

	if err != nil {
		t.Log("should no error", err)
		t.Fail()
	}

	if unsafeHash != "unsafe-hash-1" {
		t.Log("unsafeHash should not empty")
		t.Fail()
	}

	if isReady {
		t.Log("should not ready")
		t.Fail()
	}

	// Situation 2: sequencer ready

	ms.SetIsReady(true)

	ms.SetUnsafeHash("unsafe-hash-2")

	unsafeHash, _, isReady, err = GetOPSyncStatus(endpoint)

	if err != nil {
		t.Log("should no error", err)
		t.Fail()
	}

	if unsafeHash != "unsafe-hash-2" {
		t.Log("unsafeHash should not empty")
		t.Fail()
	}

	if !isReady {
		t.Log("should be ready")
		t.Fail()
	}
}
