package structwalker

import (
	"bytes"
	"errors"
	"log/slog"
	"strings"
	"testing"
	"time"
)

func TestMiddleware_Order(t *testing.T) {
	type S struct {
		Name string `tag:"name"`
	}

	var order []string
	w := New()
	w.Use(func(ctx FieldContext, next HandlerFunc) error {
		order = append(order, "mw1:before")
		err := next(ctx)
		order = append(order, "mw1:after")
		return err
	})
	w.Use(func(ctx FieldContext, next HandlerFunc) error {
		order = append(order, "mw2:before")
		err := next(ctx)
		order = append(order, "mw2:after")
		return err
	})
	w.Handle("tag", func(ctx FieldContext) error {
		order = append(order, "handler")
		return nil
	})

	err := w.Walk(&S{})
	if err != nil {
		t.Fatal(err)
	}

	expected := []string{"mw1:before", "mw2:before", "handler", "mw2:after", "mw1:after"}
	if len(order) != len(expected) {
		t.Fatalf("order = %v, want %v", order, expected)
	}
	for i, v := range expected {
		if order[i] != v {
			t.Errorf("order[%d] = %q, want %q", i, order[i], v)
		}
	}
}

func TestWithRecovery(t *testing.T) {
	type S struct {
		Name string `tag:"name"`
	}

	w := New()
	w.Use(WithRecovery())
	w.Handle("tag", func(ctx FieldContext) error {
		panic("test panic")
	})

	err := w.Walk(&S{})
	if err == nil {
		t.Fatal("expected error from recovered panic")
	}
	var walkErr *WalkError
	if !errors.As(err, &walkErr) {
		t.Fatalf("expected WalkError, got %T: %v", err, err)
	}
	if !strings.Contains(walkErr.Err.Error(), "panic") {
		t.Errorf("error = %q, want to contain 'panic'", walkErr.Err.Error())
	}
}

func TestWithTiming(t *testing.T) {
	type S struct {
		Name string `tag:"name"`
	}

	var recorded bool
	w := New()
	w.Use(WithTiming(func(path string, tagKey string, duration time.Duration) {
		recorded = true
		if path != "Name" {
			t.Errorf("path = %q, want %q", path, "Name")
		}
		if tagKey != "tag" {
			t.Errorf("tagKey = %q, want %q", tagKey, "tag")
		}
		if duration < 0 {
			t.Error("duration should be non-negative")
		}
	}))
	w.Handle("tag", func(ctx FieldContext) error {
		return nil
	})

	err := w.Walk(&S{Name: "test"})
	if err != nil {
		t.Fatal(err)
	}
	if !recorded {
		t.Error("timing callback was not invoked")
	}
}

func TestWithLogging(t *testing.T) {
	type S struct {
		Name string `tag:"name"`
	}

	var buf bytes.Buffer
	logger := slog.New(slog.NewTextHandler(&buf, &slog.HandlerOptions{Level: slog.LevelDebug}))

	w := New()
	w.Use(WithLogging(logger))
	w.Handle("tag", func(ctx FieldContext) error {
		return nil
	})

	err := w.Walk(&S{Name: "test"})
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(buf.String(), "visiting field") {
		t.Errorf("log output = %q, expected to contain 'visiting field'", buf.String())
	}
}

func TestWithLogging_Error(t *testing.T) {
	type S struct {
		Name string `tag:"name"`
	}

	var buf bytes.Buffer
	logger := slog.New(slog.NewTextHandler(&buf, &slog.HandlerOptions{Level: slog.LevelDebug}))

	w := New()
	w.Use(WithLogging(logger))
	w.Handle("tag", func(ctx FieldContext) error {
		return errors.New("handler error")
	})

	_ = w.Walk(&S{Name: "test"})
	if !strings.Contains(buf.String(), "handler error") {
		t.Errorf("log output = %q, expected to contain 'handler error'", buf.String())
	}
}

func TestMiddleware_NoMiddleware(t *testing.T) {
	type S struct {
		Name string `tag:"name"`
	}

	called := false
	w := New()
	w.Handle("tag", func(ctx FieldContext) error {
		called = true
		return nil
	})

	err := w.Walk(&S{})
	if err != nil {
		t.Fatal(err)
	}
	if !called {
		t.Error("handler was not called")
	}
}
