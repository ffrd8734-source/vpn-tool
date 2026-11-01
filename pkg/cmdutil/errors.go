package cmdutil

import (
	"fmt"
)

// FlagError wraps flag parsing errors with friendly usage hints
type FlagError struct {
	err error
}

func (e *FlagError) Error() string {
	return e.err.Error()
}

func (e *FlagError) Unwrap() error {
	return e.err
}

// FlagErrorWrap wraps an error encountered during flag parsing
func FlagErrorWrap(err error) error {
	return &FlagError{err: fmt.Errorf("%w. See '--help' for usage.", err)}
}
