package app

type Mode int

const (
	NormalMode Mode = iota
	InsertMode
	CommandMode
)

var modeName = map[Mode]string{
	NormalMode:  "n",
	InsertMode:  "i",
	CommandMode: "c",
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
