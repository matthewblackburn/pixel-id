package pixelid

import (
	"sync"
	"testing"
	"time"
)

func TestGenerateHappyPath(t *testing.T) {
	g := NewGenerator(WithMachineID(1))

	id, err := g.Generate()
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if id <= 0 {
		t.Errorf("expected positive ID, got %d", id)
	}
}

func TestGenerateMonotonicallyIncreasing(t *testing.T) {
	g := NewGenerator(WithMachineID(0))

	var prev int64
	for i := 0; i < 100; i++ {
		id, err := g.Generate()
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if id <= prev {
			t.Errorf("ID %d not greater than previous %d", id, prev)
		}
		prev = id
	}
}

func TestGenerateSequenceIncrement(t *testing.T) {
	// Two IDs generated in quick succession should differ in sequence.
	g := NewGenerator(WithMachineID(0))

	id1, _ := g.Generate()
	id2, _ := g.Generate()

	seq1 := uint16(id1 & maxSequence)
	seq2 := uint16(id2 & maxSequence)

	ts1 := id1 >> timestampShift
	ts2 := id2 >> timestampShift

	if ts1 == ts2 && seq2 != seq1+1 {
		t.Errorf("same-ms IDs: seq1=%d seq2=%d, expected consecutive", seq1, seq2)
	}
}

func TestGenerateMachineID(t *testing.T) {
	g := NewGenerator(WithMachineID(42))
	id, _ := g.Generate()

	_, machineID, _ := ParseID(id)
	if machineID != 42 {
		t.Errorf("expected machine ID 42, got %d", machineID)
	}
}

func TestGenerateMachineIDPanic(t *testing.T) {
	defer func() {
		r := recover()
		if r == nil {
			t.Error("expected panic for machine ID > 1023")
		}
	}()
	NewGenerator(WithMachineID(1024))
}

func TestParseIDRoundTrip(t *testing.T) {
	g := NewGenerator(WithMachineID(512))

	before := time.Now()
	id, _ := g.Generate()
	after := time.Now()

	ts, machineID, seq := ParseID(id)

	if machineID != 512 {
		t.Errorf("machine ID: got %d, want 512", machineID)
	}
	if seq > maxSequence {
		t.Errorf("sequence %d exceeds max %d", seq, maxSequence)
	}
	if ts.Before(before.Truncate(time.Millisecond)) || ts.After(after.Add(time.Millisecond)) {
		t.Errorf("timestamp %v outside expected range [%v, %v]", ts, before, after)
	}
}

func TestGenerateSmallClockDrift(t *testing.T) {
	callCount := 0
	baseTime := time.Date(2025, 6, 1, 12, 0, 0, 0, time.UTC)

	g := NewGenerator(WithMachineID(0))
	g.nowFn = func() time.Time {
		callCount++
		switch {
		case callCount == 1:
			return baseTime
		case callCount == 2:
			// Clock goes backward by 3ms.
			return baseTime.Add(-3 * time.Millisecond)
		default:
			// After sleep, clock catches up.
			return baseTime.Add(1 * time.Millisecond)
		}
	}

	// First call establishes lastTime.
	_, err := g.Generate()
	if err != nil {
		t.Fatalf("first generate: %v", err)
	}

	// Second call encounters 3ms backward drift, should wait and succeed.
	_, err = g.Generate()
	if err != nil {
		t.Fatalf("expected small drift to be tolerated, got: %v", err)
	}
}

func TestGenerateLargeClockDrift(t *testing.T) {
	callCount := 0
	baseTime := time.Date(2025, 6, 1, 12, 0, 0, 0, time.UTC)

	g := NewGenerator(WithMachineID(0))
	g.nowFn = func() time.Time {
		callCount++
		if callCount == 1 {
			return baseTime
		}
		// Clock goes backward by 10ms.
		return baseTime.Add(-10 * time.Millisecond)
	}

	_, _ = g.Generate()

	_, err := g.Generate()
	if err == nil {
		t.Error("expected error for large clock drift")
	}
}

func TestGenerateConcurrency(t *testing.T) {
	g := NewGenerator(WithMachineID(0))

	const goroutines = 100
	const idsPerGoroutine = 1000

	var mu sync.Mutex
	allIDs := make(map[int64]bool, goroutines*idsPerGoroutine)
	var wg sync.WaitGroup

	errCh := make(chan error, goroutines)

	for i := 0; i < goroutines; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			for j := 0; j < idsPerGoroutine; j++ {
				id, err := g.Generate()
				if err != nil {
					errCh <- err
					return
				}
				mu.Lock()
				if allIDs[id] {
					mu.Unlock()
					errCh <- nil // will be caught by duplicate check below
					t.Errorf("duplicate ID: %d", id)
					return
				}
				allIDs[id] = true
				mu.Unlock()
			}
		}()
	}

	wg.Wait()
	close(errCh)

	for err := range errCh {
		if err != nil {
			t.Fatalf("generation error: %v", err)
		}
	}

	expected := goroutines * idsPerGoroutine
	if len(allIDs) != expected {
		t.Errorf("expected %d unique IDs, got %d", expected, len(allIDs))
	}
}

func TestWithCustomEpoch(t *testing.T) {
	epoch := time.Date(2020, 1, 1, 0, 0, 0, 0, time.UTC)
	g := NewGenerator(WithEpoch(epoch), WithMachineID(0))

	id, _ := g.Generate()

	ts, _, _ := ParseIDWithEpoch(id, epoch)
	now := time.Now()
	if ts.Before(now.Add(-time.Second)) || ts.After(now.Add(time.Second)) {
		t.Errorf("timestamp %v not close to now %v with custom epoch", ts, now)
	}
}
