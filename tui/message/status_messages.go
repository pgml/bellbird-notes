package message

var Response = struct {
	Yes, No string
}{
	Yes: "y",
	No:  "n",
}

const (
	RemovePromptContent string = "Delete `%s` and all of its content? [y(es),n(o)]:"
	RemovePrompt        string = "Delete `%s`? [y(es),n(o)]:"
	NoteExists          string = "Note already exists"
)

//var msgColours = map[MsgType]lipgloss.TerminalColor{
//	Success:     lipgloss.NoColor{},
//	Error:       lipgloss.Color("#d75a7d"),
//	Prompt:      lipgloss.NoColor{},
//	PromptError: lipgloss.Color("#d75a7d"),
//}
