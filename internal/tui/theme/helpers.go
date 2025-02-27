package theme

import (
	"sort"

	"github.com/charmbracelet/lipgloss"
)

func TruncateText(text string, maxWidth int) string {
	if lipgloss.Width(text) > maxWidth {
		if maxWidth > 3 {
			return text[:maxWidth-3] + "..."
		}
		return text[:maxWidth] // No space for "..."
	}
	return text
}

func GetSortedKeys[T any](mapToSort map[int]T) []int {
	var keys []int
	for key := range mapToSort {
		keys = append(keys, key)
	}
	sort.Ints(keys)

	return keys
}
