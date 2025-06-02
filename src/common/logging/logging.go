package logging

// Import Dependencies
import (
	"fmt"
	"log"
	"os"
	"strings"
)

// logger is the internal logger instance
var logger *log.Logger

// logLevel sets the current logging level
var logLevel LogLevel = Info

// init initializes the logger
func init() {
	logger = log.New(os.Stdout, "", log.LstdFlags|log.LUTC)
	Log(Debug, "Logging Package initializing...")
}

// SetLogLevel sets the logging level (e.g., Debug, Info, Warn, Error, Fatal)
func SetLogLevel(level LogLevel) {
	logLevel = level
}

// Log logs a message at the specified log level
func Log(level LogLevel, format string, v ...interface{}) {
	if level >= logLevel {
		msg := fmt.Sprintf(format, v...)
		logger.Printf("%s: %s", level.String(), msg)
	}
}

// Fmt formats an error message similarly to fmt.Errorf
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
		// Default to Info if unknown string
		return Info
	}
}
