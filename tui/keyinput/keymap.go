package keyinput

import (
	"encoding/json"
	"slices"

	"bellbird-notes/tui/mode"
)

type KeyMap struct {
	Mode       string                `json:"mode"`
	Components []string              `json:"components"`
	Bindings   map[string]MapBinding `json:"bindings"`
}

func (km *KeyMap) ResolveComponents(ki *Input) []FocusedComponent {
	var components []FocusedComponent
	for i := range ki.Components {
		if slices.Contains(km.Components, ki.Components[i].Name()) {
			components = append(components, ki.Components[i])
		}
	}
	return components
}

func (km *KeyMap) ResolveMode(ki *Input) mode.Mode {
	switch km.Mode {
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

type MapBinding struct {
	Action  string
	Options Options
	HasOpts bool
}

func (b *MapBinding) UnmarshalJSON(data []byte) error {
	var keyString string

	if err := json.Unmarshal(data, &keyString); err == nil {
		b.Action = keyString
		b.Options = nil
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
		}
		b.HasOpts = true
	}

	return nil
}
