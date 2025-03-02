package directories

import (
	"bellbird-notes/internal/app"
	"bellbird-notes/internal/tui/errors"
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

func List(dirPath string) ([]Directory, error) {
	var Directories []Directory

	dirs, err := os.ReadDir(dirPath)
	if err != nil {
		app.LogErr(err)
		return nil, err
	}

	for _, child := range dirs {
		filePath := filepath.Join(dirPath, child.Name())
		if !child.IsDir() || isHidden(filePath) {
			continue
		}

		nbrDirs, _ := List(filePath)
		Directories = append(Directories, Directory{
			Name:       child.Name(),
			Path:       filePath,
			NbrNotes:   0,
			NbrFolders: len(nbrDirs),
			IsExpanded: false,
		})
	}

	return Directories, nil
}

func ContainsDir(path string, dirName string) (error, bool) {
	dirs, err := List(path)
	if err != nil {
		return err, false
	}

	for _, dir := range dirs {
		if dir.Name == dirName {
			return nil, true
		}
	}
	return nil, false
}

func Create(path string) error {
	if err := os.Mkdir(path, 0755); err != nil {
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

func Delete(path string, deleteContent bool) error {
	if _, err := os.Stat(path); err != nil {
		app.LogErr(err)
		return err
	}

	if !deleteContent {
		if err := os.Remove(path); err != nil {
			app.LogErr(err)
			return &errors.PromptError{Arg: path, Message: err.Error()}
		}
	} else {
		if err := os.RemoveAll(path); err != nil {
			app.LogErr(err)
			return err
		}
	}
	return nil
}

func isHidden(path string) bool {
	return path[0] == 46
}
