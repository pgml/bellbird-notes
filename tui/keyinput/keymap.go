package keyinput

import (
	_ "embed"
	"encoding/json"
	"errors"
	"os"
	"path/filepath"
	"slices"

	"bellbird-notes/app"
	"bellbird-notes/app/debug"
	"bellbird-notes/app/utils"
	"bellbird-notes/tui/mode"
)

//go:embed keymap.json
var defaultKeyMap []byte

//go:embed custom.json
var keyMapCustomTpl []byte

const keyMapFileName = "keymap.json"

var Args = struct {
	Outer, Prev, WhiteSpace, Remaining, Operator, AwaitInput,
	End, NewLine, MultiLine, Cycle, IgnoreCase, Insert, Include string
}{
	Outer:      "outer",
	Prev:       "prev",
	WhiteSpace: "white_space",
	Remaining:  "remaining",
	Operator:   "operator",
	AwaitInput: "await_input",
	End:        "end",
	NewLine:    "new_line",
	MultiLine:  "multiline",
	Cycle:      "cycle",
	IgnoreCase: "ignore_case",
	Insert:     "insert",
	Include:    "include",
}

type KeyMap struct {
	path    string
	entries []KeyMapEntry
}

func (m *KeyMap) Path() string { return m.path }

func NewKeyMap() KeyMap {
	path, err := keyMapPath()

	if err != nil {
		debug.LogErr(err)
	}

	return KeyMap{
		path:    path,
		entries: []KeyMapEntry{},
	}
}

func keyMapPath() (string, error) {
	confDir, err := app.ConfigDir()

	if err != nil {
		return "", err
	}

	return filepath.Join(confDir, keyMapFileName), nil
}

// Exists checks whether a file exists at the given path.
func (km *KeyMap) Exists() bool {
	if _, err := os.Stat(km.path); errors.Is(err, os.ErrNotExist) {
		return false
	}
	return true
}

func (km *KeyMap) Create() error {
	f, err := utils.CreateFile(km.path, true)

	if err != nil {
		return err
	}

	defer f.Close()

	_, err = f.Write(keyMapCustomTpl)

	if err != nil {
		return err
	}

	return nil
}

func (km *KeyMap) Content() ([]byte, error) {
	f, err := os.ReadFile(km.path)

	if err != nil {
		return nil, err
	}

	return f, nil
}

type KeyMapEntry struct {
	Mode       string                `json:"mode"`
	Components []string              `json:"components"`
	Bindings   map[string]MapBinding `json:"bindings"`
}

func (e *KeyMapEntry) ResolveComponents(ki *Input) []FocusedComponent {
	var components []FocusedComponent
	for i := range ki.Components {
		if slices.Contains(e.Components, ki.Components[i].Name()) {
			components = append(components, ki.Components[i])
		}
	}
	return components
}

func (e *KeyMapEntry) ResolveMode(ki *Input) mode.Mode {
	switch e.Mode {
	case "normal":
		return mode.Normal
	case "insert":
		return mode.Insert
	case "replace":
		return mode.Replace
	case "visual":
		return mode.Visual
	case "visual_line":
		return mode.VisualLine
	case "visual_block":
		return mode.VisualBlock
	case "command":
		return mode.Command
	case "search":
		return mode.SearchPrompt
	}
	return mode.Normal
}

type Options map[string]any

func (o Options) GetBool(key string) bool {
	bool := false
	if val, ok := o[key]; ok && val == true {
		bool = true
	}
	return bool
}

func (o Options) GetString(key string) string {
	str := ""
	val, ok := o[key]

	if !ok {
		return str
	}

	if s, ok := val.(string); ok {
		str = s
	}

	return str
}

type MapBinding struct {
	Action  string
	Options Options
	HasOpts bool
}

func (b *MapBinding) UnmarshalJSON(data []byte) error {
	var keyString string

	if err := json.Unmarshal(data, &keyString); err == nil {
		b.Action = keyString
		b.Options = Options{}
		b.HasOpts = false

		return nil
	}

	var keyArr []json.RawMessage
	if err := json.Unmarshal(data, &keyArr); err != nil {
		return err
	}

	if len(keyArr) > 0 {
		if err := json.Unmarshal(keyArr[0], &b.Action); err != nil {
			return err
		}

		if len(keyArr) > 1 {
			if err := json.Unmarshal(keyArr[1], &b.Options); err != nil {
				return err
			}
			b.HasOpts = true
		}
	}

	return nil
}
