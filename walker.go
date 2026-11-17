package structwalker

import (
	"errors"
	"fmt"
	"reflect"
)

// Walker is the entry point. Register handlers for tag keys, then Walk any struct.
type Walker struct {
	handlers   *handlerRegistry
	middleware []Middleware
	cache      *typeCache
	opts       walkerOptions
}

// New creates a Walker configured with the given options.
func New(opts ...WalkerOption) *Walker {
	o := defaultOptions()
	for _, opt := range opts {
		opt(&o)
	}
	return &Walker{
		handlers: newHandlerRegistry(),
		cache:    newTypeCache(),
		opts:     o,
	}
}

// Handle registers a handler for a specific tag key.
func (w *Walker) Handle(tagKey string, fn HandlerFunc, opts ...HandlerOption) {
	w.handlers.add(tagKey, fn, opts...)
	w.cache.clear()
}

// HandleAll registers a handler invoked for every field regardless of tags.
func (w *Walker) HandleAll(fn HandlerFunc, opts ...HandlerOption) {
	w.handlers.addGlobal(fn, opts...)
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

	var visited map[uintptr]bool
	if w.opts.followPointers {
		visited = make(map[uintptr]bool)
	}
	return w.walkStruct(v, "", 0, visited)
}

func (w *Walker) walkStruct(v reflect.Value, basePath string, depth int, visited map[uintptr]bool) error {
	if depth > w.opts.maxDepth {
		return fmt.Errorf("structwalker: max depth %d exceeded at %s", w.opts.maxDepth, basePath)
	}

	if err := w.checkContext(); err != nil {
		return err
	}

	t := v.Type()
	meta := w.cache.get(t, w.handlers.keys(), w.opts.tagParser)

	var multi *MultiError
	if w.opts.errorMode == CollectErrors {
		multi = &MultiError{}
	}

	for i, fm := range meta.Fields {
		if err := w.checkContext(); err != nil {
			return err
		}

		if !fm.StructField.IsExported() && !w.opts.unexported {
			continue
		}

		fieldVal := v.Field(i)
		path := joinPath(basePath, fm.StructField.Name)

		fc := FieldContext{
			AllTags: fm.Tags,
			Field:   fm.StructField,
			Value:   fieldVal,
			Path:    path,
			Depth:   depth,
			Parent:  v,
			Index:   i,
			Walker:  w,
		}

		skipChildren := false

		if err := w.dispatchHandlers(fc, fm, BeforeChildren, multi); err != nil {
			if errors.Is(err, ErrSkipField) {
				continue
			}
			if errors.Is(err, ErrSkipChildren) {
				skipChildren = true
			} else if multi == nil {
				return err
			}
		}

		if err := w.dispatchGlobal(fc, BeforeChildren, multi); err != nil {
			if errors.Is(err, ErrSkipField) {
				continue
			}
			if errors.Is(err, ErrSkipChildren) {
				skipChildren = true
			} else if multi == nil {
				return err
			}
		}

		if !skipChildren {
			if err := w.recurseField(fieldVal, fm, path, depth, visited); err != nil {
				if multi != nil {
					if walkErr, ok := err.(*WalkError); ok {
						multi.add(walkErr)
					} else {
						multi.add(&WalkError{Path: path, Err: err})
					}
				} else {
					return err
				}
			}
		}

		if err := w.dispatchHandlers(fc, fm, AfterChildren, multi); err != nil {
			if errors.Is(err, ErrSkipField) {
				continue
			}
			if multi == nil && !errors.Is(err, ErrSkipChildren) {
				return err
			}
		}

		if err := w.dispatchGlobal(fc, AfterChildren, multi); err != nil {
			if errors.Is(err, ErrSkipField) {
				continue
			}
			if multi == nil && !errors.Is(err, ErrSkipChildren) {
				return err
			}
		}
	}

	if multi != nil && len(multi.Errors) > 0 {
		return multi
	}
	return nil
}

func (w *Walker) checkContext() error {
	select {
	case <-w.opts.ctx.Done():
		return w.opts.ctx.Err()
	default:
		return nil
	}
}

func (w *Walker) handleError(err error, fc FieldContext, tagKey string, multi *MultiError) error {
	if errors.Is(err, ErrSkipField) || errors.Is(err, ErrSkipChildren) {
		return err
	}
	if w.opts.errorMode == SkipOnError {
		return nil
	}
	walkErr := &WalkError{
		Path:   fc.Path,
		TagKey: tagKey,
		Field:  fc.Field.Name,
		Err:    err,
	}
	if multi != nil {
		multi.add(walkErr)
		return nil
	}
	return walkErr
}

