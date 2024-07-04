package logger

import (
	"fmt"
	"io"
	"strings"

	"github.com/fatih/color"
)

type LogLevel int

const (
	NOTSET LogLevel = iota
	DEBUG
	INFO
	NOTICE
	WARNING
	ERROR
)

func AllLevels() []LogLevel {
	return []LogLevel{
		NOTSET,
		DEBUG,
		INFO,
		NOTICE,
		WARNING,
		ERROR,
	}
}

func (l LogLevel) String() string {
	return [...]string{"NOTSET", "DEBUG", "INFO", "NOTICE", "WARNING", "ERROR"}[l]
}

func (l LogLevel) EnumIndex() int {
	return int(l)
}

type Logger interface {
	Errorf(level LogLevel, format string, a ...any)
	SetErrorLevel(level LogLevel)
}

func NewLogger(appName string, errStream io.Writer) Logger {
	return &logger{
		appName:   appName,
		errStream: errStream,
	}
}

type logger struct {
	appName    string
	errStream  io.Writer
	errorLevel LogLevel
}

var (
	output        = color.New(color.FgWhite, color.Faint).FprintlnFunc()
	outputError   = color.New(color.FgRed, color.Faint).FprintlnFunc()
	outputWarning = color.New(color.FgYellow, color.Faint).FprintlnFunc()
)

// SetErrorLevel implements Logger.
func (l *logger) SetErrorLevel(level LogLevel) {
	l.errorLevel = level
}

// Errorf implements Logger.
func (l *logger) Errorf(level LogLevel, format string, a ...any) {
	if level >= l.errorLevel {
		fn := output
		switch level {
		case WARNING:
			fn = outputWarning
		case ERROR:
			fn = outputError
		}
		fn(
			l.errStream,
			l.appName,
			level.String(),
			strings.TrimSpace(fmt.Sprintf(format, a...)),
		)
	}
}
