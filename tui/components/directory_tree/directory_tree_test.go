package directorytree_test

import (
	"testing"

	"bellbird-notes/app/config"
	directorytree "bellbird-notes/tui/components/directory_tree"
	"bellbird-notes/tui/shared"
)

func TestDirTreeMoveLines(t *testing.T) {
	conf := config.New()
	dirTree := directorytree.New("Folders", conf)

	// Move to the top if we're not already to don't screw up test results
	// with the latest dir index from the config
	if dirTree.SelectedIndex > 0 {
		dirTree.GoToTop()
	}

	// --- TEST LINE DOWN

	// Save current line index
	currentRowIndex := dirTree.SelectedIndex
	// attempt to move one line down
	dirTree.LineDown()

	if dirTree.SelectedIndex == currentRowIndex {
		t.Fatalf(
			"Expected line to move down to index %d, but is %d",
			currentRowIndex+1, dirTree.SelectedIndex,
		)
	}

	// -- TEST LINE UP

	// Save current line index
	currentRowIndex = dirTree.SelectedIndex
	// attempt to move one line down
	dirTree.LineUp()

	if dirTree.SelectedIndex == currentRowIndex {
		t.Fatalf(
			"Expected line to move up to index %d, but is %d",
			currentRowIndex-1, dirTree.SelectedIndex,
		)
	}

	// -- TEST GO TO BOTTOM

	// Save current line index
	currentRowIndex = dirTree.SelectedIndex
	// Attempt to go to bottom
	dirTree.GoToBottom()

	if dirTree.SelectedIndex == currentRowIndex {
		t.Fatalf(
			"Expected line to move to the bottom but is still at index %d",
			dirTree.SelectedIndex,
		)
	}

	// -- TEST GO TO TOP

	// Save current line index
	currentRowIndex = dirTree.SelectedIndex
	// Attempt to go to bottom
	dirTree.GoToTop()

	if dirTree.SelectedIndex == currentRowIndex {
		t.Fatalf(
			"Expected line to move to the top but is still at index %d",
			dirTree.SelectedIndex,
		)
	}
}

// This test only tests if the rename action works on a ui level
// not an actual renaming process on file level
func TestDirTreeRenameAndConfirm(t *testing.T) {
	conf := config.New()
	dirTree := directorytree.New("Folders", conf)

	if dirTree.SelectedIndex != 1 {
		dirTree.GoToTop()
		dirTree.LineDown()
	}

	curName := dirTree.SelectedDir().Name()
	dirTree.Rename(curName)

	if dirTree.EditState != shared.EditStateRename {
		t.Fatalf(
			"Expected EditState to be %d, but is: %d",
			shared.EditStates.Rename,
			dirTree.EditState,
		)
	}

	dirTree.ConfirmAction()

	if dirTree.EditState != shared.EditStateNone {
		t.Fatalf(
			"Expected EditState to be %d, but is: %d",
			shared.EditStates.Rename,
			dirTree.EditState,
		)
	}
}

func TestDirTreeExpandCollapse(t *testing.T) {
	conf := config.New()
	dirTree := directorytree.New("Folders", conf)

	// Move to top since top directory is always expandable
	if dirTree.SelectedIndex > 0 {
		dirTree.GoToTop()
	}

	dirTree.Collapse()
	e, _ := conf.MetaValue(dirTree.SelectedDir().Path(), config.Expanded)
	if dirTree.SelectedDir().Expanded() || e == "true" {
		t.Fatalf("Directory should be collapsed but isn't")
	}

	dirTree.Expand()
	e, _ = conf.MetaValue(dirTree.SelectedDir().Path(), config.Expanded)
	if !dirTree.SelectedDir().Expanded() || e == "false" {
		t.Fatalf("Directory should be expanded but isn't")
	}
}
