// I need stuff that is not public to be public for the vim motions
// That's why this file exists
// Also I need need new textarea functions
package textarea

import (
	"image/color"
	"slices"
	"strings"
	"unicode"

	"github.com/charmbracelet/bubbles/v2/cursor"
	tea "github.com/charmbracelet/bubbletea/v2"
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

	Mode SelectionMode

	wrappedLline []rune
	lineIndex    int

	Content *string
}

type SelectionMode int

const (
	SelectNone SelectionMode = iota
	SelectVisual
	SelectVisualLine
	SelectVisualBlock
)

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

// same as m.wordLeft but checks for non-letters instead of just spaces
func (m *Model) WordLeft() {
	for {
		m.characterLeft(true /* insideLine */)
		if m.col < len(m.value[m.row]) && unicode.IsLetter(m.value[m.row][m.col]) {
			break
		}
	}

	for m.col > 0 {
		if !unicode.IsLetter(m.value[m.row][m.col-1]) {
			break
		}
		m.SetCursorColumn(m.col - 1)
	}

	m.repositionView()
}

// WordRight moves the cursor to the start of the next word.
// Skips any non-letter characters that follow.
func (m *Model) WordRight() {
	m.col = clamp(m.col, 0, len(m.value[m.row])-1)

	if len(m.value[m.row]) == 0 {
		m.MoveCursor(m.row+1, 0)
		m.repositionView()
		return
	}

	for {
		m.characterRight()

		if m.col >= len(m.value[m.row]) {
			m.MoveCursor(m.row+1, 0)
			break
		}

		if !unicode.IsLetter(m.value[m.row][m.col]) {
			m.CharacterRight(false)
			break
		}
	}

	m.repositionView()
}

