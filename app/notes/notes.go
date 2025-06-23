package notes

import (
	"errors"
	"os"
	"path/filepath"
	"strings"

	"bellbird-notes/app/config"
	"bellbird-notes/app/debug"
	"bellbird-notes/app/utils"
	"bellbird-notes/tui/bb_errors"
	"bellbird-notes/tui/message"
)

type Note struct {
	name     string
	Path     string
	IsPinned bool
}

// Name returns the note name without its file extension.
func (n Note) Name() string {
	name := strings.TrimSuffix(
		n.name,
		filepath.Ext(n.name),
	)
	return name
}

// NameWithExt returns the full note name including the extension.
func (n Note) NameWithExt() string {
	if strings.HasSuffix(n.name, n.Ext()) {
		return n.name
	}

	var name strings.Builder
	name.WriteString(n.name)
	name.WriteString(n.Ext())
	return name.String()
}

const (
	Ext       = ".txt"
	legacyExt = ".note"
)

func (n Note) Ext() string { return Ext }

// LegacyExt is the extension used in the old rust version
// of bellbird notes and is just here for compatibility reasons
func (n Note) LegacyExt() string { return legacyExt }

func NewNote(name string, path string, isPinned bool) Note {
	return Note{
		name:     name,
		Path:     path,
		IsPinned: isPinned,
	}
}

// List returns a list of notes in the given directory path.
// Only files with valid extensions (.txt or .note) are included.
// Hidden files and directories are ignored.
func List(notePath string) ([]Note, error) {
	var notes []Note

	dirsList, err := os.ReadDir(notePath)
	if err != nil {
		debug.LogErr(err)
		return nil, err
	}

	conf := config.New()

	for _, child := range dirsList {
		filePath := filepath.Join(notePath, child.Name())

		if child.IsDir() || isHidden(child.Name()) {
			continue
		}

		// skip unsupported files
		if !strings.HasSuffix(child.Name(), Ext) &&
			!strings.HasSuffix(child.Name(), legacyExt) {

			continue
		}

		p, _ := conf.MetaValue(filePath, config.Pinned)
		pinned := p == "true"

		notes = append(notes, Note{
			name:     child.Name(),
			Path:     filePath,
			IsPinned: pinned,
		})
	}

	// Sort directory list aphabetically
	utils.SortSliceAsc(notes, false, nil)

	return notes, nil
}

// Create creates a new note file at the specified path.
func Create(path string) error {
	path = checkPath(path)

	if Exists(path) {
		return errors.New(message.StatusBar.NoteExists)
	}

	if _, err := os.Create(path); err != nil {
		debug.LogErr(err)
		return err
	}

	return nil
}

// Write replaces the contents of a note at the given path with the provided string.
func Write(path string, content string) (int, error) {
	path = checkPath(path)

	if !Exists(path) {
		return 0, errors.New(message.StatusBar.NoteExists)
	}

	f, err := os.OpenFile(path, os.O_WRONLY|os.O_TRUNC, 0644)
	if err != nil {
		debug.LogErr(err)
		return 0, err
	}
	defer f.Close()

	n, err := f.WriteString(content)

	if err != nil {
		debug.LogErr(err)
		return 0, err
	}

	return n, nil
}

// Rename changes the name or path of a note file.
func Rename(oldPath string, newPath string) error {
	newPath = checkPath(newPath)

	if err := os.Rename(oldPath, newPath); err != nil {
		debug.LogErr(err)
		return err
	}
	return nil
}

// Delete removes the specified note file.
func Delete(path string) error {
	path = checkPath(path)

	if _, err := os.Stat(path); err != nil {
		debug.LogErr(err)
		return err
	}

	if err := os.Remove(path); err != nil {
		debug.LogErr(err)
		return &bb_errors.PromptError{Arg: path, Message: err.Error()}
	}

	return nil
}

// Exists checks whether a file exists at the given path.
func Exists(path string) bool {
	if _, err := os.Stat(path); errors.Is(err, os.ErrNotExist) {
		return false
	}
	return true
}

// isHidden returns true if the file or directory is hidden
func isHidden(path string) bool {
	return path[0] == 46
}

// checkPath ensures that the path ends with a valid extension.
// If not, it appends the default extension.
func checkPath(path string) string {
	if strings.HasSuffix(path, Ext) ||
		strings.HasSuffix(path, legacyExt) {

		return path
	}

	return path + Ext
}
