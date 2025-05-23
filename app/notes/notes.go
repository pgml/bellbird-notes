package notes

import (
	"os"
	"path/filepath"

	"bellbird-notes/app"
	"bellbird-notes/app/utils"
	"bellbird-notes/tui/errors"
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
		app.LogErr(err)
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
	if _, err := os.Create(path); err != nil {
		app.LogErr(err)
		return err
	}
	return nil
}

func Rename(oldPath string, newPath string) error {
	if err := os.Rename(oldPath, newPath); err != nil {
		app.LogErr(err)
		return err
	}
	return nil
}

func Delete(path string) error {
	if _, err := os.Stat(path); err != nil {
		app.LogErr(err)
		return err
	}

	if err := os.Remove(path); err != nil {
		app.LogErr(err)
		return &errors.PromptError{Arg: path, Message: err.Error()}
	}

	return nil
}

func isHidden(path string) bool {
	return path[0] == 46
}
