package structwalker

import (
	"context"
	"reflect"
)

// FieldContext is passed to every handler invocation with full field metadata.
type FieldContext struct {
	Tag     Tag
	AllTags []Tag
	Field   reflect.StructField
	Value   reflect.Value
	Path    string
	Depth   int
	Parent  reflect.Value
	Index   int
	Walker  *Walker

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

// IsNil reports whether the field is a nil pointer, slice, map, or interface.
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
