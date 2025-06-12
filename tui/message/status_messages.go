package message

var Response = struct {
	Yes, No string
}{
	Yes: "y",
	No:  "n",
}

var StatusBar = struct {
	CmdPrompt, RemovePromptDirContent, RemovePrompt, NoteExists,
	CtrlCExitNote, FileWritten string
}{
	RemovePromptDirContent: "Delete `%s` and all of its content? [y(es),n(o)]",
	RemovePrompt:           "Delete `%s`? [y(es),n(o)]",
	NoteExists:             "Note already exists",
	CtrlCExitNote:          "Type :q and press <Enter> to quit",
	FileWritten:            "\"%s\" %dL, %dB written",
}

//var msgColours = map[MsgType]lipgloss.TerminalColor{
//	Success:     lipgloss.NoColor{},
//	Error:       lipgloss.Color("#d75a7d"),
//	Prompt:      lipgloss.NoColor{},
//	PromptError: lipgloss.Color("#d75a7d"),
//}
