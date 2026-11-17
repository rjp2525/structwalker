package structwalker

import "context"

// ErrorMode controls how the walker handles errors from handlers.
type ErrorMode int

const (
	// StopOnError ends the walk at the first handler error.
	StopOnError ErrorMode = iota
	// CollectErrors keeps walking and returns every error as a *MultiError.
	CollectErrors
	// SkipOnError drops handler errors and keeps walking.
	SkipOnError
)

type walkerOptions struct {
	maxDepth       int
	unexported     bool
	followPointers bool
	sliceElements  bool
	mapEntries     bool
	errorMode      ErrorMode
	ctx            context.Context
	tagParser      func(key, raw string) Tag
}

func defaultOptions() walkerOptions {
	return walkerOptions{
		maxDepth:       32,
		followPointers: true,
		ctx:            context.Background(),
	}
}

// WalkerOption configures walker behavior via functional options.
type WalkerOption func(*walkerOptions)

// WithMaxDepth limits recursion depth (default: 32).
func WithMaxDepth(n int) WalkerOption {
	return func(o *walkerOptions) { o.maxDepth = n }
}

// WithUnexported includes unexported fields in the walk (read-only, not settable).
func WithUnexported(b bool) WalkerOption {
	return func(o *walkerOptions) { o.unexported = b }
}

// WithFollowPointers controls automatic pointer dereferencing (default: true).
func WithFollowPointers(b bool) WalkerOption {
	return func(o *walkerOptions) { o.followPointers = b }
}

// WithSliceElements enables walking into individual slice/array elements.
func WithSliceElements(b bool) WalkerOption {
	return func(o *walkerOptions) { o.sliceElements = b }
}

// WithMapEntries enables walking into map values.
func WithMapEntries(b bool) WalkerOption {
	return func(o *walkerOptions) { o.mapEntries = b }
}

// WithErrorMode sets the error handling strategy for the walk.
func WithErrorMode(mode ErrorMode) WalkerOption {
	return func(o *walkerOptions) { o.errorMode = mode }
}

// WithContext attaches a context for cancellation support during walks.
func WithContext(ctx context.Context) WalkerOption {
	return func(o *walkerOptions) { o.ctx = ctx }
}

// WithTagParser overrides the default CSV-based tag parsing logic.
func WithTagParser(fn func(key, raw string) Tag) WalkerOption {
	return func(o *walkerOptions) { o.tagParser = fn }
}
