package structwalker

import (
	"errors"
	"fmt"
	"reflect"
	"sync"
	"sync/atomic"
)

// Walker walks structs and calls the handlers registered on it. Create one
// with New, register handlers, then call Walk as often as you like.
//
// A Walker is safe for concurrent use. Handlers and middleware can be added
// while walks are running. A walk uses the set that was registered when it
// started.
type Walker struct {
	mu       sync.Mutex
	registry atomic.Pointer[registry]
	opts     walkerOptions
}

// New creates a Walker configured with the given options.
func New(opts ...WalkerOption) *Walker {
	w := &Walker{opts: defaultOptions()}
	for _, opt := range opts {
		opt(&w.opts)
	}
	w.registry.Store(newRegistry(nil, nil))
	return w
}

// Handle registers a handler for fields that carry the given tag key.
func (w *Walker) Handle(tagKey string, fn HandlerFunc, opts ...HandlerOption) {
	h := newHandler(tagKey, false, fn, opts)
	w.update(func(r *registry) *registry { return r.withHandler(h) })
}

// HandleAll registers a handler that runs on every field, tagged or not.
func (w *Walker) HandleAll(fn HandlerFunc, opts ...HandlerOption) {
	h := newHandler("", true, fn, opts)
	w.update(func(r *registry) *registry { return r.withHandler(h) })
}

func (w *Walker) update(fn func(*registry) *registry) {
	w.mu.Lock()
	defer w.mu.Unlock()
	w.registry.Store(fn(w.registry.Load()))
}

// Walk processes the given struct. Pass a pointer for fields to be settable.
func (w *Walker) Walk(v any) error {
	return w.WalkValue(reflect.ValueOf(v))
}

// WalkValue processes a reflect.Value directly.
func (w *Walker) WalkValue(v reflect.Value) error {
	for v.Kind() == reflect.Pointer {
		if v.IsNil() {
			return nil
		}
		v = v.Elem()
	}

	if v.Kind() != reflect.Struct {
		return fmt.Errorf("structwalker: expected struct, got %s", v.Kind())
	}

	run := walk{Walker: w, registry: w.registry.Load()}
	if w.opts.followPointers {
		run.visited = make(map[uintptr]bool)
	}
	if err := run.structFields(v, "", 0); err != nil {
		return err
	}
	if len(run.errs) > 0 {
		return &MultiError{Errors: run.errs}
	}
	return nil
}

// walk is the state of a single Walk call.
type walk struct {
	*Walker
	registry *registry
	visited  map[uintptr]bool
	errs     []*WalkError
}

// report handles an error according to the walker's error mode. A non-nil
// result stops the walk.
func (w *walk) report(err *WalkError) error {
	if w.opts.errorMode == CollectErrors {
		w.errs = append(w.errs, err)
		return nil
	}
	return err
}

func (w *walk) structFields(v reflect.Value, basePath string, depth int) error {
	if depth > w.opts.maxDepth {
		return w.report(&WalkError{Path: basePath, Err: ErrMaxDepth})
	}

	plan := w.registry.plan(v.Type(), &w.opts)

	for i := range plan.fields {
		if err := w.opts.ctx.Err(); err != nil {
			return err
		}

		fp := &plan.fields[i]
		fc := FieldContext{
			AllTags: fp.tags,
			Field:   fp.field,
			Value:   v.Field(fp.index),
			Path:    joinPath(basePath, fp.field.Name),
			Depth:   depth,
			Parent:  v,
			Index:   fp.index,
			Walker:  w.Walker,
		}

		skipChildren, err := w.dispatch(fp.before, fc)
		if errors.Is(err, ErrSkipField) {
			continue
		}
		if err != nil {
			return err
		}

		if !skipChildren {
			if err := w.recurse(fc.Value, fp, fc.Path, depth); err != nil {
				return err
			}
		}

		if _, err := w.dispatch(fp.after, fc); err != nil && !errors.Is(err, ErrSkipField) {
			return err
		}
	}
	return nil
}

// dispatch runs a field's handlers for one phase. It returns ErrSkipField as
// soon as a handler asks for it, and any other error only when that error
// should stop the walk.
func (w *walk) dispatch(steps []step, fc FieldContext) (skipChildren bool, err error) {
	for i := range steps {
		s := &steps[i]
		fc.Tag = s.tag
		if s.filter != nil && !s.filter(fc) {
			continue
		}

		err := s.run(fc)
		switch {
		case err == nil:
		case errors.Is(err, ErrSkipField):
			return false, ErrSkipField
		case errors.Is(err, ErrSkipChildren):
			skipChildren = true
		case w.opts.errorMode == SkipOnError:
		default:
			if err := w.report(&WalkError{Path: fc.Path, TagKey: s.tag.Key, Field: fc.Field.Name, Err: err}); err != nil {
				return false, err
			}
		}
	}
	return skipChildren, nil
}

func (w *walk) recurse(fieldVal reflect.Value, fp *fieldPlan, path string, depth int) error {
	val := fieldVal
	if fp.isPtr && w.opts.followPointers {
		if val.IsNil() {
			return nil
		}
		val = val.Elem()

		ptr := fieldVal.Pointer()
		if w.visited[ptr] {
			return nil
		}
		w.visited[ptr] = true
		defer delete(w.visited, ptr)
	}

	if val.Kind() == reflect.Struct && ((fp.isStruct && !fp.isPtr) || (fp.isPtr && w.opts.followPointers)) {
		return w.structFields(val, path, depth+1)
	}

	if fp.isSlice && w.opts.sliceElements {
		for i := range val.Len() {
			if err := w.element(val.Index(i), fmt.Sprintf("%s[%d]", path, i), depth); err != nil {
				return err
			}
		}
	}

	if fp.isMap && w.opts.mapEntries {
		for iter := val.MapRange(); iter.Next(); {
			if err := w.element(iter.Value(), fmt.Sprintf("%s[%v]", path, iter.Key().Interface()), depth); err != nil {
				return err
			}
		}
	}

	return nil
}

func (w *walk) element(elem reflect.Value, path string, depth int) error {
	if elem.Kind() == reflect.Pointer {
		if elem.IsNil() {
			return nil
		}
		if w.visited != nil {
			ptr := elem.Pointer()
			if w.visited[ptr] {
				return nil
			}
			w.visited[ptr] = true
			defer delete(w.visited, ptr)
		}
		elem = elem.Elem()
	}

	if elem.Kind() == reflect.Struct {
		return w.structFields(elem, path, depth+1)
	}
	return nil
}

func joinPath(base, name string) string {
	if base == "" {
		return name
	}
	return base + "." + name
}
