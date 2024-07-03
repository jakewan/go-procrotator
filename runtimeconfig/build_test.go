package runtimeconfig_test

import (
	"os"
	"path/filepath"
	"regexp"
	"syscall"
	"testing"

	"github.com/jakewan/go-procrotator/logger"
	"github.com/jakewan/go-procrotator/runtimeconfig"
	"github.com/stretchr/testify/assert"
)

func TestBuild(t *testing.T) {
	type testConfig struct {
		desc            string
		args            []string
		argsFunc        func(d string) []string
		validateConfig  func(t *testing.T, c runtimeconfig.Config)
		validateError   func(t *testing.T, err error)
		changeToTempDir bool
		tempDirSetup    func(d string)
	}
	var defaultConfigFileContent = []byte(
		`include_file_regexes = ["\\.foo$", "\\.bar$"]
  exclude_file_regexes = ["ignore\\.foo$"]
  preamble_commands = ["some command"]
  server_command = "./some-app"
  quit_signal = "SIGTERM"
  log_level = "DEBUG"`)
	testConfigs := []testConfig{
		{
			desc:            "all settings from config file in working directory",
			changeToTempDir: true,
			tempDirSetup: func(d string) {
				if err := os.WriteFile(
					filepath.Join(d, ".procrotator.toml"),
					defaultConfigFileContent,
					0666,
				); err != nil {
					panic(err)
				}
			},
			validateConfig: func(t *testing.T, c runtimeconfig.Config) {
				assert.Equal(t, "./some-app", c.ServerCommand())
				assert.Equal(t, []string{"some command"}, c.PreambleCommands())
				assert.Equal(t, syscall.SIGTERM, c.QuitSignal())
				assert.Equal(
					t,
					[]regexp.Regexp{
						*regexp.MustCompile(`\.foo$`),
						*regexp.MustCompile(`\.bar$`),
					},
					c.IncludeFileRegexes(),
				)
				assert.Equal(
					t,
					[]regexp.Regexp{*regexp.MustCompile(`ignore\.foo$`)},
					c.ExcludeFileRegexes(),
				)
				assert.Equal(t, logger.DEBUG, c.LogLevel())
			},
		},
		{
			desc: "command line overrides",
			args: []string{
				"-s", "./some-other-app",
				"-p", "preamble foo",
				"-p", "preamble bar",
				"-i", "\\.baz$",
				"-i", "\\.quux$",
				"-e", "ignore\\.baz$",
				"-e", "ignore\\.quux$",
				"-l", "ERROR",
			},
			changeToTempDir: true,
			tempDirSetup: func(d string) {
				if err := os.WriteFile(
					filepath.Join(d, ".procrotator.toml"),
					defaultConfigFileContent,
					0666,
				); err != nil {
					panic(err)
				}
			},
			validateConfig: func(t *testing.T, c runtimeconfig.Config) {
				assert.Equal(t, "./some-other-app", c.ServerCommand())
				assert.Equal(
					t,
					[]string{
						"preamble foo",
						"preamble bar",
					},
					c.PreambleCommands(),
				)
				assert.Equal(
					t,
					[]regexp.Regexp{
						*regexp.MustCompile(`\.baz$`),
						*regexp.MustCompile(`\.quux$`),
					},
					c.IncludeFileRegexes(),
				)
				assert.Equal(
					t,
					[]regexp.Regexp{
						*regexp.MustCompile(`ignore\.baz$`),
						*regexp.MustCompile(`ignore\.quux$`),
					},
					c.ExcludeFileRegexes(),
				)
				assert.Equal(t, logger.ERROR, c.LogLevel())
			},
		},
		{
			desc:            "defaults",
			changeToTempDir: true,
			tempDirSetup: func(d string) {
				if err := os.WriteFile(
					filepath.Join(d, ".procrotator.toml"),
					[]byte(`
include_file_regexes = ["\\.foo$"]
  server_command = "./some-app"
`),
					0666,
				); err != nil {
					panic(err)
				}
			},
			validateConfig: func(t *testing.T, c runtimeconfig.Config) {
				assert.Equal(t, logger.INFO, c.LogLevel())
				assert.Equal(t, syscall.SIGINT, c.QuitSignal())
			},
		},
		{
			desc:            "specify directory",
			changeToTempDir: false,
			argsFunc: func(d string) []string {
				return []string{"-d", d}
			},
			tempDirSetup: func(d string) {
				if err := os.WriteFile(
					filepath.Join(d, ".procrotator.toml"),
					[]byte(`
include_file_regexes = ["\\.foo$"]
  server_command = "./some-app"
`),
					0666,
				); err != nil {
					panic(err)
				}
			},
			validateConfig: func(t *testing.T, c runtimeconfig.Config) {
				assert.Equal(t, "./some-app", c.ServerCommand())
			},
		},
	}
	for _, cfg := range testConfigs {
		t.Run(
			cfg.desc,
			func(t *testing.T) {
				// Setup
				tempDir := t.TempDir()
				if cfg.tempDirSetup != nil {
					cfg.tempDirSetup(tempDir)
				}
				if cfg.changeToTempDir {
					if err := os.Chdir(tempDir); err != nil {
						assert.FailNow(t, "Error setting working directory: %w", err)
					}
				}

				// Determine command line arguments.
				var args []string
				if cfg.args != nil {
					args = cfg.args
				} else if cfg.argsFunc != nil {
					args = cfg.argsFunc(tempDir)
				}

				// Code under test
				if c, err := runtimeconfig.Build(args); err != nil {
					if cfg.validateError != nil {
						cfg.validateError(t, err)
					} else {
						assert.FailNow(t, "Unexpected error", err)
					}
				} else if cfg.validateError != nil {
					assert.FailNow(t, "Expected error not returned")
				} else if cfg.validateConfig != nil {
					cfg.validateConfig(t, c)
				}
			},
		)
	}
}
