package structwalker

import (
	"errors"
	"fmt"
	"strings"
)

var (
	// ErrSkipField causes the walker to skip remaining handlers on the current field.
	ErrSkipField = errors.New("skip field")

	// ErrSkipChildren prevents recursion into a nested struct field.
	ErrSkipChildren = errors.New("skip children")
)

// WalkError wraps an error with field path context.
type WalkError struct {
	Path   string
	TagKey string
	Field  string
	Err    error
}

func (e *WalkError) Error() string {
	if e.TagKey != "" {
		return fmt.Sprintf("structwalker: %s (tag %q): %v", e.Path, e.TagKey, e.Err)
	}
	return fmt.Sprintf("structwalker: %s: %v", e.Path, e.Err)
}

func (e *WalkError) Unwrap() error {
	return e.Err
}

// MultiError collects multiple errors from a walk (used with CollectErrors mode).
type MultiError struct {
	Errors []*WalkError
}

func (e *MultiError) Error() string {
	if len(e.Errors) == 1 {
		return e.Errors[0].Error()
	}
	var b strings.Builder
	fmt.Fprintf(&b, "structwalker: %d errors:\n", len(e.Errors))
	for _, err := range e.Errors {
		fmt.Fprintf(&b, "  - %s\n", err.Error())
	}
	return b.String()
}

func (e *MultiError) Unwrap() []error {
	errs := make([]error, len(e.Errors))
	for i, err := range e.Errors {
		errs[i] = err
	}
	return errs
}

func (e *MultiError) add(err *WalkError) {
	e.Errors = append(e.Errors, err)
}
