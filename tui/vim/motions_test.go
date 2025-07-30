package vim

import (
	"bellbird-notes/tui/components"
	"bellbird-notes/tui/keyinput"
	"bellbird-notes/tui/mode"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func createTestApp(t *testing.T) (*Vim, *components.App) {
	// create a file to test editor line down
	tmp := t.TempDir()
	path := filepath.Join(tmp, "test_note.txt")

	content := "TEST1\nTest2\nTest3\ntest4\ntes5t"
	if err := os.WriteFile(path, []byte(content), 0644); err != nil {
		t.Fatalf("Write failed: %v", err)
	}

	// Simulate starting the application
	vim := &Vim{}
	app := components.NewApp(vim)
	vim.SetApp(app)

	app.KeyInput = keyinput.New(vim)
	app.KeyInput.Components = []keyinput.FocusedComponent{
		app.DirTree,
		app.NotesList,
		app.Editor,
		app.BufferList,
	}
	vim.KeyMap = app.KeyInput

	// Focus editor
	editor := app.Editor
	editor.SetFocus(true)
	editor.SetBuffers(&app.Buffers)
	app.BufferList.SetBuffers(&app.Buffers)

	// Create a buffer
	editor.NewBuffer(path)
	editor.Buffers = &components.Buffers{}

	return vim, app
}

func TestLineUpDown(t *testing.T) {
	vim, app := createTestApp(t)

	buf := app.Editor.CurrentBuffer
	row := buf.CursorPos.Row

	var opts keyinput.Options

	// --- TEST LINE UP

	row = buf.CursorPos.Row
	vim.lineDown(opts)()

	if row == buf.CursorPos.Row {
		t.Fatalf("Expected line to move up to index %d, but is %d",
			row,
			buf.CursorPos.Row,
		)
	}

	// --- TEST LINE DOWN

	vim.lineDown(opts)()

	if row == buf.CursorPos.Row {
		t.Fatalf("Expected line to move down to index %d, but is %d",
			row,
			buf.CursorPos.Row,
		)
	}

}

func TestGoToTopBottom(t *testing.T) {
	vim, app := createTestApp(t)

	buf := app.Editor.CurrentBuffer
	var opts keyinput.Options

	// -- TEST GO TO BOTTOM

	vim.goToBottom(opts)()

	nbrLines := strings.Count(buf.Content, "\n")
	if buf.CursorPos.Row != nbrLines {
		t.Fatalf("Expected line to be index %d, but is %d",
			nbrLines,
			buf.CursorPos.Row,
		)
	}

	// -- TEST GO TO BOTTOM

	vim.goToTop(opts)()

	if buf.CursorPos.Row != 0 {
		t.Fatalf("Expected line to be index 0, but is %d",
			buf.CursorPos.Row,
		)
	}
}

func TestFocusColumn(t *testing.T) {
	vim, app := createTestApp(t)

	var opts keyinput.Options

	// -- TEST FOCUS NEXT COLUMN

	col := app.CurrColFocus
	vim.focusNextColumn(opts)()

	if app.CurrColFocus == col {
		t.Fatalf("Expected column to be index %d, but is %d",
			col+1,
			app.CurrColFocus,
		)
	}

	// -- TEST FOCUS PREV COLUMN

	col = app.CurrColFocus
	vim.focusPrevColumn(opts)()

	if app.CurrColFocus == col {
		t.Fatalf("Expected column to be index %d, but is %d",
			col-1,
			app.CurrColFocus,
		)
	}
}

//func TestRename(t *testing.T) {}
//func TestDelete(t *testing.T) {}
//func TestYankListItem(t *testing.T) {}
//func TestCutListItem(t *testing.T) {}
//func TestPasteListItem(t *testing.T) {}
//func TestTogglePinListItem(t *testing.T) {}
//func TestCreateDir(t *testing.T) {}
//func TestCreateNote(t *testing.T) {}

func TestEnterCmdMode(t *testing.T) {
	vim, app := createTestApp(t)
	vim.enterCmdMode(keyinput.Options{})()

	if app.Mode.Current != mode.Command {
		t.Fatalf("Expected mode to be %s, but is %s",
			mode.Command.FullString(false),
			app.Mode.Current.FullString(false),
		)
	}
}

func TestOpenCloseBufferList(t *testing.T) {
	vim, app := createTestApp(t)

	// -- TEST OPEN BUFFER LIST

	vim.showBufferList(keyinput.Options{})()

	if !app.BufferList.Focused() && !app.Editor.ListBuffers {
		t.Fatal("Expected buffer list to be focused but isn't")
	}

	// -- TEST CLOSE BUFFER LIST

	vim.closeBufferList(keyinput.Options{})()

	if app.BufferList.Focused() && app.Editor.ListBuffers {
		t.Fatal("Expected buffer list to be closed but isn't")
	}
}

//func TestConfirmAction(t *testing.T) {}
//func TestCancelAction(t *testing.T) {}

func TestNewScratchBuffer(t *testing.T) {
	vim, app := createTestApp(t)
	vim.newScratchBuffer(keyinput.Options{})()

	if !app.Editor.CurrentBuffer.IsScratch {
		t.Fatal("Expected buffer to be a scratch buffer but isn't")
	}
}

func TestEnterNormalMode(t *testing.T) {
	vim, app := createTestApp(t)
	vim.enterNormalMode(keyinput.Options{})()

	if app.Mode.Current != mode.Normal {
		t.Fatalf("Expected mode to be %s, but is %s",
			mode.Normal.FullString(false),
			app.Mode.Current.FullString(false),
		)
	}
}

func TestEnterToggleVisualMode(t *testing.T) {
	vim, app := createTestApp(t)

	// -- TEST ENTER VISUAL MODE

	vim.toggleVisual(keyinput.Options{})()

	if app.Editor.Mode.Current != mode.Visual {
		t.Fatalf("Expected mode to be %s, but is %s",
			mode.Visual.FullString(false),
			app.Editor.Mode.Current.FullString(false),
		)
	}

	// -- TEST ENTER NORMAL MODE FROM VISUAL MODE

	vim.toggleVisual(keyinput.Options{})()

	if app.Editor.Mode.Current != mode.Normal {
		t.Fatalf("Expected mode to be %s, but is %s",
			mode.Normal.FullString(false),
			app.Editor.Mode.Current.FullString(false),
		)
	}
}

func TestEnterToggleVisualLineMode(t *testing.T) {
	vim, app := createTestApp(t)
	vim.toggleVisualLine(keyinput.Options{})()

	if app.Editor.Mode.Current != mode.VisualLine {
		t.Fatalf("Expected mode to be %s, but is %s",
			mode.VisualLine.FullString(false),
			app.Editor.Mode.Current.FullString(false),
		)
	}

	vim.toggleVisualLine(keyinput.Options{})()

	if app.Editor.Mode.Current != mode.Normal {
		t.Fatalf("Expected mode to be %s, but is %s",
			mode.Normal.FullString(false),
			app.Editor.Mode.Current.FullString(false),
		)
	}
}

//func TestEnterToggleVisualBlockMode(t *testing.T) {
//	vim, app := createTestApp(t)
//	vim.toggleVisualBlock(keyinput.Options{})()
//
//	if app.Editor.Mode.Current != mode.VisualBlock {
//		t.Fatalf("Expected mode to be %s, but is %s",
//			mode.VisualBlock.FullString(false),
//			app.Editor.Mode.Current.FullString(false),
//		)
//	}
//
//	vim.toggleVisualBlock(keyinput.Options{})()
//
//	if app.Editor.Mode.Current != mode.Normal {
//		t.Fatalf("Expected mode to be %s, but is %s",
//			mode.Normal.FullString(false),
//			app.Editor.Mode.Current.FullString(false),
//		)
//	}
//}

func TestEnterInsertMode(t *testing.T) {
	vim, app := createTestApp(t)
	vim.enterInsertMode(keyinput.Options{})()

	if app.Editor.Mode.Current != mode.Insert {
		t.Fatalf("Expected mode to be %s, but is %s",
			mode.Insert.FullString(false),
			app.Mode.Current.FullString(false),
		)
	}
}

func TestInsertBelow(t *testing.T) {
	vim, app := createTestApp(t)

	buf := app.Editor.CurrentBuffer
	ta := app.Editor.Textarea

	vim.insertBelow(keyinput.Options{})()

	if len(ta.Val()[buf.CursorPos.Row+1]) != 0 {
		t.Fatalf(
			"An empty line should be at position: %d, but isn't",
			buf.CursorPos.Row+1,
		)
	}
}

func TestInsertAbove(t *testing.T) {
	vim, app := createTestApp(t)

	buf := app.Editor.CurrentBuffer
	ta := app.Editor.Textarea

	vim.lineDown(keyinput.Options{})()
	vim.insertAbove(keyinput.Options{})()

	if len(ta.Val()[buf.CursorPos.Row]) != 0 {
		t.Fatalf(
			"An empty line should be at position: %d, but isn't",
			buf.CursorPos.Row,
		)
	}
}

// func TestSelectWord(t *testing.T) {}
// func TestNextWord(t *testing.T) {}
// func TestPrevWord(t *testing.T) {}
// func TestFindCharachter(t *testing.T) {}
// func TestFind(t *testing.T) {}
// func TestMoveToMatch(t *testing.T) {}
// func TestDeleteWord(t *testing.T) {}
// func TestDeleteAfterCursor(t *testing.T) {}
// func TestDeleteSelection(t *testing.T) {}
// func TestDeleteCharacter(t *testing.T) {}
// func TestDeleteFromCursorToChar(t *testing.T) {}
// func TestSubstituteText(t *testing.T) {}
// func TestChangeAfterCursor(t *testing.T) {}
// func TestChangeLine(t *testing.T) {}
// func TestChangeWord(t *testing.T) {}
// func TestYankSelection(t *testing.T) {}
// func TestPaste(t *testing.T) {}
