package directories

import (
	"os"
	"path/filepath"

	"bellbird-notes/app/debug"
	"bellbird-notes/app/utils"
	"bellbird-notes/tui/bb_errors"
)

type Directory struct {
	Name       string
	Path       string
	NbrNotes   int
	NbrFolders int
	IsExpanded bool
}

func (d Directory) GetName() string {
	return d.Name
}

func List(dirPath string) ([]Directory, error) {
	var Directories []Directory

	dirs, err := os.ReadDir(dirPath)
	if err != nil {
		debug.LogErr(err)
		return nil, err
	}

	for _, child := range dirs {
		filePath := filepath.Join(dirPath, child.Name())
		if !child.IsDir() || isHidden(filePath) {
			continue
		}

		nbrNotes, err := GetFileCount(filePath)
		if err != nil {
			debug.LogErr(err)
			return nil, err
		}

		nbrDirs, _ := List(filePath)
		Directories = append(Directories, Directory{
			Name:       child.Name(),
			Path:       filePath,
			NbrNotes:   nbrNotes,
			NbrFolders: len(nbrDirs),
			IsExpanded: false,
		})
	}

	// Sort directory list aphabetically
	utils.SortSliceAsc(Directories, false, nil)

	return Directories, nil
}

func GetFileCount(dir string) (int, error) {
	dirs, err := os.ReadDir(dir)
	if err != nil {
		debug.LogErr(err)
		return 0, err
	}

	nbrNotes := 0

	for _, child := range dirs {
		if child.IsDir() {
			continue
		}

		nbrNotes++
	}

	return nbrNotes, nil
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
		debug.LogErr(err)
		return err
	}
	return nil
}

func Rename(oldPath string, newPath string) error {
	if err := os.Rename(oldPath, newPath); err != nil {
		debug.LogErr(err)
		return err
	}
	return nil
}

func Delete(path string, deleteContent bool) error {
	if _, err := os.Stat(path); err != nil {
		debug.LogErr(err)
		return err
	}

	if !deleteContent {
		if err := os.Remove(path); err != nil {
			debug.LogErr(err)
			return &bb_errors.PromptError{Arg: path, Message: err.Error()}
		}
	} else {
		if err := os.RemoveAll(path); err != nil {
			debug.LogErr(err)
			return err
		}
	}
	return nil
}

func isHidden(path string) bool {
	return path[0] == 46
}
