package directories_test

import (
	"os"
	"path/filepath"
	"testing"

	"bellbird-notes/app/directories"
)

func TestCreateDelete(t *testing.T) {
	tmp := t.TempDir()
	dirPath := filepath.Join(tmp, "test_dir")
	subDirPath := filepath.Join(dirPath, "sub_test_dir")

	// create directory
	err := directories.Create(dirPath)
	if err != nil {
		t.Fatalf("Create failed: %v", err)
	}

	// create sub directory
	err = directories.Create(subDirPath)
	if err != nil {
		t.Fatalf("Sub dir Create failed: %v", err)
	}

	// check if test_dir exists
	_, err = os.Stat(dirPath)
	if err != nil {
		t.Fatalf("Expected directory at %s, but got error :%v", dirPath, err)
	}

	// check if sub_test_dir exists
	_, err = os.Stat(subDirPath)
	if err != nil {
		t.Fatalf("Expected directory at %s, but got error :%v", subDirPath, err)
	}

	err, _ = directories.ContainsDir(dirPath, filepath.Base(subDirPath))
	if err != nil {
		t.Fatalf("ContainsDir failed: %v", err)
	}

	err = directories.Delete(dirPath, true)
	if err != nil {
		t.Fatalf("Delete failed: %v", err)
	}
}

func TestRename(t *testing.T) {
	dir := t.TempDir()
	oldPath := filepath.Join(dir, "old_dir")
	newPath := filepath.Join(dir, "new_dir")

	err := directories.Create(oldPath)
	if err != nil {
		t.Fatalf("Create failed: %v", err)
	}

	err = directories.Rename(oldPath, newPath)
	if err != nil {
		t.Fatalf("Rename failed: %v", err)
	}

	err, exists := directories.ContainsDir(dir, filepath.Base(newPath))
	if !exists {
		t.Errorf("Renamed directory does not exist: %v", err)
	}

	_, exists = directories.ContainsDir(dir, filepath.Base(oldPath))
	if exists {
		t.Error("Old directory still exists")
	}
}

func TestList(t *testing.T) {
	dir := t.TempDir()
	os.Mkdir(filepath.Join(dir, "test_dir"), 0755)
	os.Mkdir(filepath.Join(dir, ".hidden_dir"), 0755)

	dirList, err := directories.List(dir)
	if err != nil {
		t.Fatalf("List failed: %v", err)
	}

	if len(dirList) != 1 {
		t.Fatalf("Expected 1 directory, got %d", len(dirList))
	}

	if dirList[0].Name() != "test_dir" {
		t.Errorf("Expected 'test_dir', got '%s'", dirList[0].Name())
	}
}
