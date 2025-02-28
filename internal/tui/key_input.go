package tui

import (
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
	keysDown map[string]bool
	keyMaps  []keyAction

	isCtrlWDown bool
	mode        mode.Mode

	functions map[string]func()
}

func NewKeyInput() *KeyInput {
	return &KeyInput{
		isCtrlWDown: false,
		mode:        mode.Normal,
		keysDown:    make(map[string]bool),
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
			keyAction{"esc", "cancelAction", mode.Normal},
			keyAction{"esc", "cancelAction", mode.Insert},
			keyAction{"esc", "cancelAction", mode.Command},
			keyAction{"enter", "confirmAction", mode.Insert},
			keyAction{"enter", "confirmAction", mode.Command},
		},
	}
}

func (m *KeyInput) handleKeyCombos(key string) {
	if key == "ctrl+w" {
		m.isCtrlWDown = true
	}

	if m.isCtrlWDown && strings.Contains(key, "ctrl+") {
		m.keysDown["ctrl+w"] = true
		key = strings.Split(key, "+")[1]
	}

	m.keysDown[key] = true

	actionString := mapToActionString(m.keysDown)
	m.executeAction(actionString)

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

	if m.mode != mode.Command {
		m.releaseKey(key)
	}
}

func (m *KeyInput) executeAction(keys string) {
	for _, keyMap := range m.keyMaps {
		if keyMap.keys == keys && m.mode == keyMap.mode {
			if fn, exists := m.functions[keyMap.action]; exists {
				fn()
				m.resetKeysDown()
			}
			return
		}
	}
}

func (m *KeyInput) resetKeysDown() {
	m.isCtrlWDown = false
	m.keysDown = make(map[string]bool)
}

// simulate keyUp event
func (m *KeyInput) releaseKey(key string) {
	var timeout time.Duration = 50
	go func() {
		time.Sleep(timeout * time.Millisecond)
		delete(m.keysDown, key)
	}()
}

func (m KeyInput) GetKeysDown() []string {
	keys := []string{}
	for key := range m.keysDown {
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
