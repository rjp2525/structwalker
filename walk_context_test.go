package structwalker

import (
	"context"
	"errors"
	"reflect"
	"testing"
)

type ctxKey struct{}

func TestWalkContext_StopsWhenCancelled(t *testing.T) {
	type S struct {
		A, B, C string
	}

	ctx, cancel := context.WithCancel(context.Background())
	var seen []string
	w := New()
	w.HandleAll(func(fc FieldContext) error {
		seen = append(seen, fc.Path)
		cancel()
		return nil
	})

	err := w.WalkContext(ctx, &S{})
	if !errors.Is(err, context.Canceled) {
		t.Errorf("got %v, want context.Canceled", err)
	}
	if len(seen) != 1 {
		t.Errorf("handlers ran on %v after cancellation", seen)
	}
}

func TestWalkContext_OneWalkerManyContexts(t *testing.T) {
	type S struct {
		Name string
	}

	w := New()
	w.HandleAll(func(FieldContext) error { return nil })

	done, cancel := context.WithCancel(context.Background())
	cancel()

	if err := w.WalkContext(done, &S{}); !errors.Is(err, context.Canceled) {
		t.Errorf("got %v, want context.Canceled", err)
	}
	if err := w.WalkContext(context.Background(), &S{}); err != nil {
		t.Errorf("a finished context leaked into the next walk: %v", err)
	}
	if err := w.Walk(&S{}); err != nil {
		t.Errorf("Walk: %v", err)
	}
}

func TestFieldContext_Context(t *testing.T) {
	type Inner struct {
		Name string
	}
	type S struct {
		Inner Inner
	}

	ctx := context.WithValue(context.Background(), ctxKey{}, "request-1")
	var got []any
	w := New()
	w.HandleAll(func(fc FieldContext) error {
		got = append(got, fc.Context().Value(ctxKey{}))
		return nil
	})

	if err := w.WalkValueContext(ctx, reflect.ValueOf(&S{})); err != nil {
		t.Fatal(err)
	}
	if len(got) != 2 || got[0] != "request-1" || got[1] != "request-1" {
		t.Errorf("got %v", got)
	}

	if (FieldContext{}).Context() == nil {
		t.Error("Context must not return nil")
	}
}

func TestWalkContext_CancellationBeatsCollectedErrors(t *testing.T) {
	type S struct {
		A, B string
	}

	ctx, cancel := context.WithCancel(context.Background())
	w := New(WithErrorMode(CollectErrors))
	w.HandleAll(func(FieldContext) error {
		cancel()
		return errors.New("invalid")
	})

	if err := w.WalkContext(ctx, &S{}); !errors.Is(err, context.Canceled) {
		t.Errorf("got %v, want context.Canceled", err)
	}
}
