package keyinput

import (
	"encoding/json"
	"maps"
	"os"
	"slices"
	"strings"
	"time"
	"unicode/utf8"

	tea "github.com/charmbracelet/bubbletea/v2"
	"github.com/tailscale/hujson"

	"bellbird-notes/app/debug"
	"bellbird-notes/tui/message"
	"bellbird-notes/tui/mode"
	sbc "bellbird-notes/tui/types/statusbar_column"
)

type Handler interface {
	FnRegistry() MotionRegistry
	Mode() *mode.ModeInstance
}

// FocusedComponent represents any UI component that can report whether
// it currently has focus.
// Used to check if input should be directed to it.
type FocusedComponent interface {
	Title() string
	Focused() bool
}

type ResetSequenceMsg struct{}

type Action struct {
	binding string
	exec    func() message.StatusBarMsg
	modes   []mode.Mode
	opts    Options
}

func (a *Action) Opts() Options   { return a.opts }
func (a *Action) Binding() string { return a.binding }

// Input represents the state and configuration of the input handler,
// including current key sequences, modifier states, mode, and the
// list of all configured key actions.
type Input struct {
	KeySequence    string
	AllowSequences bool
	sequenceKeys   []string
	sequenceLength int

	// AwaitInputAction stores the action to execute after receiving additional input.
	// This is used when a keybind has "await_input": true in the keymap,
	// meaning the action should not run immediately but wait for further key input.
	AwaitInputAction *Action

	// sequenceTimeOut is Time in milliseconds to wait for a mapped
	// sequence to complete. This is basically `timeoutlen` from Vim.
	sequenceTimeOut time.Duration

	Space bool
	Ctrl  bool
	Alt   bool
	Mode  *mode.ModeInstance

	// contains all componentActions of the currently selected component
	componentActions map[mode.Mode]map[string]Action

	KeyMap        KeyMap
	DefaultKeyMap []byte
	UserKeyMap    []byte

	Registry   MotionRegistry
	Components []FocusedComponent
}

type Motion func(opts Options) func() message.StatusBarMsg
type MotionRegistry map[string]Motion

// New creates and returns a new Input instance with default state.
func New(h Handler) *Input {
	keymap := NewKeyMap()

	if !keymap.Exists() {
		err := keymap.Create()
		if err != nil {
			debug.LogDebug(err)
		}
	}

	userKeyMap, err := keymap.Content()
	if err != nil {
		debug.LogErr(err)
	}

	return &Input{
		KeySequence:      "",
		AllowSequences:   true,
		sequenceKeys:     []string{},
		sequenceLength:   0,
		AwaitInputAction: nil,
		sequenceTimeOut:  300,
		Space:            false,
		Ctrl:             false,
		Alt:              false,
		Mode:             h.Mode(),
		componentActions: map[mode.Mode]map[string]Action{},
		KeyMap:           keymap,
		DefaultKeyMap:    defaultKeyMap,
		UserKeyMap:       userKeyMap,
		Registry:         h.FnRegistry(),
		Components:       []FocusedComponent{},
	}
}

