package textarea

import (
	"errors"
	"fmt"

	"bellbird-notes/app/debug"
)

type History struct {
	index      uint
	EntryIndex int
	entries    []Entry
	maxItems   uint
}

type Entry struct {
	Content       string
	UndoCursorPos CursorPos
	RedoCursorPos CursorPos
}

func NewHistory() History {
	history := History{
		index:    0,
		entries:  []Entry{},
		maxItems: 100,
	}

	return history
}

func (h *History) NewEntry(cursorPos CursorPos) {
	entryLen := len(h.entries)
	// if the current index is lower the the length of all entries
	// truncate the slice to the current index so the history doesn't
	// get too confusing
	if entryLen > 0 && h.EntryIndex < entryLen {
		diff := entryLen - h.EntryIndex - 1
		h.entries = h.entries[:entryLen-diff]
	}

	h.index++

	h.entries = append(h.entries, Entry{
		Content:       "",
		UndoCursorPos: cursorPos,
	})
	h.EntryIndex = len(h.entries) - 1
}

func (h *History) UpdateEntry(s string, cursorPos CursorPos) error {
	if len(h.entries) <= 0 {
		return errors.New("nope")
	}

	if len(h.entries) < h.EntryIndex {
		debug.LogErr("History entry index not found:", h.EntryIndex)
		return fmt.Errorf("History entry index %d not found", h.EntryIndex)
	}

	h.entries[h.EntryIndex].Content = s
	h.entries[h.EntryIndex].RedoCursorPos = cursorPos
	return nil
}

func (h *History) Undo() (string, CursorPos) {
	cursorPos := h.entries[h.EntryIndex].UndoCursorPos

	h.EntryIndex--
	if h.EntryIndex < 0 {
		h.EntryIndex = 0
	}

	entry := h.entries[h.EntryIndex]
	return entry.Content, cursorPos
}

func (h *History) Redo() (string, CursorPos) {
	h.EntryIndex++
	if h.EntryIndex >= len(h.entries) {
		h.EntryIndex = len(h.entries) - 1
	}

	entry := h.entries[h.EntryIndex]
	return entry.Content, entry.RedoCursorPos
}
