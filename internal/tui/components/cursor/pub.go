package cursor

import "github.com/charmbracelet/bubbles/cursor"

type Selection struct {
	Cursor cursor.Model
	start  int
	end    int
}

// SetWidth sets the character under the cursor.
func (m *Model) SetWidth(width int) {
	m.width = width
}

// SetWidth sets the character under the cursor.
func (m *Model) Width() int {
	return m.width
}

func (m *Model) UpdateStyle() {
	if m.Blink {
		m.TextStyle = m.TextStyle.Width(m.Width())
	} else {
		m.Style = m.Style.Width(m.Width())
	}
}
