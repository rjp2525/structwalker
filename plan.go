package structwalker

import (
	"cmp"
	"reflect"
	"slices"
	"sync"
)

// registry is an immutable snapshot of a walker's handlers and middleware.
// Registering anything swaps in a new snapshot, so a walk that is already
// running keeps the one it started with.
type registry struct {
	handlers   []handler
	middleware []Middleware
	chained    []HandlerFunc
	plans      sync.Map
}

func newRegistry(handlers []handler, middleware []Middleware) *registry {
	r := &registry{
		handlers:   handlers,
		middleware: middleware,
		chained:    make([]HandlerFunc, len(handlers)),
	}
	for i, h := range handlers {
		r.chained[i] = chain(h.fn, middleware)
	}
	return r
}

func (r *registry) withHandler(h handler) *registry {
	return newRegistry(append(slices.Clip(r.handlers), h), r.middleware)
}

func (r *registry) withMiddleware(mw []Middleware) *registry {
	return newRegistry(r.handlers, append(slices.Clip(r.middleware), mw...))
}

// typePlan is everything the walker needs to know about a struct type, worked
// out once: which fields to visit and which handlers run on each, in order.
type typePlan struct {
	fields []fieldPlan
}

type fieldPlan struct {
	field    reflect.StructField
	index    int
	tags     []Tag
	before   []step
	after    []step
	isStruct bool
	isPtr    bool
	isSlice  bool
	isMap    bool
}

type step struct {
	run    HandlerFunc
	filter func(FieldContext) bool
	tag    Tag
}

func (r *registry) plan(t reflect.Type, opts *walkerOptions) *typePlan {
	if p, ok := r.plans.Load(t); ok {
		return p.(*typePlan)
	}
	p, _ := r.plans.LoadOrStore(t, r.build(t, opts))
	return p.(*typePlan)
}

func (r *registry) build(t reflect.Type, opts *walkerOptions) *typePlan {
	plan := &typePlan{fields: make([]fieldPlan, 0, t.NumField())}

	for i := range t.NumField() {
		sf := t.Field(i)
		if !sf.IsExported() && !opts.unexported {
			continue
		}

		elem := sf.Type
		isPtr := elem.Kind() == reflect.Pointer
		if isPtr {
			elem = elem.Elem()
		}

		fp := fieldPlan{
			field:    sf,
			index:    i,
			tags:     parseStructTag(sf.Tag, opts.tagParser),
			isStruct: elem.Kind() == reflect.Struct,
			isPtr:    isPtr,
			isSlice:  sf.Type.Kind() == reflect.Slice || sf.Type.Kind() == reflect.Array,
			isMap:    sf.Type.Kind() == reflect.Map,
		}
		fp.before, fp.after = r.steps(fp.tags)
		plan.fields = append(plan.fields, fp)
	}

	return plan
}

// steps works out which handlers run on a field with the given tags. They are
// ordered by priority. Ties go to tag handlers in the order the tags are
// written on the field, then to HandleAll handlers, and registration order
// settles whatever is left.
func (r *registry) steps(tags []Tag) (before, after []step) {
	type candidate struct {
		step
		phase    Phase
		priority int
		position int
	}

	var candidates []candidate
	for i, h := range r.handlers {
		c := candidate{
			step:     step{run: r.chained[i], filter: h.filter},
			phase:    h.phase,
			priority: h.priority,
			position: len(tags),
		}
		if !h.global {
			pos := slices.IndexFunc(tags, func(t Tag) bool { return t.Key == h.key })
			if pos < 0 || tags[pos].IsIgnored() {
				continue
			}
			c.tag, c.position = tags[pos], pos
		}
		candidates = append(candidates, c)
	}

	slices.SortStableFunc(candidates, func(a, b candidate) int {
		return cmp.Or(cmp.Compare(a.priority, b.priority), cmp.Compare(a.position, b.position))
	})

	for _, c := range candidates {
		if c.phase == AfterChildren {
			after = append(after, c.step)
		} else {
			before = append(before, c.step)
		}
	}
	return before, after
}
