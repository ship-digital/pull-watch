package logger

import (
	"log"
	"os"
	"strings"

	"github.com/fatih/color"
)

var (
	prefix         = color.New(color.FgCyan).Sprint("[pull-watch] ")
	errorColor     = color.New(color.FgRed).SprintFunc()
	infoColor      = color.New(color.FgGreen).SprintFunc()
	highlightColor = color.New(color.FgYellow).Add(color.Bold).SprintFunc()
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

// MultiColor logs a message with multiple color segments
func (l *Logger) MultiColor(segments ...ColoredSegment) {
	var parts []string
	for _, seg := range segments {
		parts = append(parts, seg.Color(seg.Text))
	}
	l.Println(strings.Join(parts, ""))
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
