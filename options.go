package structwalker

import (
	"context"
	"reflect"
	"time"
)

// ErrorMode decides what the walker does when a handler returns an error.
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
	interfaces     bool
	leafTypes      map[reflect.Type]struct{}
	errorMode      ErrorMode
	ctx            context.Context
	tagParser      func(key, raw string) Tag
}

func defaultOptions() walkerOptions {
	return walkerOptions{
		maxDepth:       32,
		followPointers: true,
		leafTypes:      map[reflect.Type]struct{}{reflect.TypeFor[time.Time](): {}},
		ctx:            context.Background(),
	}
}

// WalkerOption configures a Walker. Pass options to New.
type WalkerOption func(*walkerOptions)

// WithMaxDepth limits how deep structs may nest. Fields of the struct passed
// to Walk are at depth 0 and the default limit is 32.
func WithMaxDepth(n int) WalkerOption {
	return func(o *walkerOptions) { o.maxDepth = n }
}

// WithUnexported includes unexported fields in the walk. Handlers can read
// them through reflection but cannot set them or call Value.Interface on them.
func WithUnexported(b bool) WalkerOption {
	return func(o *walkerOptions) { o.unexported = b }
}

// WithFollowPointers controls whether the walker follows pointers to reach
// the structs behind them. It is on by default.
func WithFollowPointers(b bool) WalkerOption {
	return func(o *walkerOptions) { o.followPointers = b }
}

// WithSliceElements makes the walker visit the structs held in slices and
// arrays, however deeply the containers nest.
func WithSliceElements(b bool) WalkerOption {
	return func(o *walkerOptions) { o.sliceElements = b }
}

// WithMapEntries makes the walker visit the structs held in map values, in
// key order. Struct values stored directly in a map cannot be set. Store
// pointers if handlers need to change them.
func WithMapEntries(b bool) WalkerOption {
	return func(o *walkerOptions) { o.mapEntries = b }
}

// WithInterfaces makes the walker look inside interface fields and visit the
// struct or pointer an interface holds.
func WithInterfaces(b bool) WalkerOption {
	return func(o *walkerOptions) { o.interfaces = b }
}

// WithLeafTypes names struct types the walker should treat as plain values.
// Handlers still run on a field of that type but the walker does not descend
// into it. time.Time is always a leaf.
func WithLeafTypes(types ...reflect.Type) WalkerOption {
	return func(o *walkerOptions) {
		for _, t := range types {
			o.leafTypes[t] = struct{}{}
		}
	}
}

// WithErrorMode sets what happens when a handler returns an error. The
// default is StopOnError.
func WithErrorMode(mode ErrorMode) WalkerOption {
	return func(o *walkerOptions) { o.errorMode = mode }
}

// WithContext sets the context used by Walk and WalkValue.
//
// Deprecated: a context stored on the walker is shared by every walk, so one
// deadline ends them all. Pass a context per walk with WalkContext instead.
func WithContext(ctx context.Context) WalkerOption {
	return func(o *walkerOptions) { o.ctx = ctx }
}

// WithTagParser replaces ParseTag for tags that use a different syntax. fn
// receives a tag key and its unquoted value and is called once per field per
// struct type.
func WithTagParser(fn func(key, raw string) Tag) WalkerOption {
	return func(o *walkerOptions) { o.tagParser = fn }
}
