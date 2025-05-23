package keyinput

import (
	"strings"
	"time"

	"bellbird-notes/tui"
	"bellbird-notes/tui/messages"
)

type keyAction struct {
	keys   string
	action string
	mode   tui.Mode
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
	Mode  tui.Mode

	Functions map[string]func() messages.StatusBarMsg
}

func New() *Input {
	return &Input{
		Ctrl:        false,
		Alt:         false,
		Shift:       false,
		Mode:        tui.NormalMode,
		KeySequence: make(map[string]bool),
		KeysDown:    make(map[string]bool),
		KeyMaps: []keyAction{
			{"ctrl+w l", "focusNextColumn", tui.NormalMode},
			{"ctrl+w h", "focusPrevColumn", tui.NormalMode},
			{"1", "focusDirectoryTree", tui.NormalMode},
			{"2", "focusNotesList", tui.NormalMode},
			{"3", "focusEditor", tui.NormalMode},
			{"e", "focusNextColumn", tui.NormalMode},
			{"q", "focusPrevColumn", tui.NormalMode},
			{"j", "lineDown", tui.NormalMode},
			{"k", "lineUp", tui.NormalMode},
			{"l", "expand", tui.NormalMode},
			{"h", "collapse", tui.NormalMode},
			{"R", "rename", tui.NormalMode},
			{"d", "createDir", tui.NormalMode},
			{"%", "createNote", tui.NormalMode},
			{"D", "delete", tui.NormalMode},
			{"g", "goToTop", tui.NormalMode},
			{"G", "goToBottom", tui.NormalMode},
			{"esc", "cancelAction", tui.NormalMode},
			{"esc", "cancelAction", tui.InsertMode},
			{"esc", "cancelAction", tui.CommandMode},
			{"enter", "confirmAction", tui.NormalMode},
			{"enter", "confirmAction", tui.InsertMode},
			{"enter", "confirmAction", tui.CommandMode},
			//{"i", "enterInsertMode", tui.NormalMode},
		},
	}
}

func (ki *Input) HandleSequences(key string) messages.StatusBarMsg {
	if key == "ctrl+w" {
		ki.Ctrl = true
	}

	if ki.Ctrl && strings.Contains(key, "ctrl+") {
		ki.KeysDown["ctrl+w"] = true
		key = strings.Split(key, "+")[1]
	}

	ki.KeysDown[key] = true
	//ki.keyComboCache[key] = true

	actionString := mapToActionString(ki.KeysDown)
	//if len(ki.keyComboCache) > 0 {
	//	actionString = mapToActionString(ki.keyComboCache)
	//}
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

	if ki.Mode != tui.CommandMode {
		ki.releaseKey(key)
	}

	return statusMsg
}

func (ki *Input) executeAction(keys string) messages.StatusBarMsg {
	for _, keyMap := range ki.KeyMaps {
		if keyMap.keys == keys && ki.Mode == keyMap.mode {
			if fn, exists := ki.Functions[keyMap.action]; exists {
				ki.ResetKeysDown()
				ki.resetSequenceCache()
				return fn()
			}
		}
	}
	return messages.StatusBarMsg{}
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
