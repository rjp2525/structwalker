package structwalker

import (
	"errors"
	"fmt"
	"strings"
)

var (
	// ErrSkipField tells the walker to leave the current field: no more
	// handlers run on it and its children are not walked.
	ErrSkipField = errors.New("skip field")

	// ErrSkipChildren tells the walker not to descend into the current field.
	// The field's other handlers still run.
	ErrSkipChildren = errors.New("skip children")

	// ErrMaxDepth is reported when a struct is nested deeper than WithMaxDepth
	// allows. Check for it with errors.Is.
	ErrMaxDepth = errors.New("max depth exceeded")
)

func isFlowControl(err error) bool {
	return errors.Is(err, ErrSkipField) || errors.Is(err, ErrSkipChildren)
}

// WalkError is an error with the field it happened on.
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

// MultiError is every error from a walk in CollectErrors mode, in the order
// the walker met them.
type MultiError struct {
	Errors []*WalkError
}

func (e *MultiError) Error() string {
	if len(e.Errors) == 1 {
		return e.Errors[0].Error()
	}
	var b strings.Builder
	fmt.Fprintf(&b, "structwalker: %d errors:", len(e.Errors))
	for _, err := range e.Errors {
		b.WriteString("\n  - ")
		b.WriteString(strings.TrimPrefix(err.Error(), "structwalker: "))
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

// PanicError is what WithRecovery turns a handler panic into.
type PanicError struct {
	Value any
	Stack []byte
}

func (e *PanicError) Error() string {
	return fmt.Sprintf("panic: %v", e.Value)
}

// Unwrap returns the panic value when the handler panicked with an error.
func (e *PanicError) Unwrap() error {
	err, _ := e.Value.(error)
	return err
}
