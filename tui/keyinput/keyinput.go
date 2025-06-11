package keyinput

import (
	"slices"
	"strings"
	"unicode/utf8"

	"bellbird-notes/tui/message"
	"bellbird-notes/tui/mode"
)

// FocusedComponent represents any UI component that can report whether
// it currently has focus.
// Used to check if input should be directed to it.
type FocusedComponent interface {
	Focused() bool
}

// KeyBinding represents one or more keys that trigger a specific action.
type KeyBinding struct {
	keys []string
}

// KeyBindings is a constructor that creates a KeyBinding from a list of keys.
func KeyBindings(keys ...string) KeyBinding {
	return KeyBinding{keys: keys}
}

// KeyAction is a set of of key bindings with one or more conditions
// under which the action can be triggered
type KeyAction struct {
	Bindings KeyBinding
	Cond     []KeyCondition
}

// KeyCondition represents the conditions under which a key action
// should be triggered. It specifies the required mode, the UI components
// that must be focused, and the action function to execute when matched.
type KeyCondition struct {
	Mode       mode.Mode
	Components []FocusedComponent
	Action     func() message.StatusBarMsg
}

// matchContext represents the current input and UI state
// used for evaluating whether a key binding matches.
// It encapsulates the current mode, the component being checked,
// and the key binding string.
type matchContext struct {
	mode      mode.Mode
	component FocusedComponent
	binding   string
}

// Input represents the state and configuration of the input handler,
// including current key sequences, modifier states, mode, and the
// list of all configured key actions.
type Input struct {
	KeySequence  string
	sequenceKeys []string
	actions      map[string]func() message.StatusBarMsg
	Ctrl         bool
	Alt          bool
	Mode         mode.Mode
	Functions    []KeyAction
}

// Matches checks if the given matchContext satisfies the KeyCondition.
// It returns true if the mode matches and either:
// - no specific component is provided and any of the condition's components are focused, or
// - the provided component matches one in the condition and is currently focused.
func (kc KeyCondition) Matches(ctx matchContext) bool {
	if kc.Mode != ctx.mode {
		return false
	}

	if ctx.component == nil {
		for _, c := range kc.Components {
			if c.Focused() {
				return true
			}
		}
		return false
	}

	for _, c := range kc.Components {
		if c == ctx.component && c.Focused() {
			return true
		}
	}

	return false
}

// New creates and returns a new Input instance with default state.
func New() *Input {
	return &Input{
		Ctrl:         false,
		Alt:          false,
		Mode:         mode.Normal,
		KeySequence:  "",
		sequenceKeys: []string{},
		Functions:    []KeyAction{},
		actions:      map[string]func() message.StatusBarMsg{},
	}
}

// isModifier checks if the provided key string represents a modifier key
// prefix ("ctrl+" or "alt+").
func (ki *Input) isModifier(binding string) (string, bool) {
	if ok := strings.HasPrefix(binding, "ctrl+"); ok {
		return "ctrl", true
	}

	if ok := strings.HasPrefix(binding, "alt+"); ok {
		return "alt", true
	}

	return "", false
}

// HandleSequences processes an incoming key string, updating the internal
// key sequence and modifier states as needed, and executing any matching
// actions.
func (ki *Input) HandleSequences(key string) []message.StatusBarMsg {
	if key == "esc" && ki.KeySequence != "" {
		ki.ResetKeysDown()
		return nil
	}

	if !ki.isBinding(key) {
		mod, isModifier := ki.isModifier(key)

		if ki.KeySequence == "" &&
			(slices.Contains(ki.sequenceKeys, key) || isModifier) {
			ki.KeySequence = key

			if ki.Ctrl || ki.Alt {
				ki.KeySequence += " " + key
			}

			switch mod {
			case "ctrl":
				ki.Ctrl = true
			case "alt":
				ki.Alt = true
			}

			return nil
		}
	}

	if ki.KeySequence != "" {
		if ki.Ctrl || ki.Alt {
			key = " " + key
		}
		key = ki.KeySequence + key
	}

	statusMsg := []message.StatusBarMsg{}
	statusMsg = append(statusMsg, ki.executeAction(key))
	ki.ResetKeysDown()

	return statusMsg
}

// executeAction attempts to find and execute an action matching the given
// key binding string in the current mode and focused component.
func (ki *Input) executeAction(binding string) message.StatusBarMsg {
	ctx := matchContext{
		mode:    ki.Mode,
		binding: binding,
	}

	for _, action := range ki.matchActions(ctx) {
		ki.ResetKeysDown()
		return action()
	}

	return message.StatusBarMsg{}
}

// FetchKeyMap updates the cached map of key bindings to actions based on
// the currently focused component and the current mode.
func (ki *Input) FetchKeyMap(resetSeq bool) {
	if resetSeq {
		ki.sequenceKeys = []string{}
	}

	ki.actions = map[string]func() message.StatusBarMsg{}

	for _, action := range ki.Functions {
		for _, key := range action.Bindings.keys {
			for _, cond := range action.Cond {
				if cond.Mode != ki.Mode {
					continue
				}

				if !ki.anyComponentFocused(cond.Components) {
					continue
				}

				ki.actions[key] = cond.Action
				ki.addSequenceKey(key)
			}
		}
	}
}

func (ki *Input) addSequenceKey(binding string) {
	if utf8.RuneCountInString(binding) == 2 {
		r := string([]rune(binding)[0])
		if !slices.Contains(ki.sequenceKeys, r) {
			ki.sequenceKeys = append(ki.sequenceKeys, r)
		}
	}
}

// isBinding returns wether the given key string is a
// known and valid key binding
func (ki *Input) isBinding(key string) bool {
	_, ok := ki.actions[key]
	return ok
}

// matchActions returns all matching actions for the given matchContext.
// It filters the registered functions by key binding and then checks whether each
// associated condition matches the context (mode + focused component).
func (ki *Input) matchActions(ctx matchContext) []func() message.StatusBarMsg {
	var matched []func() message.StatusBarMsg

	for _, action := range ki.Functions {
		if !slices.Contains(action.Bindings.keys, ctx.binding) {
			continue
		}

		for _, cond := range action.Cond {
			if cond.Matches(ctx) {
				matched = append(matched, cond.Action)
			}
		}
	}

	return matched
}

// anyComponentFocused returns whether any of the components
// in the FocusedComponents slice is focused
func (ki *Input) anyComponentFocused(components []FocusedComponent) bool {
	for _, c := range components {
		if c.Focused() {
			return true
		}
	}
	return false
}

// ResetKeysDown resets the modifier state flags and
// clears the current key sequence.
func (ki *Input) ResetKeysDown() {
	ki.Ctrl = false
	ki.Alt = false
	ki.KeySequence = ""
}
