package notes

import (
	"errors"
	"io"
	"os"
	"path/filepath"
	"strings"

	"bellbird-notes/app/config"
	"bellbird-notes/app/debug"
	"bellbird-notes/app/utils"
	"bellbird-notes/tui/bb_errors"
)

const (
	Ext       = ".txt"
	LegacyExt = ".note"
	ConfExt   = ".conf"
)

type Note struct {
	Path     string
	IsPinned bool
}

// Name returns the note name without its file extension.
func (note Note) Name() string {
	name := filepath.Base(note.Path)
	name = strings.TrimSuffix(
		name,
		filepath.Ext(name),
	)
	return name
}

// NameWithExt returns the full note name including the extension.
func (note Note) NameWithExt() string {
	filename := filepath.Base(note.Path)
	if strings.HasSuffix(filename, note.Ext()) {
		return filename
	}

	var name strings.Builder
	name.WriteString(filename)
	name.WriteString(note.Ext())
	return name.String()
}

func (note Note) Ext() string { return Ext }

// LegacyExt is the extension used in the old rust version
// of bellbird notes and is just here for compatibility reasons
func (note Note) LegacyExt() string { return LegacyExt }

func NewNote(path string, isPinned bool) Note {
	return Note{
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
			!strings.HasSuffix(child.Name(), LegacyExt) {

			continue
		}

		p, _ := conf.MetaValue(filePath, config.Pinned)
		pinned := p == "true"

		notes = append(notes, Note{
			Path:     filePath,
			IsPinned: pinned,
		})
	}

	// Sort directory list aphabetically
	utils.SortSliceAsc(notes, false, nil)

	return notes, nil
}

// Create creates a new note file at the specified path.
func Create(path string) (Note, error) {
	path = CheckPath(path)
	note := Note{}

	if _, err := Exists(path); err == nil {
		return note, err
	}

	if _, err := os.Create(path); err != nil {
		debug.LogErr(err)
		return note, err
	}

	return NewNote(path, false), nil
}

// Write replaces the contents of a note at the given path with the provided string.
func Write(path string, content string, forceCreate bool) (int, error) {
	if IsNote(path) {
		path = CheckPath(path)

		if forceCreate {
			if _, err := Create(path); err != nil {
				return 0, err
			}
		} else {
			if _, err := Exists(path); err != nil {
				return 0, nil
			}
		}
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
	newPath = CheckPath(newPath)

	if f, _ := Exists(newPath); f != nil {
		return errors.New("Couldn't rename note. Note already exists.")
	}

	if err := os.Rename(oldPath, newPath); err != nil {
		debug.LogErr(err)
		return err
	}
	return nil
}

// Delete removes the specified note file.
func Delete(path string) error {
	path = CheckPath(path)

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
func Exists(path string) (os.FileInfo, error) {
	f, err := os.Stat(path)

	if errors.Is(err, os.ErrNotExist) {
		return nil, err
	}

	return f, nil
}

func Copy(oldPath, newPath string) error {
	r, err := os.Open(oldPath)
	if err != nil {
		return err
	}
	defer r.Close() // ignore error: file was opened read-only.

	w, err := os.Create(newPath)
	if err != nil {
		return err
	}

	defer func() {
		// Report the error, if any, from Close, but do so
		// only if there isn't already an outgoing error.
		if c := w.Close(); err == nil {
			err = c
		}
	}()

	_, err = io.Copy(w, r)
	return err
}

// GetValidPath ensures the path is always valid for creating a new note.
// If the file already exists it appends "Copy" to the filename.
func GetValidPath(path string, forceNote bool) string {
	if forceNote {
		path = CheckPath(path)
	} else {
		if !IsNote(path) {
			return path
		}
	}

	note := NewNote(path, false)
	name := note.Name()
	path = filepath.Dir(note.Path)

	var newPath string

	// Apppend Copy if there's already a note
	if f, _ := Exists(path + "/" + note.NameWithExt()); f != nil {
		name += " Copy"
	}

	// Append notes extension
	newPath = CheckPath(path + "/" + name)

	// Rinse and repeat if the copy also already exists
	if _, err := Exists(note.Path); err == nil {
		newPath = GetValidPath(newPath, false)
	}

	return newPath
}

// CheckPath ensures that the path ends with a valid extension.
// If not, it appends the default extension.
func CheckPath(path string) string {
	if IsNote(path) {
		return path
	}

	return path + Ext
}

// isHidden returns true if the file or directory is hidden
func isHidden(path string) bool {
	return path[0] == 46
}

func IsNote(path string) bool {
	if strings.HasSuffix(path, Ext) ||
		strings.HasSuffix(path, LegacyExt) ||
		strings.HasSuffix(path, ConfExt) {

		return true
	}

	return false
}
