// I need stuff that is not public to be public for the vim motions
// That's why this file exists
// Also I need need new textarea functions
package textarea

import (
	"bellbird-notes/app/debug"
	"slices"

	"github.com/charmbracelet/bubbles/v2/cursor"
	"github.com/charmbracelet/lipgloss/v2"
)

type CursorPos struct {
	Row          int
	ColumnOffset int
}

type Selection struct {
	Cursor   cursor.Model
	Start    CursorPos
	StartRow int
	StartCol int

	wrappedLline []rune
	lineIndex    int
}

type SelectionContent struct {
	Before  string
	Content string
	After   string
}

// characterLeft moves the cursor one character to the left.
// If insideLine is set, the cursor is moved to the last
// character in the previous line, instead of one past that.
func (m *Model) CharacterLeft(inside bool) {
	//m.characterLeft(inside)
	if m.col > 0 {
		m.SetCursorColumn(m.col - 1)
	}
}

// characterRight moves the cursor one character to the right.
//
// If overshoot is true, the cursor moves past the last character
// in the current row
func (m *Model) CharacterRight(overshoot bool) {
	if !overshoot {
		if m.col < len(m.value[m.row])-1 {
			m.SetCursorColumn(m.col + 1)
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
			m.SetCursorColumn(i)
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
	m.SetCursorColumn(len(m.value[m.row]) - 1)
}

///
/// custom methods
///

func (m *Model) CursorVimEnd() {
	m.SetCursorColumn(len(m.value[m.row]) - 1)
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
	m.SetCursorColumn(currCursorPos)
}

// DeleteLines deletes l lines
func (m *Model) DeleteLines(l int, up bool) {
	row := m.row
	if up {
		row -= l - 1
		m.CursorUp()
	}
	for range l {
		m.SetCursorColumn(l)
		m.DeleteLine()
	}
}

func (m *Model) DeleteWordRight() {
	m.deleteWordRight()
}

// DownHalfPage move cursor and screen down 1/2 page
func (m *Model) DownHalfPage() {
	for range m.viewport.Height() / 2 {
		m.CursorDown()
	}

	min := m.viewport.YOffset
	max := min + m.viewport.Height() - 1

	if row := m.cursorLineNumber(); row > max {
		m.viewport.LineDown(m.viewport.Height() / 2)
	}
}

// UpHalfPage move cursor and screen down 1/2 page
func (m *Model) UpHalfPage() {
	for range m.viewport.Height() / 2 {
		m.CursorUp()
	}

	min := m.viewport.YOffset

	if row := m.cursorLineNumber(); row < min {
		m.viewport.LineUp(min - row)
	}
}

func (m *Model) ReplaceRune(replaceWith rune) {
	m.value[m.row][m.col] = replaceWith
}

func (m *Model) DeleteRune(row int, col int) {
	sel := m.SelectionContent()
	if sel.Content != "" {
		debug.LogDebug(sel.Content)
		for i := range []rune(sel.Content) {
			debug.LogDebug(i)
			m.value[row] = slices.Delete(m.value[row], i, i+1)
		}

	} else {
		if col+1 <= len(m.value[row]) {
			m.value[row] = slices.Delete(m.value[row], col, col+1)
		}
	}
	//m.SetCursorColumn(0)
}

func (m *Model) FirstVisibleLine() int {
	return m.viewport.YOffset
}

func (m *Model) StartSelection() {
	m.Selection.Cursor.Focus()
	m.Selection.StartRow = m.row
	m.Selection.StartCol = m.LineInfo().ColumnOffset
}

func (m *Model) SelectionRange() (CursorPos, CursorPos) {
	selectionStart := CursorPos{
		m.Selection.StartRow,
		m.Selection.StartCol,
	}

	cursor := CursorPos{
		m.row,
		m.LineInfo().ColumnOffset,
	}

	if selectionStart.GreaterThan(cursor) {
		return cursor, selectionStart
	}

	return selectionStart, cursor
}

func (p CursorPos) GreaterThan(other CursorPos) bool {
	return p.Row > other.Row || (p.Row == other.Row && p.ColumnOffset > other.ColumnOffset)
}

func (p CursorPos) InRange(minPos, maxPos CursorPos) bool {
	if minPos.ColumnOffset == -1 || minPos.Row == -1 {
		return false
	}

	maxColOffset := max(minPos.ColumnOffset, maxPos.ColumnOffset)
	minColOffset := min(minPos.ColumnOffset, maxPos.ColumnOffset)

	return p.Row >= minPos.Row && p.Row <= maxPos.Row &&
		p.ColumnOffset >= minColOffset && p.ColumnOffset <= maxColOffset
}

func (m *Model) SelectionStyle() lipgloss.Style {
	return m.activeStyle().computedCursorLine()
}

func (m *Model) ResetSelection() {
	m.Selection.StartRow = -1
	m.Selection.StartCol = -1
}

func (m *Model) SelectionContent() SelectionContent {
	line := m.Selection.wrappedLline
	l := m.Selection.lineIndex

	colOffset := m.LineInfo().ColumnOffset
	minRange, maxRange := m.SelectionRange()
	cursor := CursorPos{m.row, colOffset}
	isInRange := cursor.InRange(minRange, maxRange)
	wrappedStr := string(line)

	var (
		before    string
		selection string
		after     string
	)

	cursorOffset := colOffset
	if minRange.ColumnOffset < m.Selection.StartCol {
		cursorOffset = minRange.ColumnOffset
	}

	isCursorBeforeSel := l == minRange.Row && cursorOffset < maxRange.ColumnOffset

	// slice for unicode safety
	runes := []rune(wrappedStr)
	lineLen := len(runes)

	if isInRange {
		minCol := clamp(minRange.ColumnOffset, 0, lineLen)
		maxCol := clamp(maxRange.ColumnOffset, 0, lineLen)
		minRow, maxRow := minRange.Row, maxRange.Row

		if colOffset == minCol {
			colOffset = maxCol
		}

		switch {
		// single line selection
		case minRow == l && maxRow == l:
			before = string(runes[:minCol])

			if isCursorBeforeSel {
				minCol = clamp(minCol+1, 0, lineLen)
				colOffset = clamp(m.Selection.StartCol+1, 0, lineLen)
			}

			if colOffset <= lineLen {
				selection = string(runes[minCol:colOffset])
			}

			if maxCol < lineLen {
				after = string(runes[maxCol+1:])
			}

		// first line of multi selection
		case minRow == l:
			beforePos := minCol

			if m.Selection.StartRow > minRow {
				if minCol < m.Selection.StartCol {
					minCol = clamp(minCol+1, 0, lineLen)
				} else {
					beforePos = minCol - 1
					cursorOffset = minCol
				}
			}

			if beforePos <= lineLen {
				before = string(runes[:beforePos])
			}

			if minCol <= lineLen {
				selection = string(runes[minCol:])
			}

		// last line of multi selection
		case maxRow == l:
			beforePos := clamp(maxCol+1, 0, lineLen)
			afterPos := maxCol

			if m.Selection.StartRow > minRow {
				afterPos = clamp(maxCol+1, 0, lineLen)
			}

			if afterPos <= lineLen {
				selection = string(runes[:afterPos])
			}

			if beforePos <= lineLen {
				after = string(runes[beforePos:])
			}

		// full line within selection
		case l > minRow && l < maxRow:
			selection = string(runes)
		}
	}

	return SelectionContent{
		Before:  before,
		Content: selection,
		After:   after,
	}
}

func (m *Model) CursorBeforeSelection() string {
	wrappedLine := m.Selection.wrappedLline
	lineIndex := m.Selection.lineIndex
	cursorOffset := m.LineInfo().ColumnOffset
	minRange, maxRange := m.SelectionRange()

	if lineIndex == minRange.Row &&
		maxRange.Row <= m.Selection.StartRow &&
		cursorOffset < len(wrappedLine) {

		if cursorOffset < maxRange.ColumnOffset {
			m.virtualCursor.SetChar(string(wrappedLine[cursorOffset]))
		} else if lineIndex < m.Selection.StartRow {
			m.virtualCursor.SetChar(string(wrappedLine[cursorOffset-1]))
		}
		return m.virtualCursor.View()
	}

	return ""
}

func (m *Model) CursorAfterSelection() string {
	wrappedLine := m.Selection.wrappedLline
	cursorOffset := m.LineInfo().ColumnOffset

	minRange, maxRange := m.SelectionRange()

	if m.Selection.lineIndex == maxRange.Row &&
		minRange.Row >= m.Selection.StartRow &&
		cursorOffset < len(wrappedLine) &&
		cursorOffset == maxRange.ColumnOffset {

		m.virtualCursor.SetChar(string(wrappedLine[cursorOffset]))
		return m.virtualCursor.View()
	}

	return ""
}
