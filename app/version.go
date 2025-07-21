package app

import "fmt"

var (
	Version = "0.00"
	Dev     = ""
	Commit  = ""
)

func PrintVersion() {
	if Dev != "" {
		Version += "-dev." + Commit
	}

	if Commit != "" && Dev == "" {
		Version += " (" + Commit + ")"
	}

	fmt.Printf("%s %s\n", Name(), Version)
}
