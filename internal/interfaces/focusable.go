package interfaces

import "bellbird-notes/tui/messages"

// Focusable defines behaviour for components that can receive focus
// and respond to common navigation and confirmation actions.
type Focusable interface {
	LineUp() messages.StatusBarMsg
	LineDown() messages.StatusBarMsg
	GoToTop() messages.StatusBarMsg
	GoToBottom() messages.StatusBarMsg
	ConfirmRemove() messages.StatusBarMsg
	ConfirmAction() messages.StatusBarMsg
	CancelAction(cb func()) messages.StatusBarMsg
	Refresh(resetSelectedIndex bool) messages.StatusBarMsg
	Remove() messages.StatusBarMsg
}
