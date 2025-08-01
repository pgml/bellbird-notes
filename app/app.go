package app

import (
	"flag"
	"os"
	"path/filepath"

	"bellbird-notes/app/debug"
)

var NoNerdFonts = flag.Bool("no-nerd-fonts", false, "Disable nerd fonts")
var Debug = flag.Bool("debug", false, "Debug mode")
var DirTreeInfo = flag.Bool("tree-info", false, "Show additional info in the directory tree")
var ShowVersion = flag.Bool("version", false, "Shows the version")

func IsDev() bool {
	return os.Getenv("CHANNEL") == "dev"
}

func Name() string {
	name := "Bellbird Notes"

	return name
}

// Huh?
func ModuleName() string {
	moduleName := "bellbird-notes"
	if channel := os.Getenv("CHANNEL"); channel != "" {
		moduleName += "-" + channel
	}
	//if IsDev() {
	//	moduleName += "-dev"
	//}

	return moduleName
}

// NotesRootDir returns the root directory for notes
func NotesRootDir() (string, error) {
	Home, err := os.UserHomeDir()
	if err != nil {
		debug.LogErr(err)
		return "", err
	}

	appName := ModuleName()
	notesDir := filepath.Join(Home, "."+appName)

	if _, err := os.Stat(notesDir); err != nil {
		os.Mkdir(notesDir, 0755)
	}

	return notesDir, nil
}

// ConfigDir returns the config directory
func ConfigDir() (string, error) {
	ConfigDir, err := os.UserConfigDir()
	if err != nil {
		msg := "Could not get config directory in config.go/ConfigDir()"
		debug.LogErr(msg, err)
		return "", err
	}

	appName := ModuleName()
	confDir := filepath.Join(ConfigDir, appName)

	if _, err := os.Stat(confDir); err != nil {
		os.Mkdir(confDir, 0755)
	}

	return confDir, nil
}

// ConfigFile returns the path to the config file
func ConfigFile(isMetaInfo bool) (string, error) {
	filename := ModuleName()
	if isMetaInfo {
		filename += "_metainfos"
	} else {
		filename += ".conf"
	}

	configDir, err := ConfigDir()
	if err != nil {
		msg := "Could not get config dir in config.go/ConfigFile"
		debug.LogErr(msg, err)
		return "", err
	}

	configFile := filepath.Join(configDir, filename)

	return configFile, nil
}

func IsFlagPassed(name string) bool {
	found := false
	flag.Visit(func(f *flag.Flag) {
		if f.Name == name {
			found = true
		}
	})
	return found
}
