package state

import (
	"bellbird-notes/app"
	"bellbird-notes/app/debug"
	"bellbird-notes/app/utils"
	"bufio"
	"os"
	"strings"
	"time"
)

type StateType int

const (
	Search StateType = iota
	Command
	Mark
)

var stateTypes = map[StateType]string{
	Search:  "search",
	Command: "command",
	Mark:    "mark",
}

func (t StateType) String() string {
	return stateTypes[t]
}

type StateEntry struct {
	stateType StateType
	timestamp string
	content   string
}

func (e *StateEntry) Content() string {
	return e.content
}

func NewEntry(stateType StateType, content string) StateEntry {
	return StateEntry{
		stateType: stateType,
		timestamp: time.Now().Format(time.RFC3339),
		content:   content,
	}
}

type State struct {
	filePath string
	entries  []StateEntry
	curIndex int
}

func (s *State) Entries() []StateEntry { return s.entries }

func (s *State) Commands() []StateEntry {
	commands := []StateEntry{}

	for _, entry := range s.entries {
		if entry.stateType != Command {
			continue
		}

		commands = append(commands, entry)
	}

	return commands
}

func New() *State {
	filePath, err := app.StateFile()
	if err != nil {
		return &State{}
	}

	if _, err := os.Stat(filePath); err != nil {
		utils.CreateFile(filePath, false)
	}

	return &State{
		filePath: filePath,
		entries:  []StateEntry{},
	}
}

func (s *State) Append(entry StateEntry) {
	if len(s.entries) > 0 {
		lastIndex := s.entries[len(s.entries)-1]

		debug.LogDebug(lastIndex.content, entry.content)
		if lastIndex.content == entry.content {
			debug.LogDebug("asdasd")
			return
		}
	}

	s.entries = append(s.entries, entry)

	for _, entry := range s.entries {
		debug.LogDebug(entry.stateType.String(), entry.timestamp, entry.content)
	}

	s.curIndex = len(s.entries) - 1
}

func (s *State) Read() error {
	file, err := os.OpenFile(s.filePath, os.O_RDONLY, 0644)

	if err != nil {
		return err
	}
	defer file.Close()

	reader := bufio.NewReader(file)

	for {
		line, err := reader.ReadSlice('\n')
		if err != nil {
			if err.Error() == "EOF" {
				break
			}
		}

		ln := strings.Split(string(line), "|")
		var state StateType

		switch ln[0] {
		case "command":
			state = Command
		case "search":
			state = Search
		case "mark":
			state = Mark
		}

		s.entries = append(s.entries, StateEntry{
			stateType: state,
			timestamp: ln[1],
			content:   strings.TrimSuffix(ln[2], "\n"),
		})
	}

	return nil
}

func (s *State) Write() error {
	f, err := os.OpenFile(s.filePath, os.O_WRONLY|os.O_TRUNC, 0644)
	if err != nil {
		debug.LogErr(err)
		return err
	}
	defer f.Close()

	for _, entry := range s.entries {
		var line strings.Builder
		line.WriteString(entry.stateType.String())
		line.WriteRune('|')
		line.WriteString(entry.timestamp)
		line.WriteRune('|')
		line.WriteString(entry.content)
		line.WriteRune('\n')

		debug.LogDebug(line.String())
		_, err := f.WriteString(line.String())

		if err != nil {
			debug.LogErr(err)
		}
	}

	return nil
}

func (s *State) CycleCommands(forward bool) StateEntry {
	s.curIndex = utils.Clamp(s.curIndex, 0, len(s.entries)-1)

	if len(s.Commands()) <= 0 {
		return StateEntry{}
	}

	entry := s.Commands()[s.curIndex]

	if forward {
		s.curIndex++
	} else {
		s.curIndex--
	}

	return entry
}

func (s *State) ResetIndex() {
	s.curIndex = len(s.entries) - 1
}

//command|2025-08-05T19:20:15Z|open keymap
//command|2025-08-05T19:20:15Z|w
//search|2025-08-05T19:20:25Z|append(
