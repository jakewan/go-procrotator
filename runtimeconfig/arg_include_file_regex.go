package runtimeconfig

import "regexp"

type argMultiRegex struct {
	argname  string
	argusage string
	regexes  *[]regexp.Regexp
}

// stringFunc implements argDefWithStringFunc.
func (a argMultiRegex) stringFunc() func(s string) error {
	return func(s string) error {
		if r, err := regexp.Compile(s); err != nil {
			return err
		} else {
			*a.regexes = append(*a.regexes, *r)
			return nil
		}
	}
}

// name implements argDef.
func (a argMultiRegex) name() string {
	return a.argname
}

// usage implements argDef.
func (a argMultiRegex) usage() string {
	return a.argusage
}
