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

// FocusedComponent represents any UI component that can report whether
// it currently has focus.
// Used to check if input should be directed to it.
type FocusedComponent interface {
	Name() string
	Focused() bool
}

type ResetSequenceMsg struct{}

type Action struct {
	binding    string
	isSequence bool
	awaitInput bool
	exec       func() message.StatusBarMsg
	opts       any
	modes      []mode.Mode
}

// Input represents the state and configuration of the input handler,
// including current key sequences, modifier states, mode, and the
// list of all configured key actions.
type Input struct {
	KeySequence    string
	AllowSequences bool
	sequenceKeys   []string
	sequenceLength int

	// sequenceTimeOut is Time in milliseconds to wait for a mapped
	// sequence to complete. This is basically `timeoutlen` from Vim.
	sequenceTimeOut time.Duration

	Space bool
	Ctrl  bool
	Alt   bool
	Mode  mode.Mode
	// contains all componentActions of the currently selected component
	//componentActions map[string]Action
	componentActions map[mode.Mode]map[string]Action

	KeyMap        KeyMap
	DefaultKeyMap []byte
	UserKeyMap    []byte

	Registry   FnRegistry
	Components []FocusedComponent
}

type CmdFn func(opts Options) func() message.StatusBarMsg
type FnRegistry map[string]CmdFn

