package logger

import (
	"log"
	"os"
	"strings"

	"github.com/fatih/color"
)

type LogLevel int

const (
	QuietLevel LogLevel = iota
	DefaultLevel
	VerboseLevel
)

var (
	prefix         = color.New(color.FgCyan).Sprint("[pull-watch] ")
	errorColor     = color.New(color.FgRed).SprintFunc()
	infoColor      = color.New(color.FgGreen).SprintFunc()
	highlightColor = color.New(color.FgHiYellow).Add(color.Bold).SprintFunc()
	warnColor      = color.New(color.FgYellow).SprintFunc()
)

// Logger wraps the standard logger with custom formatting
type Logger struct {
	*log.Logger
	level LogLevel
}

// Option is a functional option for configuring the logger
type Option func(*Logger)

// WithTimestamp enables timestamps in log output
func WithTimestamp() Option {
	return func(l *Logger) {
		l.SetFlags(log.LstdFlags)
	}
}

// WithLogLevel sets the logging level
func WithLogLevel(level LogLevel) Option {
	return func(l *Logger) {
		l.level = level
	}
}

// New creates a new Logger instance with the given options
func New(opts ...Option) *Logger {
	l := &Logger{
		Logger: log.New(os.Stderr, prefix, 0),
		level:  DefaultLevel, // Set default level
	}
	for _, opt := range opts {
		opt(l)
	}
	return l
}

// Warn logs a warning message with yellow color
func (l *Logger) Warn(format string, v ...interface{}) {
	if l.level >= QuietLevel {
		l.Printf(warnColor("WARNING: "+format), v...)
	}
}

// Error logs an error message with red color
func (l *Logger) Error(format string, v ...interface{}) {
	if l.level >= QuietLevel {
		l.Printf(errorColor("ERROR: "+format), v...)
	}
}

// Info logs an info message with green color
func (l *Logger) Info(format string, v ...interface{}) {
	if l.level >= DefaultLevel {
		l.Printf(infoColor(format), v...)
	}
}

// Debug logs a debug message (only in verbose mode)
func (l *Logger) Debug(format string, v ...interface{}) {
	if l.level >= VerboseLevel {
		l.Printf(format, v...)
	}
}

// MultiColor logs a message with multiple color segments
func (l *Logger) MultiColor(level LogLevel, segments ...ColoredSegment) {
	if l.level >= level {
		var parts []string
		for _, seg := range segments {
			parts = append(parts, seg.Color(seg.Text))
		}
		l.Println(strings.Join(parts, ""))
	}
}

// ColoredSegment represents a text segment with its color
type ColoredSegment struct {
	Text  string
	Color func(a ...interface{}) string
}

func ErrorSegment(text string) ColoredSegment {
	return ColoredSegment{Text: text, Color: errorColor}
}

func HighlightSegment(text string) ColoredSegment {
	return ColoredSegment{Text: text, Color: highlightColor}
}

func InfoSegment(text string) ColoredSegment {
	return ColoredSegment{Text: text, Color: infoColor}
}
