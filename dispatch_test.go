package structwalker

import (
	"errors"
	"reflect"
	"slices"
	"sync"
	"testing"
)

func record(calls *[]string, name string) HandlerFunc {
	return func(FieldContext) error {
		*calls = append(*calls, name)
		return nil
	}
}

func TestDispatch_PriorityAppliesAcrossTagKeys(t *testing.T) {
	type S struct {
		Name string `a:"x" b:"y"`
	}

	var calls []string
	w := New()
	w.Handle("a", record(&calls, "a"))
	w.Handle("b", record(&calls, "b"), WithPriority(10))
	w.HandleAll(record(&calls, "all"), WithPriority(50))

	if err := w.Walk(&S{}); err != nil {
		t.Fatal(err)
	}
	if want := []string{"b", "all", "a"}; !slices.Equal(calls, want) {
		t.Errorf("got %v, want %v", calls, want)
	}
}

func TestDispatch_TiesFollowTagOrderThenRegistration(t *testing.T) {
	type S struct {
		Name string `b:"y" a:"x"`
	}

	var calls []string
	w := New()
	w.HandleAll(record(&calls, "all"))
	w.Handle("a", record(&calls, "a1"))
	w.Handle("b", record(&calls, "b"))
	w.Handle("a", record(&calls, "a2"))

	if err := w.Walk(&S{}); err != nil {
		t.Fatal(err)
	}
	if want := []string{"b", "a1", "a2", "all"}; !slices.Equal(calls, want) {
		t.Errorf("got %v, want %v", calls, want)
	}
}

func TestDispatch_EqualPrioritiesKeepRegistrationOrder(t *testing.T) {
	type S struct {
		Name string `tag:"x"`
	}

	var calls, want []string
	w := New()
	for i := range 40 {
		name := string(rune('A' + i))
		want = append(want, name)
		w.Handle("tag", record(&calls, name))
	}

	if err := w.Walk(&S{}); err != nil {
		t.Fatal(err)
	}
	if !slices.Equal(calls, want) {
		t.Errorf("got %v, want %v", calls, want)
	}
}

func TestDispatch_DuplicateTagKeyRunsOnce(t *testing.T) {
	st := reflect.StructOf([]reflect.StructField{{
		Name: "Name",
		Type: reflect.TypeFor[string](),
		Tag:  `tag:"first" tag:"second"`,
	}})

	var names []string
	w := New()
	w.Handle("tag", func(ctx FieldContext) error {
		names = append(names, ctx.Tag.Name)
		return nil
	})

	if err := w.WalkValue(reflect.New(st)); err != nil {
		t.Fatal(err)
	}
	if want := []string{"first"}; !slices.Equal(names, want) {
		t.Errorf("got %v, want %v", names, want)
	}
}

func TestDispatch_SkipChildrenLetsOtherHandlersRun(t *testing.T) {
	type Inner struct {
		Value string `a:"v"`
	}
	type S struct {
		Inner Inner `a:"inner" b:"inner"`
	}

	var calls []string
	w := New()
	w.Handle("a", func(ctx FieldContext) error {
		calls = append(calls, "a:"+ctx.Path)
		if ctx.Path == "Inner" {
			return ErrSkipChildren
		}
		return nil
	})
	w.Handle("b", record(&calls, "b"))
	w.Handle("b", record(&calls, "b-after"), WithPhase(AfterChildren))

	if err := w.Walk(&S{}); err != nil {
		t.Fatal(err)
	}
	if want := []string{"a:Inner", "b", "b-after"}; !slices.Equal(calls, want) {
		t.Errorf("got %v, want %v", calls, want)
	}
}

