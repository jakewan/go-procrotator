package watchdirs

import (
	"regexp"
	"slices"
	"strings"

	"github.com/fsnotify/fsnotify"
	"github.com/jakewan/go-procrotator/logger"
)

type (
	Dependencies interface {
		Logger() logger.Logger
		NewFilesystemWatcher() (FilesystemWatcher, error)
	}
	FilesystemWatcher interface {
		Add(string) error
		Events() chan fsnotify.Event
		Errors() chan error
		Close() error
	}
	FileChangedEvent struct {
		Path string
	}
)

func StartWatchDirs(
	deps Dependencies,
	dirs []string,
	includeFileRegexes []regexp.Regexp,
	excludeFileRegexes []regexp.Regexp,
	fileChangedChan chan<- FileChangedEvent,
	errors chan<- error,
	quit <-chan bool,
	done chan<- bool,
) {
	defer func() {
		done <- true
	}()
	l := deps.Logger()
	l.Errorf(
		logger.INFO,
		"Watching directories:\n%s",
		strings.Join(dirs, "\n"),
	)
	if watcher, err := deps.NewFilesystemWatcher(); err != nil {
		l.Errorf(logger.DEBUG, "Sending child error: %s", err)
		errors <- err
	} else {
		defer watcher.Close()
		for _, d := range dirs {
			if err := watcher.Add(d); err != nil {
				errors <- err
				return
			}
		}
		for {
			select {
			case <-quit:
				return
			case ev, ok := <-watcher.Events():
				if !ok {
					return
				}
				// Check the filename against the list of include regexes.
				if slices.IndexFunc(includeFileRegexes, func(r regexp.Regexp) bool {
					return r.MatchString(ev.Name)
				}) > -1 {
					l.Errorf(logger.DEBUG, "File is included: %s", ev.Name)
					if slices.IndexFunc(excludeFileRegexes, func(r regexp.Regexp) bool {
						return r.MatchString(ev.Name)
					}) > -1 {
						l.Errorf(logger.DEBUG, "File is excluded: %s", ev.Name)
					} else {
						fileChangedChan <- FileChangedEvent{Path: ev.Name}
					}
				}
			case err, ok := <-watcher.Errors():
				if !ok {
					return
				}
				errors <- err
			}
		}
	}
}
