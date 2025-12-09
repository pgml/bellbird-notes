package vim

import (
	"bellbird-notes/tui/components/application"
	"bellbird-notes/tui/keyinput"
	"bellbird-notes/tui/mode"
)

type Vim struct {
	KeyMap *keyinput.Input

	// app holds the state and behaviour of all core components
	app *application.App
}

func (vim Vim) Mode() *mode.ModeInstance {
	return vim.app.Mode
}

func New() *Vim { return &Vim{} }

func (vim *Vim) SetApp(app *application.App) {
	vim.app = app
	vim.KeyMap = nil
}
