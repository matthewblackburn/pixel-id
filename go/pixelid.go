package pixelid

import (
	"errors"
	"fmt"
	"sync"
	"time"
)

// DefaultEpoch is 2025-01-01T00:00:00Z. With 41 bits of millisecond
// timestamps, IDs are valid until approximately 2094.
var DefaultEpoch = time.Date(2025, 1, 1, 0, 0, 0, 0, time.UTC)

const (
	timestampBits = 41
	machineBits   = 10
	sequenceBits  = 12

	maxMachineID = (1 << machineBits) - 1  // 1023
	maxSequence  = (1 << sequenceBits) - 1 // 4095

	machineShift   = sequenceBits
	timestampShift = machineBits + sequenceBits

	maxClockDriftMs = 5
)

// Generator produces unique 64-bit snowflake IDs.
//
//	┌──────┬───────────────────────┬────────────────┬──────────────┐
//	│ Sign │ Timestamp (41 bits)   │ Machine (10)   │ Sequence (12)│
//	│  0   │ ms since epoch        │ 0..1023        │ 0..4095      │
//	└──────┴───────────────────────┴────────────────┴──────────────┘
type Generator struct {
	mu        sync.Mutex
	epoch     int64  // custom epoch in Unix ms
	machineID int64  // 0..1023
	sequence  int64  // 0..4095
	lastTime  int64  // last timestamp in ms since epoch
	nowFn     func() time.Time // for testing
}

// GeneratorOption configures a Generator.
type GeneratorOption func(*Generator)

// WithEpoch sets a custom epoch. Default is 2025-01-01T00:00:00Z.
func WithEpoch(epoch time.Time) GeneratorOption {
	return func(g *Generator) {
		g.epoch = epoch.UnixMilli()
	}
}

// WithMachineID sets the machine/worker ID (0..1023). Panics if >1023.
func WithMachineID(id uint16) GeneratorOption {
	return func(g *Generator) {
		if id > maxMachineID {
			panic(fmt.Sprintf("pixelid: machine ID %d exceeds maximum %d", id, maxMachineID))
		}
		g.machineID = int64(id)
	}
}

// NewGenerator creates a new ID generator.
func NewGenerator(opts ...GeneratorOption) *Generator {
	g := &Generator{
		epoch: DefaultEpoch.UnixMilli(),
		nowFn: time.Now,
	}
	for _, opt := range opts {
		opt(g)
	}
	return g
}

// Generate produces a new unique 64-bit ID.
//
// If the system clock drifts backward by <=5ms, the generator waits for
// time to catch up. If the drift exceeds 5ms, an error is returned.
func (g *Generator) Generate() (int64, error) {
	g.mu.Lock()
	defer g.mu.Unlock()

	now := g.nowFn().UnixMilli() - g.epoch

	if now == g.lastTime {
		g.sequence = (g.sequence + 1) & maxSequence
		if g.sequence == 0 {
			// Sequence overflow: wait until next millisecond.
			now = g.waitNextMs(g.lastTime)
		}
	} else if now > g.lastTime {
		g.sequence = 0
	} else {
		// Clock moved backward.
		drift := g.lastTime - now
		if drift <= maxClockDriftMs {
			// Small drift: wait for time to catch up.
			time.Sleep(time.Duration(drift) * time.Millisecond)
			now = g.nowFn().UnixMilli() - g.epoch
			if now < g.lastTime {
				return 0, fmt.Errorf("pixelid: clock still behind after waiting %dms", drift)
			}
			if now == g.lastTime {
				g.sequence = (g.sequence + 1) & maxSequence
				if g.sequence == 0 {
					now = g.waitNextMs(g.lastTime)
				}
			} else {
				g.sequence = 0
			}
		} else {
			return 0, fmt.Errorf("pixelid: clock moved backward by %dms (exceeds %dms tolerance)", drift, maxClockDriftMs)
		}
	}

	g.lastTime = now

	id := (now << timestampShift) | (g.machineID << machineShift) | g.sequence
	return id, nil
}

func (g *Generator) waitNextMs(last int64) int64 {
	for {
		now := g.nowFn().UnixMilli() - g.epoch
		if now > last {
			return now
		}
	}
}

// ParseID extracts the timestamp, machine ID, and sequence from an ID.
func ParseID(id int64) (timestamp time.Time, machineID uint16, sequence uint16) {
	return ParseIDWithEpoch(id, DefaultEpoch)
}

// ParseIDWithEpoch extracts components from an ID using a custom epoch.
func ParseIDWithEpoch(id int64, epoch time.Time) (timestamp time.Time, machineID uint16, sequence uint16) {
	ms := id >> timestampShift
	machineID = uint16((id >> machineShift) & maxMachineID)
	sequence = uint16(id & maxSequence)
	timestamp = time.UnixMilli(ms + epoch.UnixMilli())
	return
}

var (
	ErrClockBackward = errors.New("pixelid: clock moved backward")
)
