package keyinput

import (
	"strings"
	"time"

	"bellbird-notes/internal/app"
	"bellbird-notes/internal/tui/messages"
)

type keyAction struct {
	keys   string
	action string
	mode   app.Mode
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
	Mode  app.Mode

	Functions map[string]func() messages.StatusBarMsg
}

func New() *Input {
	return &Input{
		Ctrl:        false,
		Alt:         false,
		Shift:       false,
		Mode:        app.NormalMode,
		KeySequence: make(map[string]bool),
		KeysDown:    make(map[string]bool),
		KeyMaps: []keyAction{
			{"ctrl+w l", "focusNextColumn", app.NormalMode},
			{"ctrl+w h", "focusPrevColumn", app.NormalMode},
			{"1", "focusDirectoryTree", app.NormalMode},
			{"2", "focusNotesList", app.NormalMode},
			{"3", "focusEditor", app.NormalMode},
			{"e", "focusNextColumn", app.NormalMode},
			{"q", "focusPrevColumn", app.NormalMode},
			{"j", "lineDown", app.NormalMode},
			{"k", "lineUp", app.NormalMode},
			{"l", "expand", app.NormalMode},
			{"h", "collapse", app.NormalMode},
			{"R", "rename", app.NormalMode},
			{"d", "createDir", app.NormalMode},
			{"%", "createNote", app.NormalMode},
			{"D", "delete", app.NormalMode},
			{"g", "goToTop", app.NormalMode},
			{"G", "goToBottom", app.NormalMode},
			{"esc", "cancelAction", app.NormalMode},
			{"esc", "cancelAction", app.InsertMode},
			{"esc", "cancelAction", app.CommandMode},
			{"enter", "confirmAction", app.NormalMode},
			{"enter", "confirmAction", app.InsertMode},
			{"enter", "confirmAction", app.CommandMode},
			//{"i", "enterInsertMode", app.NormalMode},
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

	if ki.Mode != app.CommandMode {
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