func (w *Walker) dispatchHandlers(fc FieldContext, fm fieldMeta, phase Phase, multi *MultiError) error {
	for _, key := range fm.HandlerKeys {
		tag, ok := fm.TagMap[key]
		if !ok || tag.IsIgnored() {
			continue
		}

		entries := w.handlers.entriesForPhase(key, phase)
		for _, entry := range entries {
			fc.Tag = tag
			if entry.config.filter != nil && !entry.config.filter(fc) {
				continue
			}

			if err := w.chain(entry.fn)(fc); err != nil {
				if herr := w.handleError(err, fc, key, multi); herr != nil {
					return herr
				}
			}
		}
	}
	return nil
}

func (w *Walker) dispatchGlobal(fc FieldContext, phase Phase, multi *MultiError) error {
	entries := w.handlers.globalForPhase(phase)
	for _, entry := range entries {
		if entry.config.filter != nil && !entry.config.filter(fc) {
			continue
		}

		if err := w.chain(entry.fn)(fc); err != nil {
			if herr := w.handleError(err, fc, "", multi); herr != nil {
				return herr
			}
		}
	}
	return nil
}

func (w *Walker) recurseField(fieldVal reflect.Value, fm fieldMeta, path string, depth int, visited map[uintptr]bool) error {
	val := fieldVal
	if fm.IsPtr && w.opts.followPointers {
		if val.IsNil() {
			return nil
		}
		val = val.Elem()

		ptr := fieldVal.Pointer()
		if visited[ptr] {
			return nil
		}
		visited[ptr] = true
		defer delete(visited, ptr)
	}

	if val.Kind() == reflect.Struct && ((fm.IsStruct && !fm.IsPtr) || (fm.IsPtr && w.opts.followPointers)) {
		return w.walkStruct(val, path, depth+1, visited)
	}

	if fm.IsSlice && w.opts.sliceElements {
		return w.walkSlice(val, path, depth, visited)
	}

	if fm.IsMap && w.opts.mapEntries {
		return w.walkMap(val, path, depth, visited)
	}

	return nil
}

func (w *Walker) walkSlice(v reflect.Value, basePath string, depth int, visited map[uintptr]bool) error {
	for i := range v.Len() {
		if err := w.walkSliceElem(v.Index(i), fmt.Sprintf("%s[%d]", basePath, i), depth, visited); err != nil {
			return err
		}
	}
	return nil
}

// walkSliceElem processes a single slice element, dereferencing pointers and
// recursing into structs. Extracted from walkSlice to avoid defer-in-loop (G35).
func (w *Walker) walkSliceElem(elem reflect.Value, elemPath string, depth int, visited map[uintptr]bool) error {
	if elem.Kind() == reflect.Pointer {
		if elem.IsNil() {
			return nil
		}
		if visited != nil {
			ptr := elem.Pointer()
			if visited[ptr] {
				return nil
			}
			visited[ptr] = true
			defer delete(visited, ptr)
		}
		elem = elem.Elem()
	}

	if elem.Kind() == reflect.Struct {
		return w.walkStruct(elem, elemPath, depth+1, visited)
	}
	return nil
}

func (w *Walker) walkMap(v reflect.Value, basePath string, depth int, visited map[uintptr]bool) error {
	if v.IsNil() {
		return nil
	}
	iter := v.MapRange()
	for iter.Next() {
		keyStr := fmt.Sprintf("%v", iter.Key().Interface())
		elemPath := fmt.Sprintf("%s[%s]", basePath, keyStr)
		if err := w.walkMapElem(iter.Value(), elemPath, depth, visited); err != nil {
			return err
		}
	}
	return nil
}

// walkMapElem processes a single map value, dereferencing pointers and
// recursing into structs. Extracted from walkMap to avoid defer-in-loop (G35).
func (w *Walker) walkMapElem(elem reflect.Value, elemPath string, depth int, visited map[uintptr]bool) error {
	if elem.Kind() == reflect.Pointer {
		if elem.IsNil() {
			return nil
		}
		if visited != nil {
			ptr := elem.Pointer()
			if visited[ptr] {
				return nil
			}
			visited[ptr] = true
			defer delete(visited, ptr)
		}
		elem = elem.Elem()
	}

	if elem.Kind() == reflect.Struct {
		return w.walkStruct(elem, elemPath, depth+1, visited)
	}
	return nil
}

func joinPath(base, name string) string {
	if base == "" {
		return name
	}
	return base + "." + name
}