// New creates and returns a new Input instance with default state.
func New() *Input {
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
		Space:            false,
		Ctrl:             false,
		Alt:              false,
		Mode:             mode.Normal,
		KeySequence:      "",
		AllowSequences:   true,
		sequenceTimeOut:  300,
		sequenceKeys:     []string{},
		sequenceLength:   0,
		componentActions: map[mode.Mode]map[string]Action{},
		KeyMap:           keymap,
		DefaultKeyMap:    defaultKeyMap,
		UserKeyMap:       userKeyMap,
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
func (ki *Input) HandleSequences(key tea.Key) []message.StatusBarMsg {
	if key.String() == "esc" && ki.KeySequence != "" {
		return []message.StatusBarMsg{ki.ResetKeysDown()}
	}

	// special treatment for space to make it simulate a leader key
	if ki.Mode == mode.Normal && !ki.Space {
		if key.Code == 32 {
			ki.KeySequence += key.Keystroke()
			ki.Space = true

			return []message.StatusBarMsg{{
				Content: key.Keystroke(),
				Column:  sbc.KeyInfo,
			}}
		}
	}

	if ki.Ctrl || ki.Alt || ki.Space {
		ki.KeySequence += " " + key.String()
	} else {
		ki.KeySequence += key.String()
	}

	// reset keybinds if we exceed the max length of sequences found in
	// the keymap
	if len(strings.Split(ki.KeySequence, " ")) > ki.sequenceLength {
		ki.ResetKeysDown()
	}

	keyInfoMsg := message.StatusBarMsg{Content: "", Column: sbc.KeyInfo}
	if ki.Mode != mode.Command && !ki.isBinding(ki.KeySequence) && ki.AllowSequences {
		mod, isModifier := ki.isModifier(key.String())

		if slices.Contains(ki.sequenceKeys, ki.KeySequence) || isModifier {
			switch mod {
			case "ctrl":
				ki.Ctrl = true
			case "alt":
				ki.Alt = true
			}

			keyInfo := keyInfoMsg.Content
			if ki.Mode != mode.Insert {
				keyInfo = strings.ReplaceAll(ki.KeySequence, "ctrl", "^")
				if ki.Ctrl {
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
	statusMsg = append(statusMsg, keyInfoMsg, ki.executeAction(ki.KeySequence))
	ki.ResetKeysDown()

	return statusMsg
}

// executeAction attempts to find and execute an action matching the given
// key binding string in the current mode and focused component.
func (ki *Input) executeAction(binding string) message.StatusBarMsg {
	if action, ok := ki.componentActions[ki.Mode][binding]; ok {
		ki.ResetKeysDown()
		return action.exec()
	}

	return message.StatusBarMsg{}
}

// ReloadKeyMap reads the user's keymap.json and rewrites the cached map
// of the key bindings
func (ki *Input) ReloadKeyMap() {
	if f, err := os.ReadFile(ki.KeyMap.Path()); err != nil {
		debug.LogErr("Could not reload keymap", err)
	} else {
		ki.UserKeyMap = f
		ki.FetchKeyMap(true)
	}
}

// FetchKeyMap updates the cached map of key bindings to actions based on
// the currently focused component and the current mode.
//
// It merges the user keymap with the default keymap and
// overrides keybinds from the user map with the default keybinds
// if necessary
func (ki *Input) FetchKeyMap(resetSeq bool) {
	defaultMap, err := ki.parseKeyMap(ki.DefaultKeyMap, resetSeq)
	if err != nil {
		debug.LogErr("error parsing default keynap", err)
	}

	// get the user keymap
	userMap, err := ki.parseKeyMap(ki.UserKeyMap, false)
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

	ki.componentActions = defaultMap
}

// parseKeyMap converts a keymap json string into a executable map
func (ki *Input) parseKeyMap(
	keymap []byte,
	resetSeq bool,
) (map[mode.Mode]map[string]Action, error) {
	if resetSeq {
		ki.sequenceKeys = []string{}
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
		if _, ok := ki.anyComponentFocused(set.ResolveComponents(ki)); !ok {
			continue
		}

		resolvedMode := set.ResolveMode(ki)
		// skip binding is not registerd to this mode
		if set.Mode != "" && resolvedMode.FullString(false) != set.Mode {
			continue
		}

		for key, binding := range set.Bindings {
			actionFn, ok := ki.Registry[binding.Action]
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
						}
					}

					parsed[mode][key] = action
				}
			}

			ki.addSequenceKey(key, false)
		}
	}

	return parsed, nil
}

func (ki *Input) addSequenceKey(binding string, force bool) {
	if binding == "" {
		return
	}

	runeCount := utf8.RuneCountInString(binding)

	seqAmount := strings.Split(binding, " ")
	if len(seqAmount) > ki.sequenceLength {
		ki.sequenceLength = len(seqAmount)
	}

	if force {
		if !slices.Contains(ki.sequenceKeys, binding) {
			ki.sequenceKeys = append(ki.sequenceKeys, binding)
			return
		}
	}

	if runeCount == 3 && binding != "esc" {
		runes := []rune(binding)
		r := string(runes[0]) + string(runes[1])
		if !slices.Contains(ki.sequenceKeys, r) {
			ki.sequenceKeys = append(ki.sequenceKeys, r)
		}
	} else if runeCount == 2 {
		r := string([]rune(binding)[0])
		if !slices.Contains(ki.sequenceKeys, r) {
			ki.sequenceKeys = append(ki.sequenceKeys, r)
		}
	}
}

// isBinding returns wether the given key string is a
// known and valid key binding
func (ki *Input) isBinding(key string) bool {
	if _, ok := ki.componentActions[ki.Mode][key]; ok {
		return true
	}
	return false
}

// anyComponentFocused returns whether any of the components
// in the FocusedComponents slice is focused
func (ki *Input) anyComponentFocused(components []FocusedComponent) (*FocusedComponent, bool) {
	for i := range components {
		if components[i].Focused() {
			return &components[i], true
		}
	}
	return nil, false
}

// ResetKeysDown resets the modifier state flags and
// clears the current key sequence.
func (ki *Input) ResetKeysDown() message.StatusBarMsg {
	ki.Space = false
	ki.Ctrl = false
	ki.Alt = false
	ki.KeySequence = ""

	return message.StatusBarMsg{
		Content: "",
		Column:  sbc.KeyInfo,
	}
}

// ResetSequence resets the key sequence after the given delay.
// Simulates Vim's `timeoutlen`
func (ki *Input) ResetSequence() tea.Cmd {
	return func() tea.Msg {
		time.Sleep(ki.sequenceTimeOut * time.Millisecond)
		return ResetSequenceMsg{}
	}
}
