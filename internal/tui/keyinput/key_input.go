package keyinput

import (
	"bellbird-notes/internal/app"
	"bellbird-notes/internal/tui/messages"
	"strings"
	"time"
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
			keyAction{"ctrl+w l", "focusNextColumn", app.NormalMode},
			keyAction{"ctrl+w h", "focusPrevColumn", app.NormalMode},
			keyAction{"1", "focusDirectoryTree", app.NormalMode},
			keyAction{"2", "focusNotesList", app.NormalMode},
			keyAction{"3", "focusEditor", app.NormalMode},
			keyAction{"e", "focusNextColumn", app.NormalMode},
			keyAction{"q", "focusPrevColumn", app.NormalMode},
			keyAction{"j", "lineDown", app.NormalMode},
			keyAction{"k", "lineUp", app.NormalMode},
			keyAction{"l", "expand", app.NormalMode},
			keyAction{"h", "collapse", app.NormalMode},
			keyAction{"R", "rename", app.NormalMode},
			keyAction{"d", "createDir", app.NormalMode},
			keyAction{"%", "createNote", app.NormalMode},
			keyAction{"D", "delete", app.NormalMode},
			keyAction{"g", "goToTop", app.NormalMode},
			keyAction{"G", "goToBottom", app.NormalMode},
			keyAction{"esc", "cancelAction", app.NormalMode},
			keyAction{"esc", "cancelAction", app.InsertMode},
			keyAction{"esc", "cancelAction", app.CommandMode},
			keyAction{"enter", "confirmAction", app.NormalMode},
			keyAction{"enter", "confirmAction", app.InsertMode},
			keyAction{"enter", "confirmAction", app.CommandMode},
			//keyAction{"i", "enterInsertMode", app.NormalMode},
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
