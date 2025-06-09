package keyinput

import (
	"slices"
	"strings"
	"time"
	"unicode/utf8"

	"bellbird-notes/tui/message"
	"bellbird-notes/tui/mode"
)

type FocusedComponent interface {
	Focused() bool
}

type KeyAction struct {
	Keys string
	//action string
	Cond []KeyCondition
	//Cond KeyCondition
}

type KeyCondition struct {
	Mode       mode.Mode
	Components []FocusedComponent
	Action     func() message.StatusBarMsg
}

type Input struct {
	KeyMap       []string
	KeySequence  map[string]bool
	KeysDown     map[string]bool
	sequenceKeys []string
	//KeyMaps     []keyAction
	actions map[string]func() message.StatusBarMsg

	Ctrl  bool
	Alt   bool
	Shift bool
	Mode  mode.Mode

	Functions []KeyAction
}

func New() *Input {
	return &Input{
		Ctrl:         false,
		Alt:          false,
		Shift:        false,
		Mode:         mode.Normal,
		KeySequence:  make(map[string]bool),
		KeysDown:     make(map[string]bool),
		KeyMap:       []string{},
		sequenceKeys: []string{},
		actions:      map[string]func() message.StatusBarMsg{},
		Functions:    []KeyAction{},
	}
}

func (ki *Input) HandleSequences(key string) []message.StatusBarMsg {
	statusMsg := []message.StatusBarMsg{}
	//statusMsg := []message.StatusBarMsg{{
	//	Content: key,
	//	Column:  statusbarcolumn.KeyInfo,
	//}}

	if key == "esc" {
		ki.ResetKeysDown()
		return statusMsg
	}

	if slices.Contains(ki.sequenceKeys, key) && !ki.KeySequence[key] {
		ki.KeySequence[key] = true
		return statusMsg
	}

	if key == "ctrl+w" {
		ki.KeySequence[key] = true
		ki.Ctrl = true
	}

	// build key sequences
	if len(ki.KeySequence) > 0 {
		for k := range ki.KeySequence {
			key = k + key
		}
	}

	if ki.Ctrl && strings.Contains(key, "ctrl+") {
		ki.KeysDown["ctrl+w"] = true
		key = strings.Split(key, "+")[1]
	}

	ki.KeysDown[key] = true
	keys := mapToActionString(ki.KeysDown)
	statusMsg = append(statusMsg, ki.executeAction(keys))

	ki.releaseKey(key)

	return statusMsg
}

func (ki *Input) executeAction(keys string) message.StatusBarMsg {
	//for k, a := range ki.actions {
	//	debug.LogDebug(keys, k, a)
	//}

	//for key, fn := range ki.actions {
	//	if key == keys {
	//		debug.LogDebug(key, keys)
	//		ki.ResetKeysDown()
	//		ki.resetSequenceCache()
	//		return fn()
	//	}
	//}

	//return message.StatusBarMsg{}

	for i := range ki.Functions {
		k := ki.Functions[i].Keys

		for c := range ki.Functions[i].Cond {
			cond := ki.Functions[i].Cond[c]

			if keys == k && ki.Mode == cond.Mode {
				for ii := range cond.Components {
					if !cond.Components[ii].Focused() {
						continue
					}

					ki.ResetKeysDown()
					ki.resetSequenceCache()

					return cond.Action()
				}
			}
		}
	}

	return message.StatusBarMsg{}
}

func (ki *Input) FetchKeyMap(resetSeq bool) {
	if resetSeq {
		ki.sequenceKeys = []string{}
	}

	for i := range ki.Functions {
		k := ki.Functions[i].Keys
		runes := []byte(k)

		for c := range ki.Functions[i].Cond {
			cond := ki.Functions[i].Cond[c]

			for ii := range cond.Components {
				if !cond.Components[ii].Focused() {
					continue
				}
				if ki.Mode == cond.Mode {
					ki.actions[k] = cond.Action
				}

				if utf8.RuneCount(runes) == 2 &&
					!slices.Contains(ki.sequenceKeys, string(runes[0])) {
					ki.sequenceKeys = append(
						ki.sequenceKeys,
						string(runes[0]),
					)
				}
			}
		}
	}
}

func (ki *Input) ResetKeysDown() {
	ki.Ctrl = false
	ki.KeysDown = make(map[string]bool)
	ki.resetSequenceCache()
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
	ki.KeySequence = map[string]bool{}
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
