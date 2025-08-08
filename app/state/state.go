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

func (s *State) removeLastOccurences(st HistoryType, c string) {
	for i := len(s.entries) - 1; i >= 0; i-- {
		if s.entries[i].historyType == st && s.entries[i].content == c {
			copy(s.entries[i:], s.entries[i+1:])
			s.entries = s.entries[:len(s.entries)-1]
			return
		}
	}
}

func (s *State) Entries(st HistoryType) []StateEntry {
	entries := []StateEntry{}

	for _, entry := range s.entries {
		if entry.historyType == st {
			entries = append(entries, entry)
		}
	}

	return entries
}

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
	//if len(s.entries) > 0 {
	//	lastIndex := s.entries[len(s.entries)-1]

	//	if lastIndex.content == entry.content || entry.content == "" {
	//		return
	//	}

	//	s.removeLastOccurences(entry.historyType, entry.content)
	//}

	if entry.content == "" {
		return
	}

	s.removeLastOccurences(entry.historyType, entry.content)
	s.entries = append(s.entries, entry)
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

		debug.LogDebug(line.String())
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
