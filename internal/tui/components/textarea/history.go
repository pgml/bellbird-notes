package textarea

import (
	"bellbird-notes/internal/app"
	"fmt"
)

type History struct {
	index      uint
	entryIndex int
	entries    []string
	maxItems   uint
}

func NewHistory() History {
	history := History{
		index:    0,
		entries:  []string{},
		maxItems: 100,
	}

	return history
}

func (h *History) NewEntry() {
	h.index++
	h.entries = append(h.entries, "")
	h.entryIndex = len(h.entries) - 1
}

func (h *History) UpdateEntry(s string) error {
	if len(h.entries) < h.entryIndex {
		app.LogErr("History entry index not found:", h.entryIndex)
		return fmt.Errorf("History entry index %d not found", h.entryIndex)
	}

	h.entries[h.entryIndex] = s
	return nil
}

func (h *History) Undo() string {
	h.entryIndex--
	if h.entryIndex < 0 {
		h.entryIndex = 0
	}
	return h.entries[h.entryIndex]
}

func (h *History) Redo() string {
	h.entryIndex++
	if h.entryIndex > len(h.entries) {
		h.entryIndex = len(h.entries) - 1
	}
	return h.entries[h.entryIndex]
}
