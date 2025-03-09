// I need stuff that is not public to be public for the vim motions
// That's why this file exists
// Also I need need new textarea functions

package textarea

import "slices"

type CursorPos struct {
	Row          int
	ColumnOffset int
}

// characterLeft moves the cursor one character to the left.
// If insideLine is set, the cursor is moved to the last
// character in the previous line, instead of one past that.
func (m *Model) CharacterLeft(inside bool) {
	m.characterLeft(inside)
}

// characterRight moves the cursor one character to the right.
func (m *Model) CharacterRight() {
	m.characterRight()
}

// repositionView repositions the view of the viewport based on the defined
// scrolling behavior.
func (m *Model) RepositionView() {
	m.repositionView()
}

// wordLeft moves the cursor one word to the left. Returns whether or not the
// cursor blink should be reset. If input is masked, move input to the start
// so as not to reveal word breaks in the masked input.
func (m *Model) WordLeft() {
	m.wordLeft()
}

// wordRight moves the cursor one word to the right. Returns whether or not the
// cursor blink should be reset. If the input is masked, move input to the end
// so as not to reveal word breaks in the masked input.
func (m *Model) WordRight() {
	m.wordRight()
}

// CursorStart moves the cursor to the first non-blank character of the line
func (m *Model) CursorInputStart() {
	for i, r := range m.value[m.row] {
		if r != 32 {
			m.SetCursor(i)
			break
		}
	}
}

// moveToBegin moves the cursor to the beginning of the input.
func (m *Model) MoveToBegin() {
	m.moveToBegin()
}

// moveToEnd moves the cursor to the end of the input.
func (m *Model) MoveToEnd() {
	m.moveToEnd()
}

// deleteAfterCursor deletes all text after the cursor. Returns whether or not
// the cursor blink should be reset. If input is masked delete everything after
// the cursor so as not to reveal word breaks in the masked input.
func (m *Model) DeleteAfterCursor() {
	m.deleteAfterCursor()
	m.SetCursor(len(m.value[m.row]) - 1)
}

///
/// custom methods
///

// SetCursor moves the cursor to the given position. If the position is
// out of bounds the cursor will be moved to the start or end accordingly.
func (m *Model) MoveCursor(row int, col int) {
	m.col = clamp(col, 0, len(m.value[m.row]))
	m.row = clamp(row, 0, len(m.value[m.col]))
	// Any time that we move the cursor horizontally we need to reset the last
	// offset so that the horizontal position when navigating is adjusted.
	//m.lastCharOffset = 0
}

func (m Model) CursorPos() CursorPos {
	return CursorPos{
		Row:          m.row,
		ColumnOffset: m.LineInfo().ColumnOffset,
	}
}

// DeleteLine deletes current line
func (m *Model) DeleteLine() {
	m.value = slices.Delete(m.value, m.row, m.row+1)
}

// DeleteLines deletes l lines
func (m *Model) DeleteLines(l int, up bool) {
	row := m.row
	if up {
		row -= l - 1
		m.CursorUp()
	}
	for range l {
		m.value = slices.Delete(m.value, row, row+1)
	}
}

// DownHalfPage move cursor and screen down 1/2 page
func (m *Model) DownHalfPage() {
	for range m.viewport.Height / 2 {
		m.CursorDown()
	}

	min := m.viewport.YOffset
	max := min + m.viewport.Height - 1

	if row := m.cursorLineNumber(); row > max {
		m.viewport.LineDown(m.viewport.Height / 2)
	}
}

// UpHalfPage move cursor and screen down 1/2 page
func (m *Model) UpHalfPage() {
	for range m.viewport.Height / 2 {
		m.CursorUp()
	}

	min := m.viewport.YOffset

	if row := m.cursorLineNumber(); row < min {
		m.viewport.LineUp(min - row)
	}
}
