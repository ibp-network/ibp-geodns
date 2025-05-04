package logging

// LogLevel represents the severity level of the log message
type LogLevel int

const (
	// Debug level for detailed debugging messages
	Debug LogLevel = iota
	// Info level for informational messages
	Info
	// Warn level for warnings
	Warn
	// Error level for error messages
	Error
	// Fatal level for critical errors causing premature termination
	Fatal
)

// String returns the string representation of the LogLevel
func (level LogLevel) String() string {
	switch level {
	case Debug:
		return "DEBUG"
	case Info:
		return "INFO"
	case Warn:
		return "WARN"
	case Error:
		return "ERROR"
	case Fatal:
		return "FATAL"
	default:
		return "UNKNOWN"
	}
}
