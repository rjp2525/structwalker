package structwalker

import (
	"context"
	"reflect"
)

// FieldContext describes the field a handler is being called for.
type FieldContext struct {
	// Tag is the tag the handler was registered for. It is empty for HandleAll
	// handlers and in Fields loops, where Lookup reads tags by key.
	Tag Tag
	// AllTags is every tag on the field, in the order they are written.
	AllTags []Tag
	// Field is the field's reflection metadata.
	Field reflect.StructField
	// Value is the field's value. It is settable when Walk was given a pointer.
	Value reflect.Value
	// Path locates the field from the root, such as "Address.Street",
	// "Items[2].Name" or "Limits[eu].Max".
	Path string
	// Depth is how many structs deep the field is. Fields of the struct passed
	// to Walk are at depth 0.
	Depth int
	// Parent is the struct that holds the field.
	Parent reflect.Value
	// Index is the field's position in Parent.
	Index int
	// Walker is the walker running this walk.
	Walker *Walker

	ctx context.Context
}

// Context returns the context the walk was started with. It is never nil.
func (fc FieldContext) Context() context.Context {
	if fc.ctx == nil {
		return context.Background()
	}
	return fc.ctx
}

// Lookup returns the field's tag for the given key, whichever handler is
// running. A validate handler can use it to read the field's json name.
func (fc FieldContext) Lookup(key string) (Tag, bool) {
	for _, tag := range fc.AllTags {
		if tag.Key == key {
			return tag, true
		}
	}
	return Tag{}, false
}

// IsZero reports whether the field holds the zero value for its type.
func (fc FieldContext) IsZero() bool {
	return fc.Value.IsZero()
}

// IsNil reports whether the field is a nil pointer, interface, slice, map,
// channel or func. It is false for every other kind.
func (fc FieldContext) IsNil() bool {
	switch fc.Value.Kind() {
	case reflect.Pointer, reflect.Slice, reflect.Map, reflect.Interface, reflect.Chan, reflect.Func:
		return fc.Value.IsNil()
	default:
		return false
	}
}

// IsExported reports whether the field is exported.
func (fc FieldContext) IsExported() bool {
	return fc.Field.IsExported()
}
