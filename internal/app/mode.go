package app

type Mode int

const (
	NormalMode Mode = iota
	InsertMode
	VisualMode
	VisualLineMode
	VisualBlockMode
	OperatorMode
	CommandMode
)

var modeName = map[Mode]string{
	NormalMode:      "n",
	InsertMode:      "i",
	VisualMode:      "v",
	VisualLineMode:  "vi",
	VisualBlockMode: "vb",
	OperatorMode:    "o",
	CommandMode:     "c",
}

func (m Mode) String() string {
	return modeName[m]
}

type ModeInstance struct {
	Current Mode
}

func (m ModeInstance) GetCurrent() Mode {
	return m.Current
}