// isModifier checks if the provided key string represents a modifier key
// prefix ("ctrl+" or "alt+").
func (input *Input) isModifier(binding string) (string, bool) {
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
func (input *Input) HandleSequences(key tea.Key) []message.StatusBarMsg {
	if key.String() == "esc" && input.KeySequence != "" {
		return []message.StatusBarMsg{input.ResetKeysDown()}
	}

	// special treatment for space to make it simulate a leader key
	if input.Mode.Current == mode.Normal && !input.Space && input.KeySequence == "" {
		if key.Code == 32 {
			input.KeySequence += key.Keystroke()
			input.Space = true

			return []message.StatusBarMsg{{
				Content: key.Keystroke(),
				Column:  sbc.KeyInfo,
			}}
		}
	}

	if input.Ctrl || input.Alt || input.Space {
		input.KeySequence += " " + key.String()
	} else {
		input.KeySequence += key.String()
	}

	// If we need to wait for further input cache the original action
	if action, ok := input.componentActions[input.Mode.Current][input.KeySequence]; ok {
		if input.AwaitInputAction == nil && action.opts.GetBool("await_input") {
			input.AwaitInputAction = &action
		}
	}

	// reset keybinds if we exceed the max length of sequences found in
	// the keymap
	if len(strings.Split(input.KeySequence, " ")) > input.sequenceLength {
		input.ResetKeysDown()
	}

	keyInfoMsg := message.StatusBarMsg{Content: "", Column: sbc.KeyInfo}
	if input.Mode.Current != mode.Command &&
		!input.isBinding(input.KeySequence) &&
		input.AllowSequences {

		mod, isModifier := input.isModifier(key.String())

		if slices.Contains(input.sequenceKeys, input.KeySequence) || isModifier {
			switch mod {
			case "ctrl":
				input.Ctrl = true
			case "alt":
				input.Alt = true
			}

			keyInfo := keyInfoMsg.Content
			if input.Mode.Current != mode.Insert {
				keyInfo = strings.ReplaceAll(input.KeySequence, "ctrl", "^")
				if input.Ctrl {
					keyInfo = strings.ReplaceAll(keyInfo, "+", "")
				}
			}

			return []message.StatusBarMsg{{
				Content: keyInfo,
				Column:  sbc.KeyInfo,
			}}
		}
	}

	statusMsg := []message.StatusBarMsg{}
	statusMsg = append(statusMsg, keyInfoMsg, input.executeAction(input.KeySequence))
	input.ResetKeysDown()

	return statusMsg
}

// executeAction attempts to find and execute an action matching the given
// key binding string in the current mode and focused component.
func (input *Input) executeAction(binding string) message.StatusBarMsg {
	if action := input.AwaitInputAction; action != nil {
		return action.exec()
	}

	if action, ok := input.componentActions[input.Mode.Current][binding]; ok {
		return action.exec()
	}

	return message.StatusBarMsg{}
}

// ReloadKeyMap reads the user's keymap.json and rewrites the cached map
// of the key bindings
func (input *Input) ReloadKeyMap() {
	if f, err := os.ReadFile(input.KeyMap.Path()); err != nil {
		debug.LogErr("Could not reload keymap", err)
	} else {
		input.UserKeyMap = f
		input.FetchKeyMap(true)
	}
}

// FetchKeyMap updates the cached map of key bindings to actions based on
// the currently focused component and the current mode.
//
// It merges the user keymap with the default keymap and
// overrides keybinds from the user map with the default keybinds
// if necessary
func (input *Input) FetchKeyMap(resetSeq bool) {
	defaultMap, err := input.parseKeyMap(input.DefaultKeyMap, resetSeq)
	if err != nil {
		debug.LogErr("error parsing default keynap", err)
	}

	// get the user keymap
	userMap, err := input.parseKeyMap(input.UserKeyMap, false)
	if err != nil {
		debug.LogErr("error parsing user keymap", err)
	}

	// merge user keymap with the default and override keybinds if necessary
	for mode, bindings := range userMap {
		if _, exists := defaultMap[mode]; !exists {
			defaultMap[mode] = make(map[string]Action)
		}

		maps.Copy(defaultMap[mode], bindings)
	}

	input.componentActions = defaultMap
}

// parseKeyMap converts a keymap json string into a executable map
func (input *Input) parseKeyMap(
	keymap []byte,
	resetSeq bool,
) (map[mode.Mode]map[string]Action, error) {
	if resetSeq {
		input.sequenceKeys = []string{}
	}

	parsed := map[mode.Mode]map[string]Action{}
	entries := []KeyMapEntry{}

	// remove trailing commas and comments
	cleanedMap, err := hujson.Standardize(keymap)
	if err != nil {
		return nil, err
	}

	if err := json.Unmarshal(cleanedMap, &entries); err != nil {
		return nil, err
	}

	// map to store the modes per binding
	modes := map[string][]mode.Mode{}

	for _, set := range entries {
		if _, ok := input.anyComponentFocused(set.ResolveComponents(input)); !ok {
			continue
		}

		resolvedMode := set.ResolveMode(input)
		// skip binding is not registerd to this mode
		if set.Mode != "" && resolvedMode.FullString(false) != set.Mode {
			continue
		}

		for key, binding := range set.Bindings {
			actionFn, ok := input.Registry[binding.Action]
			if !ok {
				continue
			}

			// Add non-existing modes for this binding.
			// If no mode is set append all relvevant modes.
			if set.Mode == "" {
				modes[key] = append(modes[key], mode.SupportsMotion()...)
			} else {
				if !slices.Contains(modes[key], resolvedMode) {
					modes[key] = append(modes[key], resolvedMode)
				}
			}

			// If "operator" is set manually make the key a sequence key
			if binding.HasOpts && binding.Options.GetBool("operator") {
				input.addSequenceKey(key, true)
			}

			for key, modeSlice := range modes {
				for _, mode := range modeSlice {
					if parsed[mode] == nil {
						parsed[mode] = make(map[string]Action)
					}

					// Create the actual component actions
					action, ok := parsed[mode][key]

					if !ok {
						action = Action{
							binding: key,
							exec:    actionFn(binding.Options),
							opts:    binding.Options,
						}
					}

					parsed[mode][key] = action
				}
			}

			input.addSequenceKey(key, false)
		}
	}

	return parsed, nil
}

func (input *Input) addSequenceKey(binding string, force bool) {
	if binding == "" {
		return
	}

	runeCount := utf8.RuneCountInString(binding)

	seqAmount := strings.Split(binding, " ")
	if len(seqAmount) > input.sequenceLength {
		input.sequenceLength = len(seqAmount)
	}

	if force {
		if !slices.Contains(input.sequenceKeys, binding) {
			input.sequenceKeys = append(input.sequenceKeys, binding)
			return
		}
	}

	if runeCount == 3 && binding != "esc" {
		runes := []rune(binding)
		r := string(runes[0]) + string(runes[1])
		if !slices.Contains(input.sequenceKeys, r) {
			input.sequenceKeys = append(input.sequenceKeys, r)
		}
	} else if runeCount == 2 {
		r := string([]rune(binding)[0])
		if !slices.Contains(input.sequenceKeys, r) {
			input.sequenceKeys = append(input.sequenceKeys, r)
		}
	}
}

// isBinding returns wether the given key string is a
// known and valid key binding
func (input *Input) isBinding(key string) bool {
	if action, ok := input.componentActions[input.Mode.Current][key]; ok {
		o := action.opts
		if o.GetBool("operator") && o.GetBool("await_input") {
			return false
		}
		return true
	}
	return false
}

// anyComponentFocused returns whether any of the components
// in the FocusedComponents slice is focused
func (input *Input) anyComponentFocused(components []FocusedComponent) (*FocusedComponent, bool) {
	for i := range components {
		if components[i].Focused() {
			return &components[i], true
		}
	}
	return nil, false
}

// ResetKeysDown resets the modifier state flags and
// clears the current key sequence.
func (input *Input) ResetKeysDown() message.StatusBarMsg {
	input.Space = false
	input.Ctrl = false
	input.Alt = false
	input.KeySequence = ""
	input.AwaitInputAction = nil

	return message.StatusBarMsg{
		Content: "",
		Column:  sbc.KeyInfo,
	}
}

// ResetSequence resets the key sequence after the given delay.
// Simulates Vim's `timeoutlen`
func (input *Input) ResetSequence() tea.Cmd {
	return func() tea.Msg {
		time.Sleep(input.sequenceTimeOut * time.Millisecond)
		return ResetSequenceMsg{}
	}
}
