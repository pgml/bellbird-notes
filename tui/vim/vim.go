package vim

import (
	"bellbird-notes/tui/components"
	"bellbird-notes/tui/keyinput"
	"bellbird-notes/tui/mode"
)

type Vim struct {
	KeyMap *keyinput.Input

	// app holds the state and behaviour of all core components
	app *components.App
}

func (v Vim) Mode() *mode.ModeInstance {
	return v.app.Mode
}

func New() *Vim { return &Vim{} }

func (v *Vim) SetApp(app *components.App) {
	v.app = app
	v.KeyMap = nil
}
