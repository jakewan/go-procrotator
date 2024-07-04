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
			l.Errorf(logger.WARNING, "Warning, no include file regexes detected.")
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
		startFilesystemWatcher(l, cfg, watchDirs)
	}
}

func startFilesystemWatcher(l logger.Logger, cfg runtimeconfig.Config, watchDirs []string) {
	if w, err := fsnotify.NewWatcher(); err != nil {
		l.Errorf(logger.ERROR, err.Error())
		os.Exit(1)
	} else {
		for _, d := range watchDirs {
			if err := w.Add(d); err != nil {
				l.Errorf(logger.ERROR, err.Error())
				os.Exit(1)
			}
		}
		startBackgroundProcesses(l, cfg, w)
	}
}

func startBackgroundProcesses(
	l logger.Logger,
	cfg runtimeconfig.Config,
	watcher *fsnotify.Watcher,
) {
	defer watcher.Close()
	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, syscall.SIGINT, syscall.SIGTERM)
	trapSignalsDone := make(chan bool, 1)
	watchDirEvents := make(chan watchdirs.WatcherEvent)
	watchDirErrors := make(chan error)
	fileChangedChan := make(chan watchdirs.FileChangedEvent)
	quitWatchDirs := make(chan bool)
	watchDirsDone := make(chan bool)

	childProcManagerDone := make(chan bool)
	go childproc.StartChildProcess(
		newChildProcDeps(l),
		cfg,
		fileChangedChan,
		childProcManagerDone,
	)

	eventProcessingDone := make(chan bool)
	go watchdirs.StartEventProcessing(
		newWatchDirsDeps(l),
		cfg.IncludeFileRegexes(),
		cfg.ExcludeFileRegexes(),
		fileChangedChan,
		watchDirEvents,
		watchDirErrors,
		eventProcessingDone,
	)
	l.Errorf(logger.DEBUG, "Waiting for file change events")

	go startWatchDirs(
		l,
		watcher,
		watchDirEvents,
		quitWatchDirs,
		watchDirsDone,
	)
	l.Errorf(logger.DEBUG, "Watching directories")

	// Start trapping signals and wait for the user to terminate the program.
	go startTrapSignals(sigChan, trapSignalsDone)
	<-trapSignalsDone
	l.Errorf(logger.DEBUG, "Quit signal received")

	// Signal the director watching process to quit and wait for completion.
	quitWatchDirs <- true
	<-watchDirsDone
	l.Errorf(logger.DEBUG, "Directory watching processes completed")

	// Signal the event processor to quit by closing its receiving channels
	// and wait for it to signal completion.
	close(watchDirEvents)
	close(watchDirErrors)
	<-eventProcessingDone
	l.Errorf(logger.DEBUG, "Event processor completed")

	// Signal the child process manager to quit by closing the file change
	// channel and wait for it to signal completion.
	close(fileChangedChan)
	<-childProcManagerDone
	l.Errorf(logger.DEBUG, "Child process manager completed")
}

func startWatchDirs(l logger.Logger, w *fsnotify.Watcher, changes chan<- watchdirs.WatcherEvent, quit <-chan bool, done chan<- bool) {
	defer func() {
		done <- true
	}()
	for {
		select {
		case <-quit:
			return
		case ev, ok := <-w.Events:
			if ok {
				var ops []watchdirs.WatcherEventOp
				if ev.Op.Has(fsnotify.Chmod) {
					ops = append(ops, watchdirs.CHMOD)
				}
				if ev.Op.Has(fsnotify.Create) {
					ops = append(ops, watchdirs.CREATE)
				}
				if ev.Op.Has(fsnotify.Remove) {
					ops = append(ops, watchdirs.REMOVE)
				}
				if ev.Op.Has(fsnotify.Rename) {
					ops = append(ops, watchdirs.RENAME)
				}
				if ev.Op.Has(fsnotify.Write) {
					ops = append(ops, watchdirs.WRITE)
				}
				changes <- watchdirs.WatcherEvent{
					Path: ev.Name,
					Ops:  ops,
				}
			}
		case err, ok := <-w.Errors:
			if ok {
				l.Errorf(logger.ERROR, "Error from fsnotify: %s", err)
			}
		}
	}
}

func startTrapSignals(sigChan <-chan os.Signal, done chan<- bool) {
	defer func() {
		done <- true
	}()
	<-sigChan
}

type watchdirsDeps struct {
	logger logger.Logger
}

// Logger implements watchdirs.Dependencies.
func (w *watchdirsDeps) Logger() logger.Logger {
	return w.logger
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
