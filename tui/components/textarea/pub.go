// I need stuff that is not public to be public for the vim motions
// That's why this file exists
// Also I need need new textarea functions

package textarea

type CursorPos struct {
	Row          int
	ColumnOffset int
}

// characterLeft moves the cursor one character to the left.
// If insideLine is set, the cursor is moved to the last
// character in the previous line, instead of one past that.
func (m *Model) CharacterLeft(inside bool) {
	//m.characterLeft(inside)
	if m.col > 0 {
		m.SetCursor(m.col - 1)
	}
}

// characterRight moves the cursor one character to the right.
//
// If overshoot is true, the cursor moves past the last character
// in the current row
func (m *Model) CharacterRight(overshoot bool) {
	if !overshoot {
		if m.col < len(m.value[m.row])-1 {
			m.SetCursor(m.col + 1)
		}
	} else {
		m.characterRight()
	}
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

func (m *Model) CursorVimEnd() {
	m.SetCursor(len(m.value[m.row]) - 1)
}

func (m *Model) IsExceedingLine() bool {
	return m.col >= len(m.value[m.row])
}

func (m *Model) IsAtLineStart() bool {
	return m.col == 0
}

func (m *Model) IsAtLineEnd() bool {
	return m.col == len(m.value[m.row])-1
}

// SetCursor moves the cursor to the given position. If the position is
// out of bounds the cursor will be moved to the start or end accordingly.
func (m *Model) MoveCursor(row int, col int) {
	if len(m.value) > m.col {
		m.row = clamp(row, 0, len(m.value[m.col]))
	}
	if len(m.value) > m.row {
		m.col = clamp(col, 0, len(m.value[m.row]))
	}
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
	currCursorPos := m.LineInfo().ColumnOffset
	m.CursorStart()
	m.deleteAfterCursor()
	m.mergeLineBelow(m.row)
	m.SetCursor(currCursorPos)
}

// DeleteLines deletes l lines
func (m *Model) DeleteLines(l int, up bool) {
	row := m.row
	if up {
		row -= l - 1
		m.CursorUp()
	}
	for range l {
		m.SetCursor(l)
		m.DeleteLine()
	}
}

func (m *Model) DeleteWordRight() {
	m.deleteWordRight()
}

// DownHalfPage move cursor and screen down 1/2 page
func (m *Model) DownHalfPage() {
	for range m.viewport.Height / 2 {
		m.CursorDown()
	}

	min := m.viewport.YOffset
	max := min + m.viewport.Height - 1

	if row := m.cursorLineNumber(); row > max {
		m.viewport.ScrollDown(m.viewport.Height / 2)
	}
}

// UpHalfPage move cursor and screen down 1/2 page
func (m *Model) UpHalfPage() {
	for range m.viewport.Height / 2 {
		m.CursorUp()
	}

	min := m.viewport.YOffset

	if row := m.cursorLineNumber(); row < min {
		m.viewport.ScrollUp(min - row)
	}
}

func (m *Model) ReplaceRune(replaceWith rune) {
	m.value[m.row][m.col] = replaceWith
}

func (m *Model) FirstVisibleLine() int {
	return m.viewport.YOffset
}
