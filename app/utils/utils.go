package utils

import (
	"slices"
	"sort"
	"strings"

	"github.com/charmbracelet/lipgloss"
)

type HasName interface {
	GetName() string
}

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

func SortSliceAsc[T HasName](slice []T, skipFirst bool, setIndex func(*T, int)) {
	if len(slice) <= 0 {
		return
	}

	start := 0
	if skipFirst {
		start = 1
	}

	slices.SortFunc(slice[start:], func(i, j T) int {
		return strings.Compare(strings.ToLower(i.GetName()), strings.ToLower(j.GetName()))
	})

	if setIndex != nil {
		for i := range slice {
			setIndex(&slice[i], i)
		}
	}
}

func Pointer[T any](d T) *T {
	return &d
}
