package notes_test

import (
	"os"
	"path/filepath"
	"testing"

	"bellbird-notes/app/notes"
)

func TestNewNote(t *testing.T) {
	n := notes.NewNote("./test.txt", true)

	if n.Name() != "test" {
		t.Errorf("Expected note name to be 'test', got '%s'", n.Name())
	}
	if !n.IsPinned {
		t.Error("Expected IsPinned to be true")
	}
}

func TestCreateWriteDelete(t *testing.T) {
	tmp := t.TempDir()
	path := filepath.Join(tmp, "test_note.txt")

	_, err := notes.Create(path)
	if err != nil {
		t.Fatalf("Create failed: %v", err)
	}

	_, err = os.Stat(path)
	if err != nil {
		t.Fatalf("Expected file at %s, but got error :%v", path, err)
	}

	content := "TEST"
	n, err := notes.Write(path, content)
	if err != nil || n != len(content) {
		t.Fatalf("Write failed: %v", err)
	}

	err = notes.Delete(path)
	if err != nil {
		t.Fatalf("Delete failed: %v", err)
	}
	if notes.Exists(path) {
		t.Error("Note should be deleted")
	}
}

func TestRename(t *testing.T) {
	dir := t.TempDir()
	oldPath := filepath.Join(dir, "old.txt")
	newPath := filepath.Join(dir, "new.txt")

	_, err := notes.Create(oldPath)
	if err != nil {
		t.Fatalf("Create failed: %v", err)
	}

	err = notes.Rename(oldPath, newPath)
	if err != nil {
		t.Fatalf("Rename failed: %v", err)
	}

	if !notes.Exists(newPath) {
		t.Error("Renamed file does not exist")
	}
	if notes.Exists(oldPath) {
		t.Error("Old file still exists")
	}
}

func TestListFiltersOnlyNotes(t *testing.T) {
	dir := t.TempDir()

	os.WriteFile(filepath.Join(dir, "test_note.txt"), []byte("test"), 0644)
	os.WriteFile(filepath.Join(dir, ".hidden.txt"), []byte("nope"), 0644)
	os.WriteFile(filepath.Join(dir, "image.png"), []byte("png"), 0644)

	notesList, err := notes.List(dir)
	if err != nil {
		t.Fatalf("List failed: %v", err)
	}

	if len(notesList) != 1 {
		t.Fatalf("Expected 1 note, got %d", len(notesList))
	}

	if notesList[0].Name() != "test_note" {
		t.Errorf("Expected 'test_note', got '%s'", notesList[0].Name())
	}

	if notesList[0].NameWithExt() != "test_note.txt" {
		t.Errorf("Expected 'test_note.txt', got '%s'", notesList[0].Name())
	}
}
