package heartbeat

import (
	"testing"

	"github.com/rss3-network/vsl-reconcile/test"
)

func Test_activateSequencerWithFirstID(t *testing.T) {
	t.Parallel()

	// Prepare sequencers
	sequencersCount := 3

	sequencers := make([]*test.MockSequencer, sequencersCount)

	endpoints := make([]string, sequencersCount)

	var (
		err error
	)

	for i := 0; i < sequencersCount; i++ {
		sequencers[i], endpoints[i], err = test.NewMockSequencer()

		if err != nil {
			t.Fatal("failed to prepare mock sequencer", i, err)
		}
	}

	defer func() {
		for _, sequencer := range sequencers {
			sequencer.Close()
		}
	}()

	activateSequencerWithFirstIDCondition1(t, sequencers, endpoints)

	activateSequencerWithFirstIDCondition2(t, sequencers, endpoints)

	activateSequencerWithFirstIDCondition3(t, sequencers, endpoints)
}

func activateSequencerWithFirstIDCondition1(t *testing.T, sequencers []*test.MockSequencer, endpoints []string) {
	// Condition 1: all sequencers stopped, all is ready
	startWithID := 0

	// Situation 1: Start with 0, should activate 0

	for _, ms := range sequencers {
		ms.SetIsWithAdmin(true)
		ms.SetIsActivated(false)
		ms.SetIsReady(true)
		ms.SetUnsafeHash("unsafe-hash-1")
	}

	startWithID = 0

	activatedSequencerID := activateSequencerWithFirstID(startWithID, "unsafe-hash-1.1", endpoints)

	if activatedSequencerID != startWithID {
		t.Log("activated wrong sequencer", activatedSequencerID)

		t.Fail()
	}

	for i, ms := range sequencers {
		if ms.GetIsActivated() != (i == startWithID) {
			t.Log("sequencer state is incorrect", i)

			t.Fail()
		}
	}

	// Situation 2: Start with 1, should activate 1
	for _, ms := range sequencers {
		ms.SetIsWithAdmin(true)

		ms.SetIsActivated(false)

		ms.SetIsReady(true)
	}

	startWithID = 1

	activatedSequencerID = activateSequencerWithFirstID(startWithID, "unsafe-hash-1.2", endpoints)

	if activatedSequencerID != startWithID {
		t.Log("activated wrong sequencer", activatedSequencerID)

		t.Fail()
	}

	for i, ms := range sequencers {
		if ms.GetIsActivated() != (i == startWithID) {
			t.Log("sequencer state is incorrect", i)

			t.Fail()
		}
	}
}

func activateSequencerWithFirstIDCondition2(t *testing.T, sequencers []*test.MockSequencer, endpoints []string) {
	// Condition 2: all sequencers stopped, some is not ready
	notReadyIndex := 0

	// Situation 1: Start with 0, should activate 1
	for i, ms := range sequencers {
		ms.SetIsWithAdmin(true)

		ms.SetIsActivated(false)

		ms.SetIsReady(i != notReadyIndex)

		ms.SetUnsafeHash("unsafe-hash-2")
	}

	activatedSequencerID := activateSequencerWithFirstID(0, "unsafe-hash-2.1", endpoints)

	if activatedSequencerID != 1 {
		t.Log("activated wrong sequencer", activatedSequencerID)

		t.Fail()
	}

	for i, ms := range sequencers {
		if ms.GetIsActivated() != (i == 1) {
			t.Log("sequencer state is incorrect", i)

			t.Fail()
		}
	}

	// Situation 2: Start with 2, should activate 2
	for i, ms := range sequencers {
		ms.SetIsWithAdmin(true)

		ms.SetIsActivated(false)

		ms.SetIsReady(i != notReadyIndex)
	}

	activatedSequencerID = activateSequencerWithFirstID(2, "unsafe-hash-2.2", endpoints)

	if activatedSequencerID != 2 {
		t.Log("activated wrong sequencer", activatedSequencerID)

		t.Fail()
	}

	for i, ms := range sequencers {
		if ms.GetIsActivated() != (i == 2) {
			t.Log("sequencer state is incorrect", i)

			t.Fail()
		}
	}

	// Situation 3: Start with 2, should activate 0
	notReadyIndex = 2

	for i, ms := range sequencers {
		ms.SetIsWithAdmin(true)

		ms.SetIsActivated(false)

		ms.SetIsReady(i != notReadyIndex)
	}

	activatedSequencerID = activateSequencerWithFirstID(2, "unsafe-hash-2.3", endpoints)

	if activatedSequencerID != 0 {
		t.Log("activated wrong sequencer", activatedSequencerID)

		t.Fail()
	}

	for i, ms := range sequencers {
		if ms.GetIsActivated() != (i == 0) {
			t.Log("sequencer state is incorrect", i)

			t.Fail()
		}
	}
}