// WordRightEnd moves the cursor to the end of the next word.
func (m *Model) WordRightEnd() {
	if m.col >= len(m.value[m.row])-1 {
		m.MoveCursor(m.row+1, 0)
	}

	for {
		m.characterRight()

		if m.col+1 >= len(m.value[m.row]) {
			break
		}

		if !unicode.IsLetter(m.value[m.row][m.col+1]) {
			break
		}
	}

	m.repositionView()
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
func (m *Model) DeleteAfterCursor(overshoot bool) {
	m.deleteAfterCursor()
	if !overshoot {
		m.SetCursorColumn(len(m.value[m.row]) - 1)
	}
}

///
/// custom methods
///

func (m *Model) RenderLine(
	line *[]rune,
	wrappedLine *[]rune,
	l int,
	wl int,
	s *strings.Builder,
	style lipgloss.Style,
) {
	lineInfo := m.LineInfo()

	wrLine := *wrappedLine
	if m.row == l && lineInfo.RowOffset == wl {
		s.WriteString(style.Render(string(wrLine[:lineInfo.ColumnOffset])))
		if m.col >= len(*line) && lineInfo.CharOffset >= m.width {
			m.virtualCursor.SetChar(" ")
			s.WriteString(m.virtualCursor.View())
		} else {
			m.virtualCursor.SetChar(string(wrLine[lineInfo.ColumnOffset]))
			s.WriteString(style.Render(m.virtualCursor.View()))
			s.WriteString(style.Render(string(wrLine[lineInfo.ColumnOffset+1:])))
		}
	} else {
		s.WriteString(style.Render(string(wrLine)))
	}
}

func (m *Model) LineLength(index int) int {
	if index == -1 {
		index = m.row
	}
	return len(m.value[index])
}

func (m *Model) CursorLineEnd() {
	m.SetCursorColumn(len(m.value[m.row]))
}

func (m *Model) CursorLineVimEnd() {
	m.SetCursorColumn(len(m.value[m.row]) - 1)
}

func (m *Model) IsExceedingLine() bool {
	maxRows := len(m.value) - 1
	if m.row > maxRows {
		m.row = maxRows
	}
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
	if row < len(m.value) {
		m.row = row
	}

	m.SetCursorColumn(col)
	// Any time that we move the cursor horizontally we need to reset the last
	// offset so that the horizontal position when navigating is adjusted.
	//m.lastCharOffset = 0
}

func (m *Model) CursorPos() CursorPos {
	return CursorPos{
		Row:          m.row,
		ColumnOffset: m.LineInfo().ColumnOffset,
	}
}

func (m *Model) SetCursorColor(color color.Color) {
	style := CursorStyle{
		Color:      color,
		Shape:      tea.CursorBlock,
		Blink:      false,
		BlinkSpeed: 0,
	}
	m.Styles.Cursor = style
	m.updateVirtualCursorStyle()
}

func (m *Model) EmptyLineAbove() {
	if m.row == 0 {
		// extend slice
		m.value = m.value[:len(m.value)+1]
		// add empty item at the beginning
		m.value = append([][]rune{{}}, m.value...)
		// move column offset internally to the beginning of the line
		m.SetCursorColumn(0)
	} else {
		m.CursorUp()
		m.EmptyLineBelow()
	}
	m.RepositionView()
}

func (m *Model) EmptyLineBelow() {
	m.value = slices.Insert(m.value, m.row+1, []rune{})
	m.CursorDown()
	m.RepositionView()
}

func (m *Model) DeleteOuterWord() {
	m.DeleteInnerWord()

	if m.col < len(m.value[m.row]) && unicode.IsSpace(m.value[m.row][m.col]) {
		m.value[m.row] = slices.Delete(m.value[m.row], m.col, m.col+1)
	}

	if m.col > 0 {
		m.characterLeft(false)
		m.SetCursorColumn(m.col + 1)
	}
}

func (m *Model) DeleteInnerWord() {
	col := m.value[m.row][m.col]

	// if the current character is space then just delete the space
	// and don't walk back
	if unicode.IsSpace(col) {
		m.value[m.row] = slices.Delete(m.value[m.row], m.col, m.col+1)
	} else {
		for {
			m.characterLeft(false)

			// break early if we're at the first word and don't move
			// to the previous row
			if m.col == 0 {
				break
			}

			// move left until we hit a space rune
			if m.col < len(m.value[m.row]) &&
				!unicode.IsLetter(m.value[m.row][m.col]) {

				// increment column offset so that the cursor
				// isn't at the position where the space rune was
				m.col++
				break
			}
		}

		m.SetCursorColumn(m.col)
		m.DeleteWordRight()
	}
}

// DeleteLine deletes current line
func (m *Model) DeleteLine() {
	if m.row >= len(m.value)-1 {
		m.value = m.value[:len(m.value)-1]
		m.row--
	} else {
		m.value = append(m.value[:m.row], m.value[m.row+1:]...)
	}
}

// DeleteLines deletes l lines
func (m *Model) DeleteLines(l int, up bool) {
	row := m.row
	if up {
		row -= l - 1
		m.CursorUp()
	}
	for range l {
		m.DeleteLine()
	}
}

func (m *Model) DeleteSelectedLines() {
	minRange, maxRange := m.SelectionRange()
	for i := minRange.Row; i <= maxRange.Row; i++ {
		// If there's only one line left don't delete this line, instead
		// we just empty it
		if len(m.value) == 1 {
			m.value[0] = slices.Delete(m.value[0], 0, len(m.value[m.row]))
			break
		}
		m.value = slices.Delete(m.value, m.row, m.row+1)
		m.row = minRange.Row
	}
}

// deleteWordRight deletes the word right to the cursor.
// In contrast to m.deleteWordRight this method separates by non-letters
// instead of space
func (m *Model) DeleteWordRight() {
	if m.col >= len(m.value[m.row]) || len(m.value[m.row]) == 0 {
		return
	}

	oldCol := m.col

	for m.col < len(m.value[m.row]) && !unicode.IsLetter(m.value[m.row][m.col]) {
		// ignore series of whitespace after cursor
		m.SetCursorColumn(m.col + 1)
	}

	for m.col < len(m.value[m.row]) {
		if unicode.IsLetter(m.value[m.row][m.col]) {
			m.SetCursorColumn(m.col + 1)
		} else {
			break
		}
	}

	if m.col > len(m.value[m.row]) {
		m.value[m.row] = m.value[m.row][:oldCol]
	} else {
		m.value[m.row] = append(m.value[m.row][:oldCol], m.value[m.row][m.col:]...)
	}

	m.SetCursorColumn(oldCol)
	//m.deleteWordRight()
}

func (m *Model) VimMergeLineBelow(row int) {
	m.CursorLineEnd()
	m.InsertRune(' ')
	m.SetCursorColumn(m.LineInfo().ColumnOffset - 1)
	m.mergeLineBelow(row)
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

// ReplaceRune replaces the rune the cursor is currently
// on with `replaceRune` rune
func (m *Model) ReplaceRune(replaceWith rune) {
	m.value[m.row][m.col] = replaceWith
}

// FirstVisibleLine returns the first line of the viewport
func (m *Model) FirstVisibleLine() int {
	return m.viewport.YOffset
}

func (p CursorPos) GreaterThan(other CursorPos) bool {
	return p.Row > other.Row || (p.Row == other.Row && p.ColumnOffset > other.ColumnOffset)
}

func (m *Model) SelectionStr() string {
	minRange, maxRange := m.SelectionRange()
	minRow, maxRow := minRange.Row, maxRange.Row

	if minRow < 0 {
		return ""
	}

	var str strings.Builder
	minCol := minRange.ColumnOffset
	maxCol := maxRange.ColumnOffset

	// select whole lines in range in visual line mode
	if m.Selection.Mode == SelectVisualLine {
		for i := minRow; i <= maxRow; i++ {
			str.WriteString(string(m.value[i]))
			str.WriteRune('\n')
		}
	} else {
		if minRow == maxRow {
			line := string(m.value[minRow])
			// selection on the same line
			if minCol <= maxCol && maxCol < len(m.value[minRow]) {
				str.WriteString(line[minCol : maxCol+1])
			}
		} else {
			// get the selected part of the first line
			if minCol <= len(m.value[minRow]) {
				line := string(m.value[minRow])
				// handles backward selection (if the selection starts at a lower
				// line and ends on a higher line)
				if m.row < maxRow && minCol > 0 {
					minCol -= 1
				}
				str.WriteString(line[minCol:])
				str.WriteRune('\n')
			}

			// get any fully selected lines in between
			if maxRow > minRow+1 {
				for i := minRow + 1; i < maxRow; i++ {
					str.WriteString(string(m.value[i]))
					str.WriteRune('\n')
				}
			}

			// get the selection of the last line
			if maxCol+1 < len(m.value[maxRow]) {
				line := string(m.value[maxRow])
				str.WriteString(line[:maxCol+1])
			}
		}
	}

	content := str.String()
	m.Selection.Content = &content
	return str.String()
}

// DeleteRune deletes the rune at `col` on `row`.
func (m *Model) DeleteRune(row int, col int) string {
	deletedChar := ""
	if col+1 <= len(m.value[row]) {
		deletedChar = string(m.value[row][col])
		m.value[row] = slices.Delete(m.value[row], col, col+1)
	}
	return deletedChar
}

// DeleteRunesInRange deletes all runes from the buffer between
// minRange and maxRange.
func (m *Model) DeleteRunesInRange(minRange CursorPos, maxRange CursorPos) {
	minRow, maxRow := minRange.Row, maxRange.Row

	if minRow < 0 {
		return
	}

	val := m.value
	minCol := minRange.ColumnOffset
	maxCol := maxRange.ColumnOffset

	// ensure nothing is out of bounds
	if minRow >= len(val) || maxRow >= len(val) {
		return
	}

	if minCol > len(val[minRow]) {
		minCol = len(val[minRow])
	}

	if maxCol > len(val[maxRow]) {
		maxCol = len(val[maxRow])
	}

	if minRow == maxRow {
		// selection on the same line
		if minCol <= maxCol && maxCol <= len(val[minRow]) {
			val[minRow] = slices.Delete(val[minRow], minCol, maxCol+1)
		}
	} else {
		// multi line selection

		// trim trailing runes from the first line
		if minCol <= len(val[minRow]) {
			// handles backward selection (if the selection starts at a lower
			// line and ends on a higher line)
			if m.row < maxRow {
				minCol -= 1
			}
			val[minRow] = val[minRow][:minCol]
		}

		// trim trailing runes from the last line
		if maxCol <= len(val[maxRow]) {
			val[maxRow] = val[maxRow][maxCol+1:]
		}

		// remove any fully selected lines in between
		if maxRow > minRow+1 {
			val = slices.Delete(val, minRow+1, maxRow)
		}

		// merge first and last line
		if len(val) > minRow+1 {
			m.mergeLineBelow(minRow)
		}
	}

	m.value = val
	m.row = minRow
	m.SetCursorColumn(minCol)
	m.ResetSelection()
}

// StartSelection prepares a selection
func (m *Model) StartSelection(selectionMode SelectionMode) {
	m.Selection.Cursor.Focus()
	if m.Selection.StartRow < 0 {
		m.Selection.StartRow = m.row
		m.Selection.StartCol = m.LineInfo().ColumnOffset
	}
	m.Selection.Mode = selectionMode
}

func (m *Model) SelectRange(
	selectionMode SelectionMode,
	from CursorPos,
	to CursorPos,
) string {
	m.Selection.Mode = selectionMode
	m.Selection.StartRow = from.Row
	m.Selection.StartCol = from.ColumnOffset
	m.MoveCursor(to.Row, to.ColumnOffset)

	return m.SelectionStr()
	//m.Selection.Content = &content
}

func (m *Model) SelectInnerWord() {
	// if the current character is space then just enter visual mode
	if unicode.IsSpace(m.value[m.row][m.col]) {
		m.StartSelection(SelectVisual)
		return
	}

	m.col = clamp(m.col, 0, len(m.value[m.row])-1)

	if m.col > 0 {
		for {
			m.characterLeft(false)
			// break early if we're at the first word and don't move
			// to the previous row
			if m.col == 0 {
				break
			}
			// move left until we hit a space rune
			if m.col >= 0 && !unicode.IsLetter(m.value[m.row][m.col]) {
				// increment column offset so that the cursor
				// isn't at the position where the space rune was
				m.col++
				break
			}
		}
	}

	// start selection at the first character of the word
	m.Selection.StartRow = m.row
	m.Selection.StartCol = m.col
	m.Selection.Mode = SelectVisual

	// move right until we find a space and break
	for {
		m.characterRight()
		if m.col == len(m.value[m.row])-1 {
			break
		}
		if !unicode.IsLetter(m.value[m.row][m.col+1]) {
			break
		}
	}
}

func (m *Model) SelectOuterWord() {
	m.SelectInnerWord()
	//m.SetCursorColumn(m.col + 1)
	m.col = clamp(m.col+1, 0, len(m.value[m.row])-1)
}

// SelectionRange determines the range of the active selection
func (m *Model) SelectionRange() (CursorPos, CursorPos) {
	selectionStart := CursorPos{
		m.Selection.StartRow,
		m.Selection.StartCol,
	}

	// current cursor position which usually indicates the end of the selection
	cursor := CursorPos{
		m.row,
		m.LineInfo().ColumnOffset,
	}

	// if it's a backward selection ensure the first CursorPos is always lower
	if selectionStart.GreaterThan(cursor) {
		return cursor, selectionStart
	}

	return selectionStart, cursor
}

// InRange checks whether the current row is between `minPos` and `maxPos`
func (p CursorPos) InRange(minPos, maxPos CursorPos) bool {
	if minPos.ColumnOffset == -1 || minPos.Row == -1 {
		return false
	}

	minColOffset := min(minPos.ColumnOffset, maxPos.ColumnOffset)
	maxColOffset := max(minPos.ColumnOffset, maxPos.ColumnOffset)

	return p.Row >= minPos.Row && p.Row <= maxPos.Row &&
		p.ColumnOffset >= minColOffset && p.ColumnOffset <= maxColOffset
}

func (m *Model) SelectionStyle() lipgloss.Style {
	return m.activeStyle().computedCursorLine()
}

// ResetSelection clears a selection
func (m *Model) ResetSelection() {
	m.Selection.StartRow = -1
	m.Selection.StartCol = -1
	m.Selection.Mode = SelectNone
}

// SelectionContent returns the buffer content within the current selection
// range, along with the unselected text before and after it.
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
		if m.Selection.Mode == SelectVisualLine {
			before = ""
			if l >= minRange.Row && l <= maxRange.Row {
				selection = string(runes)
			}
			//if m.row >= minRange.Row && m.row <= maxRange.Row {
			//	selection = wrappedStr
			//}
			after = ""
		} else {
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

				if beforePos <= lineLen && beforePos >= 0 {
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
	}

	return SelectionContent{
		Before:  before,
		Content: selection,
		After:   after,
	}
}

// CursorBeforeSelection returns the cursor that is at the beginning
// of a selection as a string.
// Returns an empty string of the selection doesn't require a
// cursor at the beginning (e.g. a forward selection)
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
		} else if lineIndex < m.Selection.StartRow && cursorOffset-1 >= 0 {
			m.virtualCursor.SetChar(string(wrappedLine[cursorOffset-1]))
		}
		return m.virtualCursor.View()
	}

	return ""
}

// CursorAfterSelection returns the cursor that is at the end
// of a selection as a string.
// Returns an empty string of the selection doesn't require a
// cursor at the beginning (e.g. a backward selection)
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
