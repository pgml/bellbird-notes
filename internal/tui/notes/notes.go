package notes

import (
	"bellbird-notes/internal/app"
	"os"
	"path/filepath"
)

type Note struct {
	Name     string
	Path     string
	IsPinned bool
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

	return notes, nil
}

func isHidden(path string) bool {
	return path[0] == 46
}
