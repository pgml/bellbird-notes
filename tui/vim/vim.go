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

func (v Vim) Mode() *mode.ModeInstance {
	return v.app.Mode
}

func New() *Vim { return &Vim{} }

func (v *Vim) SetApp(app *application.App) {
	v.app = app
	v.KeyMap = nil
}
