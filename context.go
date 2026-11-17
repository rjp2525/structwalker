package structwalker

import (
	"fmt"
	"reflect"
	"strconv"
	"time"
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
}

// SetValue sets the field value with automatic type coercion.
// Supports string-to-numeric, string-to-bool, and string-to-time.Time (RFC 3339).
func (fc FieldContext) SetValue(v any) error {
	if !fc.Value.CanSet() {
		return fmt.Errorf("field %s is not settable", fc.Path)
	}

	if v == nil {
		fc.Value.Set(reflect.Zero(fc.Value.Type()))
		return nil
	}

	rv := reflect.ValueOf(v)

	if rv.Type().AssignableTo(fc.Value.Type()) {
		fc.Value.Set(rv)
		return nil
	}

	if rv.Type().ConvertibleTo(fc.Value.Type()) {
		fc.Value.Set(rv.Convert(fc.Value.Type()))
		return nil
	}

	if rv.Kind() == reflect.String {
		return fc.setFromString(rv.String())
	}

	return fmt.Errorf("cannot assign %T to field %s of type %s", v, fc.Path, fc.Value.Type())
}

func (fc FieldContext) setFromString(s string) error {
	t := fc.Value.Type()

	if t == reflect.TypeFor[time.Time]() {
		parsed, err := time.Parse(time.RFC3339, s)
		if err != nil {
			return fmt.Errorf("cannot parse %q as time.Time for field %s: %w", s, fc.Path, err)
		}
		fc.Value.Set(reflect.ValueOf(parsed))
		return nil
	}

	switch fc.Value.Kind() {
	case reflect.String:
		fc.Value.SetString(s)
	case reflect.Int, reflect.Int8, reflect.Int16, reflect.Int32, reflect.Int64:
		n, err := strconv.ParseInt(s, 10, 64)
		if err != nil {
			return fmt.Errorf("cannot parse %q as int for field %s: %w", s, fc.Path, err)
		}
		fc.Value.SetInt(n)
	case reflect.Uint, reflect.Uint8, reflect.Uint16, reflect.Uint32, reflect.Uint64:
		n, err := strconv.ParseUint(s, 10, 64)
		if err != nil {
			return fmt.Errorf("cannot parse %q as uint for field %s: %w", s, fc.Path, err)
		}
		fc.Value.SetUint(n)
	case reflect.Float32, reflect.Float64:
		n, err := strconv.ParseFloat(s, 64)
		if err != nil {
			return fmt.Errorf("cannot parse %q as float for field %s: %w", s, fc.Path, err)
		}
		fc.Value.SetFloat(n)
	case reflect.Bool:
		b, err := strconv.ParseBool(s)
		if err != nil {
			return fmt.Errorf("cannot parse %q as bool for field %s: %w", s, fc.Path, err)
		}
		fc.Value.SetBool(b)
	default:
		return fmt.Errorf("cannot coerce string to %s for field %s", t, fc.Path)
	}
	return nil
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
