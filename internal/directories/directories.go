package directories

import (
	"bellbird-notes/internal/app"
	"os"
)

type Directory struct {
	Name        string
	Path        string
	Children    []Directory
	NbrNotes    int
	NbrFolders  int
	IsExpanded   bool
}

func List(dirPath string) []Directory {
	var Directories []Directory

	dirs, err := os.ReadDir(dirPath)
	if err != nil {
		app.LogErr(err)
	}

	for _, child := range dirs {
		filePath := dirPath + "/" + child.Name()
		if !child.IsDir() || isHidden(filePath) {
			continue
		}

		Directories = append(Directories, Directory{
			Name: child.Name(),
			Path: filePath,
			//Children: List(filePath),
			Children: []Directory{},
			NbrNotes: 0,
			NbrFolders: len(List(filePath)),
			IsExpanded: true,
		})
	}

	return Directories
}

func isHidden(path string) bool {
	return path[0] == 46
}
