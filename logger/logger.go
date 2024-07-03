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
		output:    color.New(color.FgWhite, color.Faint).FprintlnFunc(),
	}
}

type logger struct {
	appName    string
	errStream  io.Writer
	errorLevel LogLevel
	output     func(w io.Writer, a ...interface{})
}

// SetErrorLevel implements Logger.
func (l *logger) SetErrorLevel(level LogLevel) {
	l.errorLevel = level
}

// Errorf implements Logger.
func (l *logger) Errorf(level LogLevel, format string, a ...any) {
	if level >= l.errorLevel {
		l.output(
			l.errStream,
			l.appName,
			level.String(),
			strings.TrimSpace(fmt.Sprintf(format, a...)),
		)
	}
}
