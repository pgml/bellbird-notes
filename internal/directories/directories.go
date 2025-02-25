package directories

import (
	"bellbird-notes/internal/app"
	"os"
	"path/filepath"
)

type Directory struct {
	Name       string
	Path       string
	NbrNotes   int
	NbrFolders int
	IsExpanded bool
}

func List(dirPath string) []Directory {
	var Directories []Directory

	dirs, err := os.ReadDir(dirPath)
	if err != nil {
		app.LogErr(err)
	}

	for _, child := range dirs {
		filePath := filepath.Join(dirPath, child.Name())
		if !child.IsDir() || isHidden(filePath) {
			continue
		}

		Directories = append(Directories, Directory{
			Name:       child.Name(),
			Path:       filePath,
			NbrNotes:   0,
			NbrFolders: len(List(filePath)),
			IsExpanded: false,
		})
	}

	return Directories
}

func isHidden(path string) bool {
	return path[0] == 46
}
