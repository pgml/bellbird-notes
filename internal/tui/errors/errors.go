package errors

import "fmt"

type PromptError struct {
	Arg     any
	Message string
}

func IsPromptError(t any) bool {
	switch t.(type) {
	case PromptError:
		return true
	default:
		return false
	}
}

func (e *PromptError) Error() string {
	return fmt.Sprintf("%d - %s", e.Arg, e.Message)
}
