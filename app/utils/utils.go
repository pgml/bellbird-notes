package utils

import (
	"net/url"
	"os"
	"path/filepath"
	"slices"
	"sort"
	"strings"

	"bellbird-notes/app"
	"bellbird-notes/app/debug"

	"github.com/charmbracelet/lipgloss/v2"
)

// HasName is an interface for types that expose a Name method.
// Used to generically sort any slice of such types.
type HasName interface {
	Name() string
}

// TruncateText shortens the given text to fit within maxWidth.
// If the text exceeds maxWidth, it appends "..." (if possible).
func TruncateText(text string, maxWidth int) string {
	if lipgloss.Width(text) > maxWidth {
		if maxWidth > 3 {
			return text[:maxWidth-3] + "..."
		}
		return text[:maxWidth] // No space for "..."
	}
	return text
}

// GetSortedKeys returns the keys of a map[int]T in ascending order.
func GetSortedKeys[T any](mapToSort map[int]T) []int {
	var keys []int
	for key := range mapToSort {
		keys = append(keys, key)
	}
	sort.Ints(keys)

	return keys
}

// SortSliceAsc sorts a slice of items that implement HasName
// in ascending (case-insensitive) order.
// If skipFirst is true, the first element is left unsorted
// (e.g., for keeping a "default" item first).
// Optionally, a setIndex callback can be used to update the index after sorting.
func SortSliceAsc[T HasName](slice []T, skipFirst bool, setIndex func(*T, int)) {
	if len(slice) <= 0 {
		return
	}

	start := 0
	if skipFirst {
		start = 1
	}

	slices.SortFunc(slice[start:], func(i, j T) int {
		return strings.Compare(strings.ToLower(i.Name()), strings.ToLower(j.Name()))
	})

	if setIndex != nil {
		for i := range slice {
			setIndex(&slice[i], i)
		}
	}
}

// RelativePath removes the root notes directory from the full path,
// Optionally includes trailing slash.
func RelativePath(path string, trailingSlash bool) string {
	rootDir, _ := app.NotesRootDir()

	if trailingSlash {
		pathSeparator := string(os.PathSeparator)
		rootDir = rootDir + pathSeparator
	}

	relPath := strings.ReplaceAll(path, rootDir, "")
	return filepath.FromSlash(relPath)
}

func PathFromUrl(path string) string {
	u, err := url.Parse(path)

	if err != nil {
		debug.LogErr(u)
	}

	p, err := url.PathUnescape(u.Path)

	if err != nil {
		debug.LogErr(err)
		return ""
	}

	return filepath.FromSlash(p)
}
