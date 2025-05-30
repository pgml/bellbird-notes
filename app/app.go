package app

import (
	"flag"
	"os"
	"path/filepath"

	"bellbird-notes/app/debug"

	"golang.org/x/mod/modfile"
)

var NoNerdFonts = flag.Bool("no-nerd-fonts", false, "Nerd fonts disabled")
var Debug = flag.Bool("debug", false, "Debug mode")

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

func ModuleName() (string, error) {
	data, err := os.ReadFile("go.mod")
	if err != nil {
		debug.LogErr(err)
		return "", err
	}

	mod, err := modfile.Parse("go.mod", data, nil)
	if err != nil {
		debug.LogErr(err)
		return "", err
	}

	moduleName := mod.Module.Mod.Path
	if IsSnapshot() {
		moduleName += "-snapshot"
	}

	return moduleName, nil
}

func NotesRootDir() (string, error) {
	Home, err := os.UserHomeDir()
	if err != nil {
		debug.LogErr(err)
		return "", err
	}

	appName, _ := ModuleName()
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

	appName, err := ModuleName()
	if err != nil {
		msg := "Could not get config file in config.go/ConfigFile()"
		debug.LogErr(msg, err)
		return "", err
	}

	confDir := filepath.Join(ConfigDir, appName)

	if _, err := os.Stat(confDir); err != nil {
		os.Mkdir(confDir, 0755)
	}

	return confDir, nil
}

func ConfigFile() (string, error) {
	appName, err := ModuleName()
	if err != nil {
		msg := "Could not get config file in config.go/ConfigFile()"
		debug.LogErr(msg, err)
		return "", err
	}

	configDir, err := ConfigDir()
	if err != nil {
		msg := "Could not get config dir in config.go/ConfigFile"
		debug.LogErr(msg, err)
		return "", err
	}

	configFile := filepath.Join(configDir, appName+".conf")

	return configFile, nil
}