func activateSequencerWithFirstIDCondition3(t *testing.T, sequencers []*test.MockSequencer, endpoints []string) {
	// Condition 1: all sequencers stopped, none is ready
	// Situation 1: Start with 0, should no active
	for _, ms := range sequencers {
		ms.SetIsWithAdmin(true)

		ms.SetIsActivated(false)

		ms.SetIsReady(false)

		ms.SetUnsafeHash("unsafe-hash-3")
	}

	activatedSequencerID := activateSequencerWithFirstID(0, "unsafe-hash-3.1", endpoints)

	if activatedSequencerID != -1 {
		t.Log("activated wrong sequencer", activatedSequencerID)

		t.Fail()
	}

	for i, ms := range sequencers {
		if ms.GetIsActivated() != (i == -1) {
			t.Log("sequencer state is incorrect", i)

			t.Fail()
		}
	}

	// Reset
	for _, ms := range sequencers {
		ms.SetIsWithAdmin(true)

		ms.SetIsActivated(false)

		ms.SetIsReady(false)
	}

	// Situation 2: Start with 1, should no active
	activatedSequencerID = activateSequencerWithFirstID(1, "unsafe-hash-3.2", endpoints)

	if activatedSequencerID != -1 {
		t.Log("activated wrong sequencer", activatedSequencerID)

		t.Fail()
	}

	for i, ms := range sequencers {
		if ms.GetIsActivated() != (i == -1) {
			t.Log("sequencer state is incorrect", i)

			t.Fail()
		}
	}
}

func TestBootstrap(t *testing.T) {
	t.Parallel()

	// Prepare sequencers
	sequencersCount := 3

	sequencers := make([]*test.MockSequencer, sequencersCount)

	endpoints := make([]string, sequencersCount)

	var (
		err error
	)

	for i := 0; i < sequencersCount; i++ {
		sequencers[i], endpoints[i], err = test.NewMockSequencer()

		if err != nil {
			t.Fatal("failed to prepare mock sequencer", i, err)
		}
	}

	defer func() {
		for _, sequencer := range sequencers {
			sequencer.Close()
		}
	}()

	BootstrapCondition1(t, sequencers, endpoints)

	BootstrapCondition2(t, sequencers, endpoints)

	BootstrapCondition3(t, sequencers, endpoints)

	BootstrapCondition4(t, sequencers, endpoints)

	BootstrapCondition5(t, sequencers, endpoints)
}

func BootstrapCondition1(t *testing.T, sequencers []*test.MockSequencer, endpoints []string) {
	// Condition 1: all sequencers stopped, all is ready
	// Situation 1: Should activate 0
	for _, ms := range sequencers {
		ms.SetIsWithAdmin(true)

		ms.SetIsActivated(false)

		ms.SetIsReady(true)

		ms.SetUnsafeHash("unsafe-hash-1.1")
	}

	activatedSequencerID, err := Bootstrap(endpoints)

	if err != nil {
		t.Log("should no error", err)

		t.Fail()
	}

	if !sequencers[0].GetIsActivated() {
		t.Log("should be activated")

		t.Fail()
	}

	if activatedSequencerID != 0 {
		t.Log("activated wrong sequencer", activatedSequencerID)

		t.Fail()
	}
}

