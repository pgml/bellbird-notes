package notes

import (
	"errors"
	"os"
	"path/filepath"

	"bellbird-notes/app/debug"
	"bellbird-notes/app/utils"
	"bellbird-notes/tui/bb_errors"
	"bellbird-notes/tui/message"
)

type Note struct {
	Name     string
	Path     string
	IsPinned bool
}

func (n Note) GetName() string {
	return n.Name
}

func List(notePath string) ([]Note, error) {
	var notes []Note

	dirsList, err := os.ReadDir(notePath)
	if err != nil {
		debug.LogErr(err)
		return nil, err
	}

	for _, child := range dirsList {
		filePath := filepath.Join(notePath, child.Name())
		if child.IsDir() || isHidden(filePath) {
			continue
		}

		notes = append(notes, Note{
			Name:     child.Name(),
			Path:     filePath,
			IsPinned: false,
		})
	}

	// Sort directory list aphabetically
	utils.SortSliceAsc(notes, false, nil)

	return notes, nil
}

func Create(path string) error {
	if Exists(path) {
		return errors.New(message.StatusBar.NoteExists)
	}

	if _, err := os.Create(path); err != nil {
		debug.LogErr(err)
		return err
	}

	return nil
}

func Write(path string, content string) (int, error) {
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

func Rename(oldPath string, newPath string) error {
	if err := os.Rename(oldPath, newPath); err != nil {
		debug.LogErr(err)
		return err
	}
	return nil
}

func Delete(path string) error {
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

func Exists(path string) bool {
	if _, err := os.Stat(path); errors.Is(err, os.ErrNotExist) {
		return false
	}
	return true
}

func isHidden(path string) bool {
	return path[0] == 46
}
