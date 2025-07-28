package textarea

import (
	"sort"
	"strings"
)

type Search struct {
	Query      string
	IgnoreCase bool
	Matches    map[int][]int
}

// FirstMatch returns the first item of a search result
func (s Search) FirstMatch() CursorPos {
	for _, r := range s.sortedRows(false) {
		return CursorPos{
			Row:          r,
			RowOffset:    0,
			ColumnOffset: s.Matches[r][0],
		}
	}
	return CursorPos{}
}

// FindMatch returns the next or previous match from the current position.
// If reverse is false, it searches forward
// If reverse is true, it searches backward
func (s Search) FindMatch(current CursorPos, prev bool) (CursorPos, bool) {
	matches := s.Matches
	pos := CursorPos{}

	if len(matches) == 0 {
		return pos, false
	}

	rows := s.sortedRows(prev)

	// Helper: find next column in sorted slice depending
	// on matching direction
	findCol := func(cols []int, col int) (int, bool) {
		sort.Ints(cols)
		if prev {
			for i := len(cols) - 1; i >= 0; i-- {
				if cols[i] < col {
					return cols[i], true
				}
			}
		} else {
			for _, c := range cols {
				if c > col {
					return c, true
				}
			}
		}
		return 0, false
	}

	// Check current row for next column
	if cols, ok := matches[current.Row]; ok {
		if c, ok := findCol(cols, current.ColumnOffset); ok {
			return CursorPos{current.Row, 0, c}, true
		}
	}

	// Check rows > curRow for first column
	for _, r := range rows {
		if (prev && r < current.Row) || (!prev && r > current.Row) {
			cols := matches[r]
			sort.Ints(cols)

			if len(cols) > 0 {
				index := 0
				if prev {
					index = len(cols) - 1
				}
				return CursorPos{r, 0, cols[index]}, true
			}
		}
	}

	// Wrap around - return first column of smallest row
	firstRow := rows[0]
	cols := matches[firstRow]
	sort.Ints(cols)

	if len(cols) > 0 {
		index := 0
		if prev {
			index = len(cols) - 1
		}
		return CursorPos{firstRow, 0, cols[index]}, true
	}

	return CursorPos{}, false
}

// FindMatches returns all occurences of the current search query for
// the given row
func (s *Search) FindMatches(lineStr *string, row int) []int {
	var positions []int

	query := s.Query

	if query == "" {
		return nil
	}

	if s.IgnoreCase {
		query = strings.ToLower(query)
		*lineStr = strings.ToLower(*lineStr)
	}

	for i := 0; i <= len(*lineStr)-len(query); i++ {
		if (*lineStr)[i:i+len(query)] == query {
			positions = append(positions, i)
		}
	}

	return positions
}

// sortedRows returns a sorted slice of the s.Matches
func (s Search) sortedRows(reverse bool) []int {
	rows := make([]int, 0, len(s.Matches))

	for r := range s.Matches {
		rows = append(rows, r)
	}

	sort.Ints(rows)

	if reverse {
		// Reverse rows in place
		for i, j := 0, len(rows)-1; i < j; i, j = i+1, j-1 {
			rows[i], rows[j] = rows[j], rows[i]
		}
	}

	return rows
}

func (m *Model) FindNextMatch() {
	if pos, ok := m.Search.FindMatch(m.CursorPos(), false); ok {
		m.row = pos.Row
		m.SetCursorColumn(pos.ColumnOffset)
	}
}

func (m *Model) FindPrevMatch() {
	if pos, ok := m.Search.FindMatch(m.CursorPos(), true); ok {
		m.row = pos.Row
		m.SetCursorColumn(pos.ColumnOffset)
	}
}
