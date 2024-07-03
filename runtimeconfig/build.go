package runtimeconfig

import (
	"errors"
	"flag"
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"slices"
	"syscall"

	"github.com/BurntSushi/toml"
	"github.com/jakewan/go-procrotator/logger"
)

var (
	errConfigFileNotFound = errors.New("no config file found")
)

type (
	argDef interface {
		name() string
		usage() string
	}
	argDefWithStringFunc interface {
		argDef
		stringFunc() func(string) error
	}
	tomlConfig struct {
		IncludeFileRegexes []string `toml:"include_file_regexes"`
		ExcludeFileRegexes []string `toml:"exclude_file_regexes"`
		PreambleCommands   []string `toml:"preamble_commands"`
		ServerCommand      string   `toml:"server_command"`
		QuitSignal         string   `toml:"quit_signal"`
		quitSignalInt      syscall.Signal
		LogLevel           string `toml:"log_level"`
	}
)

func configFileNames() []string {
	return []string{
		".procrotator.toml",
		"procrotator.toml",
	}
}

func addFlagsetFuncs(f *flag.FlagSet, a argDefWithStringFunc, aliases ...string) {
	f.Func(a.name(), a.usage(), a.stringFunc())
	for _, alias := range aliases {
		f.Func(
			alias,
			fmt.Sprintf("Alias of -%s", a.name()),
			a.stringFunc(),
		)
	}
}

func addFlagsetStringVar(
	f *flag.FlagSet,
	stringVar *string,
	defaultValue string,
	a argDef,
	aliases ...string,
) {
	f.StringVar(stringVar, a.name(), defaultValue, a.usage())
	for _, alias := range aliases {
		f.StringVar(
			stringVar,
			alias,
			defaultValue,
			fmt.Sprintf("Alias of -%s", a.name()),
		)
	}
}

func addFlagsetStringVarAdder(
	f *flag.FlagSet,
	target *[]string,
	a argDef,
	aliases ...string,
) {
	fn := func(s string) error {
		*target = append(*target, s)
		return nil
	}
	f.Func(a.name(), a.usage(), fn)
	for _, alias := range aliases {
		f.Func(
			alias,
			fmt.Sprintf("Alias of -%s", a.name()),
			fn,
		)
	}
}

func Build(args []string) (Config, error) {
	var (
		logLevel           logger.LogLevel
		wd                 string
		serverCommand      string
		includeFileRegexes []regexp.Regexp
		excludeFileRegexes []regexp.Regexp
		preambleCommands   []string
	)

	// Figure out the working directory first because it would contain any
	// configuration file.
	f := flag.NewFlagSet("go-procrotator", flag.ExitOnError)
	addFlagsetFuncs(f, argDirectory{value: &wd}, "d")
	addFlagsetFuncs(f, argLogLevel{stream: errStream, value: &logLevel}, "l")
	addFlagsetStringVar(f, &serverCommand, "", argServerCommand{}, "s")
	addFlagsetFuncs(
		f,
		argMultiRegex{
			argname: "includefileregexes",
			argusage: `A regular expression matching files to observe.

May be specified multiple times.`,
			regexes: &includeFileRegexes,
		},
		"i",
	)
	addFlagsetFuncs(
		f,
		argMultiRegex{
			argname: "excludefileregexes",
			argusage: `A regular expression matching files to exclude from observation.

May be specified multiple times.`,
			regexes: &excludeFileRegexes,
		},
		"e",
	)
	addFlagsetStringVarAdder(f, &preambleCommands, argPreambleCommand{}, "p")

	if err := f.Parse(args); err != nil {
		return nil, err
	}

	result := config{
		logLevel:         logger.INFO,
		quitSignal:       syscall.SIGINT,
		workingDirectory: wd,
	}

	// Try to find a config file.
	if d, err := readConfigFile(wd); err != nil {
		if errors.Is(err, errConfigFileNotFound) {
			// No configuration file found.
			// Obtain settings from the command line.
			result.serverCommand = serverCommand
			result.includeFileRegexes = includeFileRegexes
			result.excludeFileRegexes = excludeFileRegexes
			result.preambleCommands = preambleCommands
		} else {
			return nil, fmt.Errorf("reading config file: %w", err)
		}
	} else {
		// Load config from the file settings. Then override with any
		// command line arguments that were provided.
		for _, s := range d.IncludeFileRegexes {
			if r, err := regexp.Compile(s); err != nil {
				return nil, fmt.Errorf("parsing include file expressions: %w", err)
			} else {
				result.includeFileRegexes = append(result.includeFileRegexes, *r)
			}
		}
		for _, s := range d.ExcludeFileRegexes {
			if r, err := regexp.Compile(s); err != nil {
				return nil, fmt.Errorf("parsing exclude file expressions: %w", err)
			} else {
				result.excludeFileRegexes = append(result.excludeFileRegexes, *r)
			}
		}
		result.serverCommand = d.ServerCommand
		result.preambleCommands = d.PreambleCommands
		if d.LogLevel != "" {
			if i := slices.IndexFunc(
				logger.AllLevels(),
				func(l logger.LogLevel) bool {
					return l.String() == d.LogLevel
				},
			); i < 0 {
				return nil, fmt.Errorf(
					"config file specifies unexpected log level: %s",
					d.LogLevel,
				)
			} else {
				result.logLevel = logger.AllLevels()[i]
			}
		}
		result.quitSignal = d.quitSignalInt

		// Now check command line arguments.
		if serverCommand != "" {
			result.serverCommand = serverCommand
		}
		if len(preambleCommands) > 0 {
			result.preambleCommands = preambleCommands
		}
		if len(includeFileRegexes) > 0 {
			result.includeFileRegexes = includeFileRegexes
		}
		if len(excludeFileRegexes) > 0 {
			result.excludeFileRegexes = excludeFileRegexes
		}
		if logLevel != logger.NOTSET {
			result.logLevel = logLevel
		}
	}

	if result.serverCommand == "" {
		return nil, fmt.Errorf("server command required")
	}

	return &result, nil
}

func readConfigFile(wd string) (*tomlConfig, error) {
	for _, filename := range configFileNames() {
		joined := filepath.Join(wd, filename)
		var d tomlConfig
		if _, err := toml.DecodeFile(joined, &d); err != nil {
			if errors.Is(err, os.ErrNotExist) {
				continue
			}
			return nil, err
		} else {
			switch d.QuitSignal {
			case "":
				d.quitSignalInt = syscall.SIGINT
			case "SIGINT":
				d.quitSignalInt = syscall.SIGINT
			case "SIGTERM":
				d.quitSignalInt = syscall.SIGTERM
			default:
				return nil, fmt.Errorf("quit_signal value not supported: %s", d.QuitSignal)
			}
			return &d, nil
		}
	}
	return nil, errConfigFileNotFound
}
