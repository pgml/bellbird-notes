package tui

import (
	"bellbird-notes/internal/tui/messages"
	"bellbird-notes/internal/tui/mode"
	"strings"
	"time"
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

type KeyInput struct {
	keyComboCache map[string]bool
	keysDown      map[string]bool
	keyMaps       []keyAction

	isCtrlWDown bool
	mode        mode.Mode

	functions map[string]func() messages.StatusBarMsg
}

func NewKeyInput() *KeyInput {
	return &KeyInput{
		isCtrlWDown:   false,
		mode:          mode.Normal,
		keyComboCache: make(map[string]bool),
		keysDown:      make(map[string]bool),
		keyMaps: []keyAction{
			keyAction{"ctrl+w l", "focusNextColumn", mode.Normal},
			keyAction{"ctrl+w h", "focusPrevColumn", mode.Normal},
			keyAction{"j", "moveDown", mode.Normal},
			keyAction{"k", "moveUp", mode.Normal},
			keyAction{"l", "expand", mode.Normal},
			keyAction{"h", "collapse", mode.Normal},
			keyAction{"R", "rename", mode.Normal},
			keyAction{"d", "createDir", mode.Normal},
			keyAction{"%", "createNote", mode.Normal},
			keyAction{"D", "delete", mode.Normal},
			keyAction{"g", "goToTop", mode.Normal},
			keyAction{"G", "goToBottom", mode.Normal},
			keyAction{"esc", "cancelAction", mode.Normal},
			keyAction{"esc", "cancelAction", mode.Insert},
			keyAction{"esc", "cancelAction", mode.Command},
			keyAction{"enter", "confirmAction", mode.Insert},
			keyAction{"enter", "confirmAction", mode.Command},
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
	ki.keyComboCache[key] = true

	actionString := mapToActionString(ki.keysDown)
	if len(ki.keyComboCache) > 0 {
		actionString = mapToActionString(ki.keyComboCache)
	}
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

	if ki.mode != mode.Command {
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
