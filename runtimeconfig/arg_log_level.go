package runtimeconfig

import (
	"fmt"
	"strings"

	"github.com/jakewan/go-procrotator/logger"
)

type logStream int

const (
	outStream logStream = iota
	errStream
)

func (l logStream) String() string {
	return [...]string{"stdout", "stderr"}[l]
}

type argLogLevel struct {
	stream logStream
	value  *logger.LogLevel
}

// name implements argDef.
func (a argLogLevel) name() string {
	switch a.stream {
	case outStream:
		return "loglevel"
	case errStream:
		return "errorloglevel"
	}
	panic(fmt.Sprintf("unexpected stream setting: %d", a.stream))
}

// stringFunc implements argDef.
func (a argLogLevel) stringFunc() func(string) error {
	return func(s string) error {
		switch s {
		case logger.NOTSET.String():
			*a.value = logger.NOTSET
		case logger.DEBUG.String():
			*a.value = logger.DEBUG
		case logger.INFO.String():
			*a.value = logger.INFO
		case logger.NOTICE.String():
			*a.value = logger.NOTICE
		case logger.WARNING.String():
			*a.value = logger.WARNING
		case logger.ERROR.String():
			*a.value = logger.ERROR
		default:
			return fmt.Errorf(
				"invalid log level. expected one of: %s (got %s)",
				strings.Join(allLogLevelStrings(), ", "),
				s,
			)
		}
		return nil
	}
}

// usage implements argDef.
func (a argLogLevel) usage() string {
	return fmt.Sprintf(
		`The log level for %s.

Expected values: %s

The default is INFO.`,
		a.stream.String(),
		strings.Join(allLogLevelStrings(), ", "),
	)
}

func allLogLevelStrings() []string {
	allLevels := logger.AllLevels()
	levelStrings := make([]string, 0, len(allLevels))
	for _, l := range allLevels {
		levelStrings = append(levelStrings, l.String())
	}
	return levelStrings
}
