package textarea

import (
	"fmt"

	"bellbird-notes/app/debug"

	dmp "github.com/sergi/go-diff/diffmatchpatch"
)

// History manages a list of text changes (undo/redo history)
type History struct {
	// EntryIndex is the current index in the history
	EntryIndex int

	// entries holds all recorded undo/redo history entries.
	entries []Entry

	tmpEntry Entry

	// maxItems is the maximum number of entries allowed in history
	maxItems uint

	Dmp *dmp.DiffMatchPatch
}

// Entry represents a single change in the undo/redo history.
type Entry struct {
	redoPatch     string
	undoPatch     string
	UndoCursorPos CursorPos
	RedoCursorPos CursorPos

	// hash of the content
	hash string
}

// Hash returns the hash of the entry.
func (e *Entry) Hash() string {
	return e.hash
}

// NewHistory returns a new initialized History.
func NewHistory() History {
	history := History{
		entries:    []Entry{},
		tmpEntry:   Entry{},
		maxItems:   100,
		Dmp:        dmp.New(),
		EntryIndex: -1,
	}

	return history
}

func (h *History) NewTmpEntry(cursorPos CursorPos) {
	h.tmpEntry = Entry{UndoCursorPos: cursorPos}
}

// NewEntry creates a new history entry.
// If future entries exist (after undo), they are discarded.
func (h *History) NewEntry(cursorPos CursorPos) {
	// if the current index is lower the the length of all entries
	// truncate the slice to the current index so the history doesn't
	// get too confusing
	if h.EntryIndex < len(h.entries) {
		h.entries = h.entries[:h.EntryIndex+1]
	}

	h.entries = append(h.entries, Entry{
		UndoCursorPos: cursorPos,
	})

	h.EntryIndex = len(h.entries) - 1
}

func (h *History) AppendTmpEntry() {
	h.NewEntry(h.tmpEntry.UndoCursorPos)
	h.tmpEntry = Entry{}
}

// UpdateEntry updates the current entry with patches and metadata.
func (h *History) UpdateEntry(
	redoPatch []dmp.Patch,
	undoPatch []dmp.Patch,
	cursorPos CursorPos,
	hash string,
) error {
	h.AppendTmpEntry()

	if h.EntryIndex >= len(h.entries) || h.EntryIndex < 0 {
		debug.LogErr("History entry index not found:", h.EntryIndex)
		return fmt.Errorf("History entry index %d not found", h.EntryIndex)
	}

	h.entries[h.EntryIndex].redoPatch = h.Dmp.PatchToText(redoPatch)
	h.entries[h.EntryIndex].undoPatch = h.Dmp.PatchToText(undoPatch)
	h.entries[h.EntryIndex].RedoCursorPos = cursorPos
	h.entries[h.EntryIndex].hash = hash

	return nil
}

// Entry returns the entry at the given index or nil if out of bounds.
func (h *History) Entry(index int) *Entry {
	if index > len(h.entries)-1 {
		return nil
	}
	return &h.entries[index]
}

// MakePatch generates a diff patch between oldStr and newStr.
func (h *History) MakePatch(oldStr string, newStr string) []dmp.Patch {
	return h.Dmp.PatchMake(oldStr, newStr)
}

// Undo returns the undo patch, content hash, and cursor position.
// If no undo is available, returns empty values.
func (h *History) Undo() ([]dmp.Patch, string, CursorPos) {
	if h.EntryIndex < 0 || h.EntryIndex >= len(h.entries) {
		return nil, "", CursorPos{}
	}

	if h.EntryIndex >= len(h.entries) {
		h.EntryIndex = len(h.entries) - 1
	}

	entry := h.entries[h.EntryIndex]
	cursorPos := h.entries[h.EntryIndex].UndoCursorPos
	patch, _ := h.Dmp.PatchFromText(entry.undoPatch)

	h.EntryIndex--

	return patch, entry.hash, cursorPos
}

// Redo returns the redo patch, content hash, and cursor position.
// If no redo is available, returns empty values.
func (h *History) Redo() ([]dmp.Patch, string, CursorPos) {
	if h.EntryIndex+1 >= len(h.entries) {
		return nil, "", CursorPos{}
	}

	h.EntryIndex++
	entry := h.entries[h.EntryIndex]

	patch, _ := h.Dmp.PatchFromText(entry.redoPatch)
	return patch, entry.hash, entry.RedoCursorPos
}
