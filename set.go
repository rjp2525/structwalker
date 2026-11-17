package structwalker

import (
	"encoding"
	"errors"
	"fmt"
	"math"
	"reflect"
	"strconv"
	"time"
)

var (
	textUnmarshalerType = reflect.TypeFor[encoding.TextUnmarshaler]()
	durationType        = reflect.TypeFor[time.Duration]()
)

// SetValue sets the field to v, converting where that can be done without
// losing anything.
//
// A value of the field's own type is stored as is. Numbers convert between
// numeric types as long as they fit. A string is parsed into whatever the
// field is: a number, a bool, a time.Duration, a []byte or any type that
// implements encoding.TextUnmarshaler, which covers time.Time, net/netip.Addr
// and most ID types. A pointer field is allocated and pointed at the result.
// nil resets the field to its zero value.
//
// SetValue returns ErrNotSettable if the field cannot be changed.
func (fc FieldContext) SetValue(v any) error {
	if !fc.Value.CanSet() {
		return fmt.Errorf("field %s: %w", fc.Path, ErrNotSettable)
	}
	if err := assign(fc.Value, v); err != nil {
		return fmt.Errorf("field %s: %w", fc.Path, err)
	}
	return nil
}

func assign(dst reflect.Value, v any) error {
	if v == nil {
		dst.SetZero()
		return nil
	}

	src := reflect.ValueOf(v)
	switch {
	case src.Type().AssignableTo(dst.Type()):
		dst.Set(src)
		return nil

	case dst.Kind() == reflect.Pointer:
		target := reflect.New(dst.Type().Elem())
		if err := assign(target.Elem(), v); err != nil {
			return err
		}
		dst.Set(target)
		return nil

	case src.Kind() == reflect.String:
		return assignString(dst, src.String())

	case isNumeric(src.Kind()) && isNumeric(dst.Kind()):
		return assignNumber(dst, src)

	case !isNumeric(src.Kind()) && !isNumeric(dst.Kind()) && src.Type().ConvertibleTo(dst.Type()):
		dst.Set(src.Convert(dst.Type()))
		return nil
	}

	return fmt.Errorf("cannot assign %s to %s", src.Type(), dst.Type())
}

func assignString(dst reflect.Value, s string) error {
	if dst.CanAddr() && dst.Addr().Type().Implements(textUnmarshalerType) {
		if err := dst.Addr().Interface().(encoding.TextUnmarshaler).UnmarshalText([]byte(s)); err != nil {
			return parseError(s, dst, err)
		}
		return nil
	}

	if dst.Type() == durationType {
		d, err := time.ParseDuration(s)
		if err != nil {
			return parseError(s, dst, err)
		}
		dst.SetInt(int64(d))
		return nil
	}

	switch {
	case dst.Kind() == reflect.String:
		dst.SetString(s)
	case dst.CanInt():
		n, err := strconv.ParseInt(s, 10, dst.Type().Bits())
		if err != nil {
			return parseError(s, dst, err)
		}
		dst.SetInt(n)
	case dst.CanUint():
		n, err := strconv.ParseUint(s, 10, dst.Type().Bits())
		if err != nil {
			return parseError(s, dst, err)
		}
		dst.SetUint(n)
	case dst.CanFloat():
		n, err := strconv.ParseFloat(s, dst.Type().Bits())
		if err != nil {
			return parseError(s, dst, err)
		}
		dst.SetFloat(n)
	case dst.Kind() == reflect.Bool:
		b, err := strconv.ParseBool(s)
		if err != nil {
			return parseError(s, dst, err)
		}
		dst.SetBool(b)
	case dst.Kind() == reflect.Slice && dst.Type().Elem().Kind() == reflect.Uint8:
		dst.SetBytes([]byte(s))
	default:
		return fmt.Errorf("cannot parse a string into %s", dst.Type())
	}
	return nil
}

func parseError(s string, dst reflect.Value, err error) error {
	if numErr, ok := errors.AsType[*strconv.NumError](err); ok {
		err = numErr.Err
	}
	return fmt.Errorf("cannot parse %q as %s: %w", s, dst.Type(), err)
}

// assignNumber converts between numeric kinds and refuses anything that would
// change the value: overflow, a negative into an unsigned or a dropped fraction.
func assignNumber(dst, src reflect.Value) error {
	fits := false
	switch {
	case dst.CanInt():
		var n int64
		switch {
		case src.CanInt():
			n, fits = src.Int(), true
		case src.CanUint():
			n, fits = int64(src.Uint()), src.Uint() <= math.MaxInt64
		default:
			f := src.Float()
			n, fits = int64(f), f == math.Trunc(f) && f >= math.MinInt64 && f < math.MaxInt64
		}
		if fits = fits && !dst.OverflowInt(n); fits {
			dst.SetInt(n)
		}

	case dst.CanUint():
		var n uint64
		switch {
		case src.CanInt():
			n, fits = uint64(src.Int()), src.Int() >= 0
		case src.CanUint():
			n, fits = src.Uint(), true
		default:
			f := src.Float()
			n, fits = uint64(f), f == math.Trunc(f) && f >= 0 && f < math.MaxUint64
		}
		if fits = fits && !dst.OverflowUint(n); fits {
			dst.SetUint(n)
		}

	default:
		var f float64
		switch {
		case src.CanInt():
			f = float64(src.Int())
		case src.CanUint():
			f = float64(src.Uint())
		default:
			f = src.Float()
		}
		if fits = !dst.OverflowFloat(f); fits {
			dst.SetFloat(f)
		}
	}

	if !fits {
		return fmt.Errorf("%v does not fit in %s", src, dst.Type())
	}
	return nil
}

func isNumeric(k reflect.Kind) bool {
	switch k {
	case reflect.Int, reflect.Int8, reflect.Int16, reflect.Int32, reflect.Int64,
		reflect.Uint, reflect.Uint8, reflect.Uint16, reflect.Uint32, reflect.Uint64, reflect.Uintptr,
		reflect.Float32, reflect.Float64:
		return true
	}
	return false
}
