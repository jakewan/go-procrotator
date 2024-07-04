package watchdirs

import (
	"regexp"
	"slices"

	"github.com/jakewan/go-procrotator/logger"
)

type (
	Dependencies interface {
		Logger() logger.Logger
	}
	FileChangedEvent struct {
		Path string
	}
)

func StartEventProcessing(
	deps Dependencies,
	includeFileRegexes []regexp.Regexp,
	excludeFileRegexes []regexp.Regexp,
	fileChangedChan chan<- FileChangedEvent,
	changes <-chan WatcherEvent,
	errors <-chan error,
	done chan<- bool,
) {
	defer func() {
		done <- true
	}()
	l := deps.Logger()
	includeOps := []WatcherEventOp{
		CREATE,
		REMOVE,
		RENAME,
		WRITE,
	}
	slices.Sort(includeOps)
	for {
		select {
		case ev, ok := <-changes:
			if ok {
				l.Errorf(logger.DEBUG, "Event: %v", ev)
				shouldReport := false
				for _, op := range ev.Ops {
					if slices.Index(includeOps, op) > -1 {
						shouldReport = true
						break
					}
				}
				if shouldReport {
					// Check the filename against the list of include regexes.
					if slices.IndexFunc(includeFileRegexes, func(r regexp.Regexp) bool {
						return r.MatchString(ev.Path)
					}) > -1 {
						l.Errorf(logger.DEBUG, "File is included: %s", ev.Path)
						if slices.IndexFunc(excludeFileRegexes, func(r regexp.Regexp) bool {
							return r.MatchString(ev.Path)
						}) > -1 {
							l.Errorf(logger.DEBUG, "File is excluded: %s", ev.Path)
						} else {
							fileChangedChan <- FileChangedEvent{Path: ev.Path}
						}
					}
				} else {
					l.Errorf(logger.DEBUG, "Skipping %s events for %s", ev.Ops, ev.Path)
				}
			} else {
				changes = nil
			}
		case err, ok := <-errors:
			if ok {
				l.Errorf(logger.ERROR, "Error watching directories: %s", err)
			} else {
				errors = nil
			}
		}
		if changes == nil && errors == nil {
			break
		}
	}
}
