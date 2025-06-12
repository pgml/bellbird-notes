package cursor

import "github.com/charmbracelet/bubbles/cursor"

type Selection struct {
	Cursor cursor.Model
	start  int
	end    int
}
