package app

import (
	"log"
	"os"
	"time"
)

const logFile = "app.log"

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
	file, err := os.OpenFile(
		logFile,
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
