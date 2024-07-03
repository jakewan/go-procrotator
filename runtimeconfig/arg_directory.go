package runtimeconfig

import (
	"fmt"
	"os"
	"path/filepath"
)

type argDirectory struct {
	value *string
}

// name implements argDef.
func (a argDirectory) name() string {
	return "directory"
}

// stringFunc implements argDef.
func (a argDirectory) stringFunc() func(string) error {
	return func(s string) error {
		if abs, err := filepath.Abs(s); err != nil {
			return fmt.Errorf("obtaining absolute path from %s: %s", s, err)
		} else if fi, err := os.Stat(abs); err != nil {
			return fmt.Errorf("obtaining file information for %s: %w", abs, err)
		} else if fi.IsDir() {
			*a.value = abs
			return nil
		} else {
			return fmt.Errorf("%s is not a directory", abs)
		}
	}
}

// usage implements argDef.
func (a argDirectory) usage() string {
	return "The working directory where files will be watched and commands executed"
}
