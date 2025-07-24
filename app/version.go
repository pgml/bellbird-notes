package app

import "fmt"

var (
	version string
	dev     string
	commit  string
)

func PrintVersion() {
	if dev != "" {
		fmt.Printf("%s %s-dev.%s\n", BinaryName(), version, commit)
	}

	if commit != "" && dev == "" {
		fmt.Printf("%s %s (%s)\n", BinaryName(), version, commit)
	}
}

func BinaryName() string {
	return "bbnotes"
}
