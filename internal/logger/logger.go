package logger

import (
	"log"
	"os"

	"github.com/fatih/color"
)

var (
	prefix     = color.New(color.FgCyan).Sprint("[pull-watch] ")
	errorColor = color.New(color.FgRed).SprintFunc()
	infoColor  = color.New(color.FgGreen).SprintFunc()
)

// Logger wraps the standard logger with custom formatting
type Logger struct {
	*log.Logger
}

// New creates a new Logger instance
func New() *Logger {
	return &Logger{
		Logger: log.New(os.Stderr, prefix, log.LstdFlags),
	}
}

// Error logs an error message with red color
func (l *Logger) Error(format string, v ...interface{}) {
	l.Printf(errorColor("ERROR: "+format), v...)
}

// Info logs an info message with green color
func (l *Logger) Info(format string, v ...interface{}) {
	l.Printf(infoColor(format), v...)
}