func TestDispatch_SkipFieldStopsEverythingOnTheField(t *testing.T) {
	type Inner struct {
		Value string `a:"v"`
	}
	type S struct {
		Inner Inner  `a:"inner" b:"inner"`
		Next  string `b:"next"`
	}

	var calls []string
	w := New()
	w.Handle("a", func(ctx FieldContext) error {
		calls = append(calls, "a:"+ctx.Path)
		return ErrSkipField
	})
	w.Handle("b", func(ctx FieldContext) error {
		calls = append(calls, "b:"+ctx.Path)
		return nil
	})
	w.HandleAll(func(ctx FieldContext) error {
		calls = append(calls, "after:"+ctx.Path)
		return nil
	}, WithPhase(AfterChildren))

	if err := w.Walk(&S{}); err != nil {
		t.Fatal(err)
	}
	if want := []string{"a:Inner", "b:Next", "after:Next"}; !slices.Equal(calls, want) {
		t.Errorf("got %v, want %v", calls, want)
	}
}

func TestDispatch_SkipChildrenAfterChildrenIsNotAnError(t *testing.T) {
	type S struct {
		Name string `tag:"x"`
	}

	for _, mode := range []ErrorMode{StopOnError, CollectErrors, SkipOnError} {
		w := New(WithErrorMode(mode))
		w.Handle("tag", func(FieldContext) error { return ErrSkipChildren }, WithPhase(AfterChildren))
		if err := w.Walk(&S{}); err != nil {
			t.Errorf("mode %d: got %v", mode, err)
		}
	}
}

func TestDispatch_WrappedSentinelsAreHonoured(t *testing.T) {
	type S struct {
		A string `tag:"a"`
		B string `tag:"b"`
	}

	var calls []string
	w := New()
	w.Handle("tag", func(ctx FieldContext) error {
		return errors.Join(errors.New("context"), ErrSkipField)
	})
	w.Handle("tag", record(&calls, "second"))

	if err := w.Walk(&S{}); err != nil {
		t.Fatal(err)
	}
	if len(calls) != 0 {
		t.Errorf("handlers ran after ErrSkipField: %v", calls)
	}
}

func TestUse_AppliesToHandlersRegisteredEarlier(t *testing.T) {
	type S struct {
		Name string `tag:"x"`
	}

	var calls []string
	w := New()
	w.Handle("tag", record(&calls, "handler"))
	w.Use(func(ctx FieldContext, next HandlerFunc) error {
		calls = append(calls, "middleware")
		return next(ctx)
	})

	if err := w.Walk(&S{}); err != nil {
		t.Fatal(err)
	}
	if want := []string{"middleware", "handler"}; !slices.Equal(calls, want) {
		t.Errorf("got %v, want %v", calls, want)
	}
}

func TestWalker_RegisterWhileWalking(t *testing.T) {
	type S struct {
		A string `tag:"a"`
		B string `tag:"b"`
	}

	w := New()
	w.Handle("tag", func(FieldContext) error { return nil })

	var wg sync.WaitGroup
	for range 8 {
		wg.Go(func() {
			for range 200 {
				if err := w.Walk(&S{}); err != nil {
					t.Error(err)
					return
				}
			}
		})
	}
	wg.Go(func() {
		for range 50 {
			w.Handle("tag", func(FieldContext) error { return nil })
			w.HandleAll(func(FieldContext) error { return nil })
			w.Use(func(ctx FieldContext, next HandlerFunc) error { return next(ctx) })
		}
	})
	wg.Wait()
}

func TestWalker_HandlerAddedAfterFirstWalkIsUsed(t *testing.T) {
	type S struct {
		Name string `tag:"x"`
	}

	var calls []string
	w := New()
	w.Handle("tag", record(&calls, "first"))
	if err := w.Walk(&S{}); err != nil {
		t.Fatal(err)
	}

	w.Handle("tag", record(&calls, "second"))
	calls = nil
	if err := w.Walk(&S{}); err != nil {
		t.Fatal(err)
	}
	if want := []string{"first", "second"}; !slices.Equal(calls, want) {
		t.Errorf("got %v, want %v", calls, want)
	}
}
