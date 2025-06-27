package app

import "fmt"

var (
	Version    = "0.00"
	DevVersion = ""
)

func PrintVersion() {
	if DevVersion != "" {
		Version += "-dev." + DevVersion
	}
	fmt.Printf("%s Version: %s\n", Name(), Version)
}
