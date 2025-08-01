package debug

import (
	"log"
	"os"
	"path/filepath"
	"time"
)

const appLogFile = "app.log"
const errorLogFile = "error.log"

type ErrorLvl int

const (
	Info ErrorLvl = iota
	Debug
	Warn
	Error
)

var errLvl = map[ErrorLvl]string{
	Info:  "INFO",
	Debug: "DEBUG",
	Warn:  "WARN",
	Error: "ERROR",
}

func (e ErrorLvl) String() string {
	return errLvl[e]
}

func LogInfo(args ...any) {
	logMsg(Info, args)
}

func LogDebug(args ...any) {
	logMsg(Debug, args)
}

func LogWarn(args ...any) {
	logMsg(Warn, args)
}

func LogErr(args ...any) {
	logMsg(Error, args)
}

func logMsg(level ErrorLvl, args ...any) {
	configDir, err := ConfigDir()
	if err != nil {
		return
	}

	logFile := appLogFile
	if level == Error {
		logFile = errorLogFile
	}

	file, err := os.OpenFile(
		configDir+"/"+logFile,
		os.O_APPEND|os.O_CREATE|os.O_WRONLY,
		0644,
	)
	if err != nil {
		log.Fatal(err)
	}
	defer file.Close()

	log.SetOutput(file)
	log.Printf(
		"[%s] %s: %s\n",
		time.Now().Format("00:00:00"),
		level.String(), args,
	)
}

// this is lazy and stupid because it's duplicate of the same method in config.go
// but i am too tired to deal with import cycles right now.
// @todo make it not stupid
func ConfigDir() (string, error) {
	ConfigDir, err := os.UserConfigDir()
	if err != nil {
		msg := "Could not get config directory in config.go/ConfigDir()"
		LogErr(msg, err)
		return "", err
	}

	configDir := ConfigDir
	appName := "bellbird-notes"
	if channel := os.Getenv("CHANNEL"); channel != "" {
		configDir += "-" + channel
		appName += "-" + channel
	}

	confDir := filepath.Join(ConfigDir, appName)

	if _, err := os.Stat(confDir); err != nil {
		os.Mkdir(confDir, 0755)
	}

	return confDir, nil
}
