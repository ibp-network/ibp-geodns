package logging

import (
	"fmt"
	"log"
	"os"
	"strings"
)

var logger *log.Logger
var logLevel LogLevel = Info

func init() {
	logger = log.New(os.Stdout, "", log.LstdFlags|log.LUTC)
	Log(Debug, "Logging Package initializing...")
}

func SetLogLevel(level LogLevel) {
	logLevel = level
}

func Log(level LogLevel, format string, v ...interface{}) {
	if level >= logLevel {
		msg := fmt.Sprintf(format, v...)
		logger.Printf("%s: %s", level.String(), msg)
	}
}

func Fmt(format string, v ...interface{}) error {
	return fmt.Errorf(format, v...)
}

func ParseLogLevel(levelStr string) LogLevel {
	switch strings.ToLower(levelStr) {
	case "debug":
		return Debug
	case "info":
		return Info
	case "warn":
		return Warn
	case "error":
		return Error
	case "fatal":
		return Fatal
	default:
		return Info
	}
}
