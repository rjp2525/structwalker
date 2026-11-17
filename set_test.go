package structwalker

import (
	"errors"
	"math"
	"net/netip"
	"reflect"
	"strings"
	"testing"
	"time"
)

type status string

type level int

type target struct {
	String   string
	Status   status
	Int      int
	Int8     int8
	Level    level
	Uint8    uint8
	Float32  float32
	Float64  float64
	Bool     bool
	Bytes    []byte
	Duration time.Duration
	Time     time.Time
	Addr     netip.Addr
	IntPtr   *int
	TimePtr  *time.Time
	Any      any
	Chan     chan int
}

func setField(t *testing.T, dst *target, field string, v any) error {
	t.Helper()
	found := false
	w := New()
	w.HandleAll(func(ctx FieldContext) error {
		if ctx.Path != field {
			return nil
		}
		found = true
		return ctx.SetValue(v)
	})
	err := w.Walk(dst)
	if !found {
		t.Fatalf("no field %s", field)
	}
	if walkErr, ok := errors.AsType[*WalkError](err); ok {
		return walkErr.Err
	}
	return err
}

func TestSetValue(t *testing.T) {
	seven := 7
	at := time.Date(2026, 4, 7, 12, 0, 0, 0, time.UTC)

	tests := []struct {
		field string
		in    any
		want  any
	}{
		{"String", "hello", "hello"},
		{"String", []byte("bytes"), "bytes"},
		{"Status", "active", status("active")},
		{"Status", status("typed"), status("typed")},
		{"Int", 42, 42},
		{"Int", "42", 42},
		{"Int", "-7", -7},
		{"Int", int64(9), 9},
		{"Int", uint8(9), 9},
		{"Int", 3.0, 3},
		{"Int8", "127", int8(127)},
		{"Int8", 100, int8(100)},
		{"Level", 3, level(3)},
		{"Level", "4", level(4)},
		{"Uint8", "255", uint8(255)},
		{"Uint8", 200, uint8(200)},
		{"Float32", "1.5", float32(1.5)},
		{"Float64", 2, 2.0},
		{"Float64", float32(0.5), 0.5},
		{"Bool", "true", true},
		{"Bool", "0", false},
		{"Bytes", "raw", []byte("raw")},
		{"Duration", "1m30s", 90 * time.Second},
		{"Duration", time.Second, time.Second},
		{"Duration", 5, time.Duration(5)},
		{"Time", "2026-04-07T12:00:00Z", at},
		{"Time", at, at},
		{"Addr", "10.0.0.1", netip.MustParseAddr("10.0.0.1")},
		{"IntPtr", "7", &seven},
		{"IntPtr", 7, &seven},
		{"IntPtr", &seven, &seven},
		{"TimePtr", "2026-04-07T12:00:00Z", &at},
		{"Any", "anything", "anything"},
		{"Any", 12, 12},
	}

	for _, tt := range tests {
		var dst target
		if err := setField(t, &dst, tt.field, tt.in); err != nil {
			t.Errorf("%s = %#v: %v", tt.field, tt.in, err)
			continue
		}
		got := reflect.ValueOf(dst).FieldByName(tt.field).Interface()
		if !reflect.DeepEqual(got, tt.want) {
			t.Errorf("%s = %#v: got %#v, want %#v", tt.field, tt.in, got, tt.want)
		}
	}
}

func TestSetValue_RefusesLossyConversions(t *testing.T) {
	tests := []struct {
		field   string
		in      any
		message string
	}{
		{"String", 65, "cannot assign int to string"},
		{"Status", 65, "cannot assign int to structwalker.status"},
		{"Int8", "300", `cannot parse "300" as int8: value out of range`},
		{"Int8", 300, "300 does not fit in int8"},
		{"Int", 1.5, "1.5 does not fit in int"},
		{"Int", math.NaN(), "NaN does not fit in int"},
		{"Int", uint64(math.MaxUint64), "18446744073709551615 does not fit in int"},
		{"Uint8", -1, "-1 does not fit in uint8"},
		{"Uint8", "-1", `cannot parse "-1" as uint8: invalid syntax`},
		{"Uint8", 256, "256 does not fit in uint8"},
		{"Float32", math.MaxFloat64, "does not fit in float32"},
		{"Int", "abc", `cannot parse "abc" as int: invalid syntax`},
		{"Bool", "maybe", `cannot parse "maybe" as bool: invalid syntax`},
		{"Bool", 1, "cannot assign int to bool"},
		{"Duration", "soon", `cannot parse "soon" as time.Duration`},
		{"Time", "yesterday", `cannot parse "yesterday" as time.Time`},
		{"IntPtr", "abc", `cannot parse "abc" as int: invalid syntax`},
		{"Chan", "x", "cannot parse a string into chan int"},
		{"Int", true, "cannot assign bool to int"},
	}

	for _, tt := range tests {
		var dst target
		err := setField(t, &dst, tt.field, tt.in)
		if err == nil {
			t.Errorf("%s = %#v: expected an error", tt.field, tt.in)
			continue
		}
		if want := "field " + tt.field + ": "; !strings.HasPrefix(err.Error(), want) {
			t.Errorf("%s: %q should start with %q", tt.field, err, want)
		}
		if !strings.Contains(err.Error(), tt.message) {
			t.Errorf("%s = %#v: got %q, want it to contain %q", tt.field, tt.in, err, tt.message)
		}
		if !reflect.DeepEqual(dst, target{}) {
			t.Errorf("%s = %#v: the field changed despite the error", tt.field, tt.in)
		}
	}
}

func TestSetValue_NilResetsTheField(t *testing.T) {
	seven := 7
	dst := target{String: "set", Int: 3, IntPtr: &seven, Any: "x"}

	for _, field := range []string{"String", "Int", "IntPtr", "Any"} {
		if err := setField(t, &dst, field, nil); err != nil {
			t.Fatal(err)
		}
	}
	if !reflect.DeepEqual(dst, target{}) {
		t.Errorf("got %+v", dst)
	}
}

func TestSetValue_NotSettable(t *testing.T) {
	type S struct {
		Name   string
		hidden string
		ByKey  map[string]leaf
	}

	tests := map[string]struct {
		v    any
		path string
		opts []WalkerOption
	}{
		"struct passed by value": {S{}, "Name", nil},
		"unexported field":       {&S{hidden: "x"}, "hidden", []WalkerOption{WithUnexported(true)}},
		"struct stored in a map": {&S{ByKey: map[string]leaf{"a": {}}}, "ByKey[a].Name", []WalkerOption{WithMapEntries(true)}},
	}

	for name, tt := range tests {
		t.Run(name, func(t *testing.T) {
			w := New(tt.opts...)
			w.HandleAll(func(ctx FieldContext) error {
				if ctx.Path == tt.path {
					return ctx.SetValue("x")
				}
				return nil
			})

			err := w.Walk(tt.v)
			if !errors.Is(err, ErrNotSettable) {
				t.Fatalf("got %v, want ErrNotSettable", err)
			}
			if want := "field " + tt.path + ": not settable"; !strings.HasSuffix(err.Error(), want) {
				t.Errorf("got %q, want it to end with %q", err, want)
			}
		})
	}
}
