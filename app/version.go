package app

import "fmt"

var (
	version string
	dev     string
	commit  string
)

func PrintVersion() {
	if dev != "" {
		version += "-dev." + commit
	}

	if commit != "" && dev == "" {
		version += " (" + commit + ")"
	}

	fmt.Printf("%s %s\n", Name(), version)
}
