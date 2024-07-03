package runtimeconfig

import (
	"fmt"
	"regexp"
	"syscall"

	"github.com/jakewan/go-procrotator/logger"
)

type Config interface {
	fmt.Stringer
	IncludeFileRegexes() []regexp.Regexp
	ExcludeFileRegexes() []regexp.Regexp
	LogLevel() logger.LogLevel
	PreambleCommands() []string
	QuitSignal() syscall.Signal
	ServerCommand() string
	WorkingDirectory() string
}

type config struct {
	logLevel           logger.LogLevel
	workingDirectory   string
	includeFileRegexes []regexp.Regexp
	excludeFileRegexes []regexp.Regexp
	preambleCommands   []string
	serverCommand      string
	quitSignal         syscall.Signal
}

// ExcludeFileRegexes implements Config.
func (c *config) ExcludeFileRegexes() []regexp.Regexp {
	return c.excludeFileRegexes
}

// IncludeFileRegexes implements Config.
func (c *config) IncludeFileRegexes() []regexp.Regexp {
	return c.includeFileRegexes
}

// LogLevel implements cmd.Config.
func (c *config) LogLevel() logger.LogLevel {
	return c.logLevel
}

// String implements cmd.Config.
func (c *config) String() string {
	preambleCommands := make([]string, 0, len(c.preambleCommands))
	for _, s := range c.preambleCommands {
		preambleCommands = append(preambleCommands, fmt.Sprintf("'%s'", s))
	}
	includeFileRegexes := make([]string, 0, len(c.includeFileRegexes))
	for _, r := range c.includeFileRegexes {
		includeFileRegexes = append(
			includeFileRegexes,
			fmt.Sprintf("'%s'", r.String()),
		)
	}
	excludeFileRegexes := make([]string, 0, len(c.excludeFileRegexes))
	for _, r := range c.excludeFileRegexes {
		excludeFileRegexes = append(
			excludeFileRegexes,
			fmt.Sprintf("'%s'", r.String()),
		)
	}
	return fmt.Sprintf(`Config:
  Working directory: %s
  Log level: %s
  Server command: %s
  Preamble commands: %s
  Include file regexes: %s
  Exclude file regexes: %s`,
		c.workingDirectory,
		c.logLevel,
		c.serverCommand,
		preambleCommands,
		includeFileRegexes,
		excludeFileRegexes,
	)
}

// PreambleCommands implements cmd.Config.
func (c *config) PreambleCommands() []string {
	return c.preambleCommands
}

// QuitSignal implements cmd.Config.
func (c *config) QuitSignal() syscall.Signal {
	return c.quitSignal
}

// ServerCommand implements cmd.Config.
func (c *config) ServerCommand() string {
	return c.serverCommand
}

// WorkingDirectory implements cmd.Config.
func (c *config) WorkingDirectory() string {
	return c.workingDirectory
}
