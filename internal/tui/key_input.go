package tui

import (
	"strings"
	"time"
)

type keyAction struct {
}

type keyMap struct {
	action map[string]string
	keys   []string
	//triggered bool
}

type keyInput struct {
	keysDown map[string]bool
	keyMaps  []keyMap

	isCtrlWDown bool
	isCmdMode   bool
}

func NewKeyInput() keyInput {
	return keyInput{
		isCtrlWDown: false,
		isCmdMode:   false,
		keysDown:    make(map[string]bool),
		keyMaps: []keyMap{
			{action: map[string]string{"ctrl+w l": "focusNextColumn", "mode": "normal"}},
			{action: map[string]string{"ctrl+w h": "focusPrevColumn", "mode": "normal"}},
			{action: map[string]string{"j": "moveDown", "mode": "normal"}},
			{action: map[string]string{"k": "moveUp", "mode": "normal"}},
			{action: map[string]string{"h": "collapse", "mode": "normal"}},
			{action: map[string]string{"l": "expand", "mode": "normal"}},
			{action: map[string]string{"d": "createFolder", "mode": "normal"}},
		},
	}
}

// simulate keyUp event
func (i *keyInput) releaseKey(key string) {
	var timeout time.Duration = 50
	go func() {
		time.Sleep(timeout * time.Millisecond)
		delete(i.keysDown, key)
	}()
}

func (i keyInput) GetKeysDown() []string {
	keys := []string{}
	for key := range i.keysDown {
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
