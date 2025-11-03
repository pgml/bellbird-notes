package interfaces

import "bellbird-notes/tui/message"

// Focusable defines behaviour for components that can receive focus
// and respond to common navigation and confirmation actions.
type Focusable interface {
	LineUp() message.StatusBarMsg
	LineDown() message.StatusBarMsg
	GoToTop() message.StatusBarMsg
	GoToBottom() message.StatusBarMsg
	TogglePinnedItems() message.StatusBarMsg
	ConfirmRemove() message.StatusBarMsg
	ConfirmAction() message.StatusBarMsg
	CancelAction(cb func()) message.StatusBarMsg
	Refresh(resetSelectedIndex bool, resetPinned bool) message.StatusBarMsg
	Remove() message.StatusBarMsg
	YankSelection(markCut bool)
	PasteSelectedItems() message.StatusBarMsg
}
