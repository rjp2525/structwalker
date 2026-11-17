package structwalker

import (
	"bytes"
	"errors"
	"log/slog"
	"slices"
	"strings"
	"testing"
)

var errRequired = errors.New("required")

func requireNonZero(ctx FieldContext) error {
	if ctx.IsZero() {
		return errRequired
	}
	return nil
}

func errorPaths(t *testing.T, err error) []string {
	t.Helper()
	multi, ok := errors.AsType[*MultiError](err)
	if !ok {
		t.Fatalf("got %T (%v), want *MultiError", err, err)
	}
	paths := make([]string, len(multi.Errors))
	for i, e := range multi.Errors {
		paths[i] = e.Path
	}
	return paths
}

func TestCollectErrors_NestedErrorsAreFlat(t *testing.T) {
	type Address struct {
		Street string `validate:"required"`
		City   string `validate:"required"`
	}
	type User struct {
		Name    string `validate:"required"`
		Address Address
		Email   string `validate:"required"`
	}

	w := New(WithErrorMode(CollectErrors))
	w.Handle("validate", requireNonZero)

	err := w.Walk(&User{})
	want := []string{"Name", "Address.Street", "Address.City", "Email"}
	if got := errorPaths(t, err); !slices.Equal(got, want) {
		t.Errorf("got %v, want %v", got, want)
	}
	if !errors.Is(err, errRequired) {
		t.Error("errors.Is should reach the handler's error")
	}
}

func TestCollectErrors_KeepsGoingThroughSlicesAndMaps(t *testing.T) {
	type Item struct {
		Name string `validate:"required"`
	}
	type Cart struct {
		Items []Item
		ByKey map[string]*Item
	}

	w := New(WithErrorMode(CollectErrors), WithSliceElements(true), WithMapEntries(true))
	w.Handle("validate", requireNonZero)

	err := w.Walk(&Cart{
		Items: []Item{{}, {Name: "ok"}, {}},
		ByKey: map[string]*Item{"a": {}},
	})
	want := []string{"Items[0].Name", "Items[2].Name", "ByKey[a].Name"}
	if got := errorPaths(t, err); !slices.Equal(got, want) {
		t.Errorf("got %v, want %v", got, want)
	}
}

func TestErrMaxDepth(t *testing.T) {
	type C struct {
		Leaf string `tag:"leaf"`
	}
	type B struct{ C C }
	type A struct {
		B    B
		Name string `tag:"name"`
	}

	t.Run("stops the walk by default", func(t *testing.T) {
		w := New(WithMaxDepth(1))
		err := w.Walk(&A{})
		if !errors.Is(err, ErrMaxDepth) {
			t.Fatalf("got %v, want ErrMaxDepth", err)
		}
		if walkErr, ok := errors.AsType[*WalkError](err); !ok || walkErr.Path != "B.C" {
			t.Errorf("got %v, want a WalkError at B.C", err)
		}
	})

	t.Run("is collected and the walk carries on", func(t *testing.T) {
		var seen []string
		w := New(WithMaxDepth(1), WithErrorMode(CollectErrors))
		w.Handle("tag", func(ctx FieldContext) error {
			seen = append(seen, ctx.Path)
			return nil
		})

		err := w.Walk(&A{})
		if got := errorPaths(t, err); !slices.Equal(got, []string{"B.C"}) {
			t.Errorf("got %v", got)
		}
		if !errors.Is(err, ErrMaxDepth) {
			t.Errorf("got %v, want ErrMaxDepth", err)
		}
		if !slices.Equal(seen, []string{"Name"}) {
			t.Errorf("handlers saw %v, want [Name]", seen)
		}
	})
}

func TestMultiError_Message(t *testing.T) {
	multi := &MultiError{Errors: []*WalkError{
		{Path: "Name", TagKey: "validate", Err: errRequired},
		{Path: "Address.City", Err: errRequired},
	}}

	want := "structwalker: 2 errors:\n  - Name (tag \"validate\"): required\n  - Address.City: required"
	if got := multi.Error(); got != want {
		t.Errorf("got %q, want %q", got, want)
	}

	single := &MultiError{Errors: multi.Errors[:1]}
	if got, want := single.Error(), `structwalker: Name (tag "validate"): required`; got != want {
		t.Errorf("got %q, want %q", got, want)
	}
}

func TestWithRecovery_PanicError(t *testing.T) {
	type S struct {
		Name string `tag:"x"`
	}

	boom := errors.New("boom")
	w := New()
	w.Use(WithRecovery())
	w.Handle("tag", func(FieldContext) error { panic(boom) })

	err := w.Walk(&S{})
	panicErr, ok := errors.AsType[*PanicError](err)
	if !ok {
		t.Fatalf("got %T (%v), want *PanicError", err, err)
	}
	if panicErr.Value != boom {
		t.Errorf("Value = %v", panicErr.Value)
	}
	if !bytes.Contains(panicErr.Stack, []byte("errors_test.go")) {
		t.Error("Stack should point at the panicking handler")
	}
	if !errors.Is(err, boom) {
		t.Error("errors.Is should reach an error used as the panic value")
	}
	if got, want := err.Error(), `structwalker: Name (tag "tag"): panic: boom`; got != want {
		t.Errorf("got %q, want %q", got, want)
	}
}

func TestWithLogging_FlowControlIsNotAnError(t *testing.T) {
	type S struct {
		Name string `tag:"x"`
	}

	var buf bytes.Buffer
	w := New()
	w.Use(WithLogging(slog.New(slog.NewTextHandler(&buf, &slog.HandlerOptions{Level: slog.LevelDebug}))))
	w.Handle("tag", func(FieldContext) error { return ErrSkipField })

	if err := w.Walk(&S{}); err != nil {
		t.Fatal(err)
	}
	if strings.Contains(buf.String(), "level=ERROR") {
		t.Errorf("ErrSkipField was logged as an error:\n%s", buf.String())
	}
}