func BootstrapCondition2(t *testing.T, sequencers []*test.MockSequencer, endpoints []string) {
	// Condition 2: all sequencers stopped, some is not ready
	notReadyIndex := 0

	for i, ms := range sequencers {
		ms.SetIsWithAdmin(true)

		ms.SetIsActivated(false)

		ms.SetIsReady(i != notReadyIndex)

		ms.SetUnsafeHash("unsafe-hash-2.1")
	}

	// Situation 1: Should activate 1

	activatedSequencerID, err := Bootstrap(endpoints)

	if err != nil {
		t.Log("should no error", err)

		t.Fail()
	}

	if !sequencers[1].GetIsActivated() {
		t.Log("should be activated")

		t.Fail()
	}

	if activatedSequencerID != 1 {
		t.Log("activated wrong sequencer", activatedSequencerID)

		t.Fail()
	}
}

func BootstrapCondition3(t *testing.T, sequencers []*test.MockSequencer, endpoints []string) {
	// Condition 3: one sequencer started, all is ready
	activeIndex := 1

	// Situation 1: Started sequencer is ready, should keep activated index
	for i, ms := range sequencers {
		ms.SetIsWithAdmin(true)

		ms.SetIsActivated(i == activeIndex)

		ms.SetIsReady(true)

		ms.SetUnsafeHash("unsafe-hash-3.1")
	}

	activatedSequencerID, err := Bootstrap(endpoints)

	if err != nil {
		t.Log("should no error", err)

		t.Fail()
	}

	if activatedSequencerID != activeIndex {
		t.Log("activated wrong sequencer", activatedSequencerID)

		t.Fail()
	}

	for i, ms := range sequencers {
		if ms.GetIsActivated() != (i == activeIndex) {
			t.Log("sequencer state is incorrect", i)

			t.Fail()
		}
	}
}

func BootstrapCondition4(t *testing.T, sequencers []*test.MockSequencer, endpoints []string) {
	// Condition 4: multiple sequencer started, all is ready
	// Situation 1: Should activate 0 and deactivate others
	for _, ms := range sequencers {
		ms.SetIsWithAdmin(true)

		ms.SetIsActivated(true)

		ms.SetIsReady(true)

		ms.SetUnsafeHash("unsafe-hash-4.1")
	}

	activatedSequencerID, err := Bootstrap(endpoints)

	if err != nil {
		t.Log("should no error", err)

		t.Fail()
	}

	if activatedSequencerID != 0 {
		t.Log("activated wrong sequencer", activatedSequencerID)

		t.Fail()
	}

	for i, ms := range sequencers {
		if ms.GetIsActivated() != (i == 0) {
			t.Log("sequencer state is incorrect", i)

			t.Fail()
		}
	}
}

func BootstrapCondition5(t *testing.T, sequencers []*test.MockSequencer, endpoints []string) {
	// Condition 5: no sequencer ready
	// Situation 1: Ready w/ empty unsafe hash (invalid state)
	for _, ms := range sequencers {
		ms.SetIsWithAdmin(true)

		ms.SetIsActivated(false)

		ms.SetIsReady(true)

		ms.SetUnsafeHash("") // Empty is invalid
	}

	activatedSequencerID, err := Bootstrap(endpoints)

	if err == nil {
		t.Log("should be error")

		t.Fail()
	}

	if activatedSequencerID != -1 {
		t.Log("activated wrong sequencer", activatedSequencerID)

		t.Fail()
	}

	for i, ms := range sequencers {
		if ms.GetIsActivated() != (i == -1) {
			t.Log("sequencer state is incorrect", i)

			t.Fail()
		}
	}

	// Situation 2: Not ready
	for _, ms := range sequencers {
		ms.SetIsWithAdmin(true)

		ms.SetIsActivated(false)

		ms.SetIsReady(false)

		ms.SetUnsafeHash("unsafe-hash-5.2")
	}

	activatedSequencerID, err = Bootstrap(endpoints)

	if err == nil {
		t.Log("should be error")

		t.Fail()
	}

	if activatedSequencerID != -1 {
		t.Log("activated wrong sequencer", activatedSequencerID)

		t.Fail()
	}

	for i, ms := range sequencers {
		if ms.GetIsActivated() != (i == -1) {
			t.Log("sequencer state is incorrect", i)

			t.Fail()
		}
	}
}
