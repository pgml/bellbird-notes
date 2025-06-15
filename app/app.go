package app

import (
	"flag"
	"os"
	"path/filepath"

	"bellbird-notes/app/debug"
)

var NoNerdFonts = flag.Bool("no-nerd-fonts", false, "Nerd fonts disabled")
var Debug = flag.Bool("debug", false, "Debug mode")
var DirTreeInfo = flag.Bool("tree-info", false, "Debug mode")

func IsSnapshot() bool {
	return os.Getenv("CHANNEL") == "snapshot"
}

func Name() string {
	name := "Bellbird Notes"

	//if IsSnapshot() {
	//	name += " Snapshot"
	//}

	return name
}

// Huh?
func ModuleName() string {
	moduleName := "bellbird-notes"
	if IsSnapshot() {
		moduleName += "-snapshot"
	}

	return moduleName
}

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

func ConfigDir() (string, error) {
	ConfigDir, err := os.UserConfigDir()
	if err != nil {
		msg := "Could not get config directory in config.go/ConfigDir()"
		debug.LogErr(msg, err)
		return "", err
	}

	configDir := ConfigDir
	if IsSnapshot() {
		configDir += "-snapshot"
	}

	appName := ModuleName()
	confDir := filepath.Join(ConfigDir, appName)

	if _, err := os.Stat(confDir); err != nil {
		os.Mkdir(confDir, 0755)
	}

	return confDir, nil
}

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
