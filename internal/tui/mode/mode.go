package mode

type Mode int

const (
	Normal Mode = iota
	Insert
	Command
)

var modeName = map[Mode]string{
	Normal:  "n",
	Insert:  "i",
	Command: "c",
}

func (m Mode) String() string {
	return modeName[m]
}

type ModeInstance struct {
	Current Mode
}

func New() *ModeInstance {
	return &ModeInstance{
		Current: Normal,
	}
}

func (m ModeInstance) GetCurrent() Mode {
	return m.Current
}
