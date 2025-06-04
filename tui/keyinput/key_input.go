package keyinput

import (
	"strings"
	"time"

	"bellbird-notes/tui/message"
	"bellbird-notes/tui/mode"
)

type keyAction struct {
	keys   string
	action string
	mode   mode.Mode
}

type keyMap struct {
	action map[string]string
	keys   []string
	//triggered bool
}

type Input struct {
	KeySequence map[string]bool
	KeysDown    map[string]bool
	KeyMaps     []keyAction

	Ctrl  bool
	Alt   bool
	Shift bool
	Mode  mode.Mode

	Functions map[string]func() message.StatusBarMsg
}

func New() *Input {
	return &Input{
		Ctrl:        false,
		Alt:         false,
		Shift:       false,
		Mode:        mode.Normal,
		KeySequence: make(map[string]bool),
		KeysDown:    make(map[string]bool),
		KeyMaps: []keyAction{
			{"ctrl+w l", "focusNextColumn", mode.Normal},
			{"ctrl+w h", "focusPrevColumn", mode.Normal},
			{"1", "focusDirectoryTree", mode.Normal},
			{"2", "focusNotesList", mode.Normal},
			{"3", "focusEditor", mode.Normal},
			{"e", "focusNextColumn", mode.Normal},
			{"q", "focusPrevColumn", mode.Normal},
			{"j", "lineDown", mode.Normal},
			{"k", "lineUp", mode.Normal},
			{"l", "expand", mode.Normal},
			{"h", "collapse", mode.Normal},
			{"R", "rename", mode.Normal},
			{"d", "createDir", mode.Normal},
			{"%", "createNote", mode.Normal},
			{"D", "delete", mode.Normal},
			{"g", "goToTop", mode.Normal},
			{"G", "goToBottom", mode.Normal},
			//{"esc", "cancelAction", mode.Normal},
			{"esc", "cancelAction", mode.Insert},
			{"esc", "cancelAction", mode.Command},
			{"enter", "confirmAction", mode.Normal},
			{"enter", "confirmAction", mode.Insert},
			{"enter", "confirmAction", mode.Command},
			//{"i", "enterInsertMode", mode.NormalMode},
		},
	}
}

func (ki *Input) HandleSequences(key string) message.StatusBarMsg {
	if key == "ctrl+w" {
		ki.Ctrl = true
	}

	if ki.Ctrl && strings.Contains(key, "ctrl+") {
		ki.KeysDown["ctrl+w"] = true
		key = strings.Split(key, "+")[1]
	}

	ki.KeysDown[key] = true

	actionString := mapToActionString(ki.KeysDown)
	statusMsg := ki.executeAction(actionString)

	// special key actions for cmd mode
	switch key {
	case ":":
		//m.enterCmdMode()
	case "esc":
		//m.mode = mode.Normal
		//m.exitCmdMode()
	case "enter":
		//m.executeCmdModeCommand()
	}
	//if key == ":" {
	//	m.enterCmdMode()
	//}
	//if key == "esc" {
	//	m.exitCmdMode()
	//}
	//if key == "enter" {
	//	m.executeCmdModeCommand()
	//}

	if ki.Mode != mode.Command {
		ki.releaseKey(key)
	}

	return statusMsg
}

func (ki *Input) executeAction(keys string) message.StatusBarMsg {
	for _, keyMap := range ki.KeyMaps {
		if keyMap.keys == keys && ki.Mode == keyMap.mode {
			if fn, exists := ki.Functions[keyMap.action]; exists {
				ki.ResetKeysDown()
				ki.resetSequenceCache()
				return fn()
			}
		}
	}
	return message.StatusBarMsg{}
}

func (ki *Input) ResetKeysDown() {
	ki.Ctrl = false
	ki.KeysDown = make(map[string]bool)
}

func (ki *Input) resetSequenceCache() {
	ki.Ctrl = false
	ki.KeySequence = make(map[string]bool)
}

// simulate keyUp event
func (ki *Input) releaseKey(key string) {
	var timeout time.Duration = 50
	go func() {
		time.Sleep(timeout * time.Millisecond)
		delete(ki.KeysDown, key)
	}()
}

func (ki Input) GetKeysDown() []string {
	keys := []string{}
	for key := range ki.KeysDown {
		keys = append(keys, key)
	}
	return keys
}

func mapToActionString(keyMap map[string]bool) string {
	action := ""
	for keys := range keyMap {
		action += keys + " "
	}
	return strings.TrimSpace(action)
}
