package router

import (
	"sync"
	"time"

	"github.com/0skillallluck/scanline/app/preference"
	"github.com/0skillallluck/scanline/internal/signals"
	"github.com/jwijenbergh/puregotk/v4/gtk"
)

type HistoryEntry struct {
	ExpiresAt *time.Time
	PageTitle string
	Path      string
	View      *gtk.Widget
	Toolbar   *gtk.Widget
}

type History struct {
	sync.RWMutex
	Current *HistoryEntry
	Entries []*HistoryEntry
}

func (h *History) IsCurrentlyOn(path string) bool {
	h.RLock()
	defer h.RUnlock()

	if h.Current == nil {
		return false
	}

	if h.Current.Path != path {
		return false
	}

	return true
}

func (h *History) GetCurrent() *HistoryEntry {
	h.RLock()
	defer h.RUnlock()
	return h.Current
}

func (h *History) EntriesCount() int {
	h.RLock()
	defer h.RUnlock()
	return len(h.Entries)
}

func (h *History) InvalidateCurrent() (path string, ok bool) {
	h.Lock()
	defer h.Unlock()
	if h.Current == nil {
		return "", false
	}
	h.Current.Toolbar = nil
	h.Current.View = nil
	return h.Current.Path, true
}

func (h *History) Pop(updated *signals.StatelessSignal[*History]) *HistoryEntry {
	h.Lock()

	if len(h.Entries) == 0 {
		h.Unlock()
		return nil
	}

	h.Current = h.Entries[len(h.Entries)-1]
	h.Entries = h.Entries[:len(h.Entries)-1]

	if h.Current.ExpiresAt != nil && h.Current.ExpiresAt.Before(time.Now()) {
		h.Current.Toolbar = nil
		h.Current.View = nil
	}

	current := h.Current
	h.Unlock()
	updated.Notify(h)
	return current
}

func (h *History) Clear(updated *signals.StatelessSignal[*History]) {
	h.Lock()

	// Clear widget references from all entries to help GC
	for _, entry := range h.Entries {
		entry.View = nil
		entry.Toolbar = nil
	}
	if h.Current != nil {
		h.Current.View = nil
		h.Current.Toolbar = nil
	}

	h.Current = nil
	h.Entries = nil
	h.Unlock()

	updated.Notify(h)
}

func (h *History) Push(entry *HistoryEntry, updated *signals.StatelessSignal[*History]) {
	h.Lock()

	if h.Current != nil {
		if len(h.Entries) >= preference.Performance().MaxRouterHistorySize() {
			// Clear widget references from evicted entry to help GC
			evicted := h.Entries[0]
			evicted.View = nil
			evicted.Toolbar = nil
			h.Entries = h.Entries[1:]
		}

		h.Entries = append(h.Entries, h.Current)
	}

	h.Current = entry
	h.Unlock()

	updated.Notify(h)
}
