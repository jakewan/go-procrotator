package main

import (
	"fmt"
	"io/fs"
	"os"
	"os/signal"
	"path/filepath"
	"syscall"

	"github.com/fsnotify/fsnotify"
	"github.com/jakewan/go-procrotator/childproc"
	"github.com/jakewan/go-procrotator/logger"
	"github.com/jakewan/go-procrotator/runtimeconfig"
	"github.com/jakewan/go-procrotator/watchdirs"
)

func main() {
	l := logger.NewLogger("go-procrotator", os.Stderr)
	if cfg, err := runtimeconfig.Build(os.Args[1:]); err != nil {
		l.Errorf(logger.ERROR, err.Error())
		os.Exit(1)
	} else {
		l.SetErrorLevel(cfg.LogLevel())
		if cfg.WorkingDirectory() != "" {
			if err := os.Chdir(cfg.WorkingDirectory()); err != nil {
				l.Errorf(logger.ERROR, err.Error())
				os.Exit(1)
			}
		}
		if wd, err := os.Getwd(); err != nil {
			l.Errorf(logger.ERROR, err.Error())
			os.Exit(1)
		} else if len(cfg.IncludeFileRegexes()) < 1 {
			l.Errorf(logger.WARNING, "Warning, no include file globs detected.")
			os.Exit(1)
		} else {
			startProcessing(wd, l, cfg)
		}
	}
}

func startProcessing(wd string, l logger.Logger, cfg runtimeconfig.Config) {
	if watchDirs, err := getDirectoriesToWatch(wd); err != nil {
		l.Errorf(logger.ERROR, err.Error())
		os.Exit(1)
	} else {
		startBackgroundProcesses(l, cfg, watchDirs)
	}
}

func startBackgroundProcesses(l logger.Logger, cfg runtimeconfig.Config, watchDirs []string) {
	watchDirsQuitChan := make(chan bool)
	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, syscall.SIGINT, syscall.SIGTERM)
	trapSignalsDone := make(chan bool, 1)
	childErrors := make(chan error)
	fileChangedChan := make(chan watchdirs.FileChangedEvent)

	childProcManagerDone := make(chan bool)
	go childproc.StartChildProcess(
		newChildProcDeps(l),
		cfg,
		fileChangedChan,
		childProcManagerDone,
	)

	l.Errorf(logger.DEBUG, "Starting to watch directories")
	watchDirsDone := make(chan bool)
	go watchdirs.StartWatchDirs(
		newWatchDirsDeps(l),
		watchDirs,
		cfg.IncludeFileRegexes(),
		cfg.ExcludeFileRegexes(),
		fileChangedChan,
		childErrors,
		watchDirsQuitChan,
		watchDirsDone,
	)
	l.Errorf(logger.DEBUG, "Directory watch has begun")

	go startTrapSignals(sigChan, trapSignalsDone)

	l.Errorf(logger.DEBUG, "Waiting for quit signal")
	<-trapSignalsDone
	l.Errorf(logger.DEBUG, "Quit signal received")

	// Signal the director watching process to quit and wait for it to
	// signal completion.
	watchDirsQuitChan <- true
	<-watchDirsDone
	l.Errorf(logger.DEBUG, "Directory watching processes completed")

	// Signal the child process manager to quit by closing the file change
	// channel and wait for it to signal completion.
	close(fileChangedChan)
	<-childProcManagerDone
	l.Errorf(logger.DEBUG, "Child process manager done")
}

func startTrapSignals(sigChan <-chan os.Signal, done chan<- bool) {
	defer func() {
		done <- true
	}()
	<-sigChan
}

type fswatcherWrapper struct {
	watcher *fsnotify.Watcher
}

// Events implements watchdirs.FilesystemWatcher.
func (f *fswatcherWrapper) Events() chan fsnotify.Event {
	return f.watcher.Events
}

// Errors implements watchdirs.FilesystemWatcher.
func (f *fswatcherWrapper) Errors() chan error {
	return f.watcher.Errors
}

// Add implements watchdirs.FilesystemWatcher.
func (f *fswatcherWrapper) Add(name string) error {
	return f.watcher.Add(name)
}

// Close implements watchdirs.FilesystemWatcher.
func (f *fswatcherWrapper) Close() error {
	return f.watcher.Close()
}

type watchdirsDeps struct {
	logger logger.Logger
}

// Logger implements watchdirs.Dependencies.
func (w *watchdirsDeps) Logger() logger.Logger {
	return w.logger
}

// NewFilesystemWatcher implements watchdirs.Dependencies.
func (w *watchdirsDeps) NewFilesystemWatcher() (watchdirs.FilesystemWatcher, error) {
	if w, err := fsnotify.NewWatcher(); err != nil {
		return nil, fmt.Errorf("creating fsnotify.Watcher: %w", err)
	} else {
		return &fswatcherWrapper{
			watcher: w,
		}, nil
	}
}

func newWatchDirsDeps(l logger.Logger) watchdirs.Dependencies {
	return &watchdirsDeps{logger: l}
}

func getDirectoriesToWatch(root string) ([]string, error) {
	result := []string{}
	if err := filepath.WalkDir(
		root,
		func(path string, d fs.DirEntry, err error) error {
			if err != nil {
				return err
			}
			if d.IsDir() {
				result = append(result, path)
			}
			return nil
		},
	); err != nil {
		return nil, fmt.Errorf("walking root directory: %w", err)
	}
	return result, nil
}

type childprocmanagerDeps struct {
	logger logger.Logger
}

// Logger implements childprocmanager.Dependencies.
func (c *childprocmanagerDeps) Logger() logger.Logger {
	return c.logger
}

func newChildProcDeps(l logger.Logger) childproc.Dependencies {
	return &childprocmanagerDeps{logger: l}
}
