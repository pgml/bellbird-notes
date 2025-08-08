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

type HistoryType int

const (
	Search HistoryType = iota
	Command
	Mark
)

var historyTypes = map[HistoryType]string{
	Search:  "SEARCH",
	Command: "CMD",
	Mark:    "MARK",
}

func (t HistoryType) String() string {
	return historyTypes[t]
}

type StateEntry struct {
	historyType HistoryType
	timestamp   string
	content     string
}

func (e *StateEntry) Content() string {
	return e.content
}

func NewEntry(stateType HistoryType, content string) StateEntry {
	return StateEntry{
		historyType: stateType,
		timestamp:   time.Now().Format(time.RFC3339),
		content:     content,
	}
}

type State struct {
	filePath string
	entries  []StateEntry
	curIndex int
}

// removeLastOccurrences removes the last occurrence of an entry
// with the specified HistoryType and content from the entries slice.
func (s *State) removeLastOccurrences(st HistoryType, c string) {
	for i := len(s.entries) - 1; i >= 0; i-- {
		if s.entries[i].historyType == st && s.entries[i].content == c {
			copy(s.entries[i:], s.entries[i+1:])
			s.entries = s.entries[:len(s.entries)-1]
			return
		}
	}
}

// Entries returns a slice of StateEntry filtered by the given HistoryType.
func (s *State) Entries(st HistoryType) []StateEntry {
	entries := []StateEntry{}

	for _, entry := range s.entries {
		if entry.historyType == st {
			entries = append(entries, entry)
		}
	}

	return entries
}

// Commands returns all entries of type Command.
func (s *State) Commands() []StateEntry {
	commands := []StateEntry{}

	for _, entry := range s.entries {
		if entry.historyType != Command {
			continue
		}

		commands = append(commands, entry)
	}

	return commands
}

// SearchResults returns all entries of type Search.
func (s *State) SearchResults() []StateEntry {
	searchResults := []StateEntry{}

	for _, entry := range s.entries {
		if entry.historyType != Search {
			continue
		}

		searchResults = append(searchResults, entry)
	}

	return searchResults
}

// New initializes a new State from the state file.
// If the state file doesn*t exist, it is created.
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

// Append adds a new StateEntry to the entries slice.
// It removes previous occurrences to avoid duplication.
func (s *State) Append(entry StateEntry) {
	if entry.content == "" {
		return
	}

	s.removeLastOccurrences(entry.historyType, entry.content)
	s.entries = append(s.entries, entry)
	s.curIndex = len(s.entries) - 1
}

// Read loads history entries from the state file.
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

		var historyType HistoryType
		for hisType, str := range historyTypes {
			if ln[0] == str {
				historyType = hisType
			}
		}

		s.entries = append(s.entries, StateEntry{
			historyType: historyType,
			timestamp:   ln[1],
			content:     strings.TrimSuffix(ln[2], "\n"),
		})
	}

	return nil
}

// Write saves the current entries to the state file.
// TYPE|TIMESTAMP|CONTENT
func (s *State) Write() error {
	f, err := os.OpenFile(s.filePath, os.O_WRONLY|os.O_TRUNC, 0644)
	if err != nil {
		debug.LogErr(err)
		return err
	}
	defer f.Close()

	for _, entry := range s.entries {
		var line strings.Builder
		line.WriteString(entry.historyType.String())
		line.WriteRune('|')
		line.WriteString(entry.timestamp)
		line.WriteRune('|')
		line.WriteString(entry.content)
		line.WriteRune('\n')

		_, err := f.WriteString(line.String())

		if err != nil {
			debug.LogErr(err)
		}
	}

	return nil
}

func (s *State) CycleCommands(forward bool) StateEntry {
	commands := s.Entries(Command)
	s.curIndex = utils.Clamp(s.curIndex, 0, len(commands)-1)

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

func (s *State) CycleSearchResults(forward bool) StateEntry {
	searchResults := s.Entries(Search)
	s.curIndex = utils.Clamp(s.curIndex, 0, len(searchResults)-1)

	if len(s.SearchResults()) <= 0 {
		return StateEntry{}
	}

	entry := s.SearchResults()[s.curIndex]

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
