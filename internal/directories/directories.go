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

func List(dirPath string) []Directory {
	var Directories []Directory

	dirs, err := os.ReadDir(dirPath)
	if err != nil {
		app.LogErr(err)
	}

	for _, child := range dirs {
		file_path := filepath.Join(dirPath, child.Name())
		if !child.IsDir() || isHidden(file_path) {
			continue
		}

		Directories = append(Directories, Directory{
			Name:       child.Name(),
			Path:       file_path,
			NbrNotes:   0,
			NbrFolders: len(List(file_path)),
			IsExpanded: false,
		})
	}

	return Directories
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
