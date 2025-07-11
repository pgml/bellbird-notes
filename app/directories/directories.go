package directories

import (
	"os"
	"path/filepath"

	"bellbird-notes/app/config"
	"bellbird-notes/app/debug"
	"bellbird-notes/app/utils"
	"bellbird-notes/tui/bb_errors"
)

type Directory struct {
	name string
	Path string

	// NbrNotes is the amount of notes in this directory
	NbrNotes int

	// NbrFolders is the amount of sub directories in this directory
	NbrFolders int

	// IsExpanded is the expanded state of this directory
	IsExpanded bool
}

// Name returns the name of the directory
func (d Directory) Name() string {
	return d.name
}

// List returns a list of Directory objects in the given directory path.
func List(dirPath string) ([]Directory, error) {
	var Directories []Directory

	dirs, err := os.ReadDir(dirPath)
	if err != nil {
		debug.LogErr(err)
		return nil, err
	}

	conf := config.New()

	for _, child := range dirs {
		filePath := filepath.Join(dirPath, child.Name())

		if !child.IsDir() || isHidden(child.Name()) {
			continue
		}

		// count number of notes
		nbrNotes, err := GetFileCount(filePath)
		if err != nil {
			debug.LogErr(err)
			return nil, err
		}

		// Get subdirectories to count folders
		nbrDirs, _ := List(filePath)

		// check if expanded
		exp, _ := conf.MetaValue(filePath, config.Expanded)
		expanded := exp == "true"

		Directories = append(Directories, Directory{
			name:       child.Name(),
			Path:       filePath,
			NbrNotes:   nbrNotes,
			NbrFolders: len(nbrDirs),
			IsExpanded: expanded,
		})
	}

	// Sort directory list aphabetically
	utils.SortSliceAsc(Directories, false, nil)

	return Directories, nil
}

// GetFileCount returns the number of files in the given directory path
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

// ContainsDir checks whether a directory with the specified
// name exists in the given path.
func ContainsDir(path string, dirName string) (error, bool) {
	dirs, err := List(path)
	if err != nil {
		return err, false
	}

	for _, dir := range dirs {
		if dir.Name() == dirName {
			return nil, true
		}
	}
	return nil, false
}

// Create creates a new directory at the given path.
func Create(path string) error {
	if err := os.Mkdir(path, 0755); err != nil {
		debug.LogErr(err)
		return err
	}
	return nil
}

// Rename renames or moves a file or directory from oldPath to newPath.
func Rename(oldPath string, newPath string) error {
	if err := os.Rename(oldPath, newPath); err != nil {
		debug.LogErr(err)
		return err
	}
	return nil
}

// Delete deletes the specified directory
// If deleteContent is false, the directory must be empty.
// If true, it deletes the directory and all its contents recursively.
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

// isHidden returns true if the given filename or path starts with a dot ('.')
func isHidden(path string) bool {
	return path[0] == 46
}
