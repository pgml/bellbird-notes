package tui

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

type KeyInput struct {
	keyComboCache map[string]bool
	keysDown      map[string]bool
	keyMaps       []keyAction

	isCtrlWDown bool
	mode        app.Mode

	functions map[string]func() messages.StatusBarMsg
}

func NewKeyInput() *KeyInput {
	return &KeyInput{
		isCtrlWDown:   false,
		mode:          app.NormalMode,
		keyComboCache: make(map[string]bool),
		keysDown:      make(map[string]bool),
		keyMaps: []keyAction{
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
			keyAction{"esc", "exitInsertMode", app.InsertMode},
			keyAction{"esc", "cancelAction", app.CommandMode},
			keyAction{"enter", "confirmAction", app.NormalMode},
			keyAction{"enter", "confirmAction", app.InsertMode},
			keyAction{"enter", "confirmAction", app.CommandMode},
			keyAction{"i", "enterInsertMode", app.NormalMode},
		},
	}
}

func (ki *KeyInput) handleKeyCombos(key string) messages.StatusBarMsg {
	if key == "ctrl+w" {
		ki.isCtrlWDown = true
	}

	if ki.isCtrlWDown && strings.Contains(key, "ctrl+") {
		ki.keysDown["ctrl+w"] = true
		key = strings.Split(key, "+")[1]
	}

	ki.keysDown[key] = true
	//ki.keyComboCache[key] = true

	actionString := mapToActionString(ki.keysDown)
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

	if ki.mode != app.CommandMode {
		ki.releaseKey(key)
	}

	return statusMsg
}

func (ki *KeyInput) executeAction(keys string) messages.StatusBarMsg {
	for _, keyMap := range ki.keyMaps {
		if keyMap.keys == keys && ki.mode == keyMap.mode {
			if fn, exists := ki.functions[keyMap.action]; exists {
				ki.resetKeysDown()
				ki.resetKeysComboCache()
				return fn()
			}
		}
	}
	return messages.StatusBarMsg{}
}

func (ki *KeyInput) resetKeysDown() {
	ki.isCtrlWDown = false
	ki.keysDown = make(map[string]bool)
}

func (ki *KeyInput) resetKeysComboCache() {
	ki.isCtrlWDown = false
	ki.keyComboCache = make(map[string]bool)
}

// simulate keyUp event
func (ki *KeyInput) releaseKey(key string) {
	var timeout time.Duration = 50
	go func() {
		time.Sleep(timeout * time.Millisecond)
		delete(ki.keysDown, key)
	}()
}

func (ki KeyInput) GetKeysDown() []string {
	keys := []string{}
	for key := range ki.keysDown {
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
