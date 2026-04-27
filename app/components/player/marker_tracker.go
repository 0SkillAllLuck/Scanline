package player

import (
	"log/slog"
	"sync/atomic"
)

// markerTracker drives one skip-button (intro or credits): it holds the
// resolved [start, end) marker offsets, edge-triggers visibility/log changes,
// and guards a single auto-skip per range entry.
//
// startMs/endMs sentinels: 0 = unresolved, -1 = no marker, >0 = valid.
type markerTracker struct {
	name        string
	startMs     atomic.Int64
	endMs       atomic.Int64
	inRange     atomic.Bool
	autoSkipped atomic.Bool
	setVisible  func(bool)
	autoSkip    func() bool
}

func (m *markerTracker) setNotFound() {
	m.startMs.Store(-1)
	m.endMs.Store(-1)
}

func (m *markerTracker) setRange(startMs, endMs int) {
	m.startMs.Store(int64(startMs))
	m.endMs.Store(int64(endMs))
}

// tick is called once per player ticker fire. It updates the button visibility
// on range entry/exit and triggers an auto-skip seek if enabled.
func (m *markerTracker) tick(positionUs int64, doSeek func(int64)) {
	startMs := m.startMs.Load()
	endMs := m.endMs.Load()
	if startMs <= 0 || endMs <= startMs {
		return
	}
	startUs := startMs * 1000
	endUs := endMs * 1000
	in := positionUs >= startUs && positionUs < endUs
	if m.inRange.Swap(in) != in {
		m.setVisible(in)
		slog.Debug("player: "+m.name+" range transition",
			"in_range", in,
			"position_ms", positionUs/1000,
			"start_ms", startMs,
			"end_ms", endMs)
	}
	if !in {
		m.autoSkipped.Store(false)
		return
	}
	if m.autoSkip() && m.autoSkipped.CompareAndSwap(false, true) {
		slog.Debug("player: auto-skipping "+m.name,
			"from_ms", positionUs/1000,
			"to_ms", endMs)
		doSeek(endUs)
	}
}

// seekToEnd handles a manual click on the skip button — seeks past the marker
// if it has been resolved.
func (m *markerTracker) seekToEnd(doSeek func(int64)) {
	if endMs := m.endMs.Load(); endMs > 0 {
		doSeek(endMs * 1000)
	}
}
