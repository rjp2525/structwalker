package structwalker

import (
	"cmp"
	"context"
	"errors"
	"fmt"
	"iter"
	"reflect"
	"slices"
	"strconv"
	"sync"
	"sync/atomic"
	"unsafe"
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

// Walk visits every field of the struct v points to and runs the handlers
// that apply. Pass a pointer if handlers need to set fields.
func (w *Walker) Walk(v any) error {
	return w.WalkValueContext(w.opts.ctx, reflect.ValueOf(v))
}

// WalkContext is Walk with a context. The walk stops with ctx.Err() once ctx
// is done and handlers can read ctx from FieldContext.Context.
func (w *Walker) WalkContext(ctx context.Context, v any) error {
	return w.WalkValueContext(ctx, reflect.ValueOf(v))
}

// WalkValue is Walk for a value you already hold as a reflect.Value.
func (w *Walker) WalkValue(v reflect.Value) error {
	return w.WalkValueContext(w.opts.ctx, v)
}

// WalkValueContext is WalkContext for a value you already hold as a
// reflect.Value.
func (w *Walker) WalkValueContext(ctx context.Context, v reflect.Value) error {
	run := walk{Walker: w, registry: w.registry.Load(), ctx: ctx}
	if err := run.start(v); err != nil {
		return err
	}
	if len(run.errs) > 0 {
		return &MultiError{Errors: run.errs}
	}
	return nil
}

// Fields returns an iterator over every field Walk would visit, in the same
// order. Registered handlers and middleware do not run and FieldContext.Tag is
// empty, so read tags with FieldContext.Lookup.
//
// The iterator yields nothing if v is not a struct or a pointer to one. It
// does not descend past the walker's depth limit.
func (w *Walker) Fields(v any) iter.Seq[FieldContext] {
	return func(yield func(FieldContext) bool) {
		run := walk{Walker: w, registry: w.registry.Load(), ctx: context.Background(), yield: yield}
		_ = run.start(reflect.ValueOf(v))
	}
}

var defaultWalker = New()

// Fields returns an iterator over the fields of v without setting up a
// Walker first:
//
//	for f := range structwalker.Fields(&cfg) {
//	    if tag, ok := f.Lookup("env"); ok {
//	        ...
//	    }
//	}
//
// See Walker.Fields for the details. Options apply to this call only.
func Fields(v any, opts ...WalkerOption) iter.Seq[FieldContext] {
	if len(opts) == 0 {
		return defaultWalker.Fields(v)
	}
	return New(opts...).Fields(v)
}

var errStopIteration = errors.New("stop iteration")

// walk is the state of a single Walk call.
type walk struct {
	*Walker
	registry *registry
	ctx      context.Context
	root     visit
	visited  map[visit]struct{}
	errs     []*WalkError
	yield    func(FieldContext) bool
}

func (w *walk) start(v reflect.Value) error {
	for v.Kind() == reflect.Pointer {
		if v.IsNil() {
			return nil
		}
		w.root = visit{v.Type(), v.UnsafePointer()}
		v = v.Elem()
	}

	if v.Kind() != reflect.Struct {
		return fmt.Errorf("structwalker: expected struct, got %s", v.Kind())
	}
	return w.structFields(v, "", 0)
}

// visit identifies a pointer, slice or map the walk is currently inside. The
// type is part of the key because a struct and its first field share an address.
type visit struct {
	typ reflect.Type
	ptr unsafe.Pointer
}

// enter marks v as being walked. It reports false when the walk is already
// inside v, which means v is part of a cycle.
func (w *walk) enter(v reflect.Value) (visit, bool) {
	key := visit{v.Type(), v.UnsafePointer()}
	if key == w.root {
		return key, false
	}
	if _, seen := w.visited[key]; seen {
		return key, false
	}
	if w.visited == nil {
		w.visited = make(map[visit]struct{})
	}
	w.visited[key] = struct{}{}
	return key, true
}

// report handles an error according to the walker's error mode. A non-nil
// result stops the walk.
func (w *walk) report(err *WalkError) error {
	if w.yield != nil {
		return nil
	}
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
		if err := w.ctx.Err(); err != nil {
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
			ctx:     w.ctx,
		}

		if w.yield != nil {
			if fp.visit && !w.yield(fc) {
				return errStopIteration
			}
			if fp.descend {
				if err := w.descend(fc.Value, fc.Path, depth); err != nil {
					return err
				}
			}
			continue
		}

		skipChildren, err := w.dispatch(fp.before, fc)
		if errors.Is(err, ErrSkipField) {
			continue
		}
		if err != nil {
			return err
		}

		if !skipChildren && fp.descend {
			if err := w.descend(fc.Value, fc.Path, depth); err != nil {
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
// soon as a handler asks for it. Any other error comes back only when it
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

// descend walks whatever structs can be reached from v. The path and depth are
// those of the field v came from.
func (w *walk) descend(v reflect.Value, path string, depth int) error {
	switch v.Kind() {
	case reflect.Pointer:
		if !w.opts.followPointers || v.IsNil() {
			return nil
		}
		key, ok := w.enter(v)
		if !ok {
			return nil
		}
		defer delete(w.visited, key)
		return w.descend(v.Elem(), path, depth)

	case reflect.Interface:
		if !w.opts.interfaces || v.IsNil() {
			return nil
		}
		return w.descend(v.Elem(), path, depth)

	case reflect.Struct:
		if _, leaf := w.opts.leafTypes[v.Type()]; leaf {
			return nil
		}
		return w.structFields(v, path, depth+1)

	case reflect.Array:
		if !w.opts.sliceElements {
			return nil
		}
		return w.elements(v, path, depth)

	case reflect.Slice:
		if !w.opts.sliceElements || v.Len() == 0 {
			return nil
		}
		key, ok := w.enter(v)
		if !ok {
			return nil
		}
		defer delete(w.visited, key)
		return w.elements(v, path, depth)

	case reflect.Map:
		if !w.opts.mapEntries || v.Len() == 0 {
			return nil
		}
		key, ok := w.enter(v)
		if !ok {
			return nil
		}
		defer delete(w.visited, key)
		return w.entries(v, path, depth)
	}
	return nil
}

func (w *walk) elements(v reflect.Value, path string, depth int) error {
	for i := range v.Len() {
		if err := w.descend(v.Index(i), path+"["+strconv.Itoa(i)+"]", depth); err != nil {
			return err
		}
	}
	return nil
}

func (w *walk) entries(v reflect.Value, path string, depth int) error {
	keys := v.MapKeys()
	slices.SortFunc(keys, compareMapKeys)
	for _, key := range keys {
		if err := w.descend(v.MapIndex(key), fmt.Sprintf("%s[%v]", path, key), depth); err != nil {
			return err
		}
	}
	return nil
}

func compareMapKeys(a, b reflect.Value) int {
	switch {
	case a.Kind() == reflect.String:
		return cmp.Compare(a.String(), b.String())
	case a.CanInt():
		return cmp.Compare(a.Int(), b.Int())
	case a.CanUint():
		return cmp.Compare(a.Uint(), b.Uint())
	case a.CanFloat():
		return cmp.Compare(a.Float(), b.Float())
	}
	return cmp.Compare(fmt.Sprint(a), fmt.Sprint(b))
}

func joinPath(base, name string) string {
	if base == "" {
		return name
	}
	return base + "." + name
}
