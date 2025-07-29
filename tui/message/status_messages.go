package message

var CmdPrompt = struct {
	Yes, No, Quit, WriteBuf, WriteQuit, DeleteBufstring, ListBufs,
	Set, Open, New, Reload, CheckTime string
}{
	Yes:             "y",
	No:              "n",
	Quit:            "q",
	WriteBuf:        "w",
	WriteQuit:       "wq",
	DeleteBufstring: "bd",
	ListBufs:        "b",
	Set:             "set",
	Open:            "open",
	New:             "new",
	Reload:          "reload",
	CheckTime:       "checktime",
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
