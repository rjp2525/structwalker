package structwalker

// HandlerFunc is called for every field a handler applies to.
type HandlerFunc func(ctx FieldContext) error

// Phase determines when a handler runs relative to a field's children.
type Phase int

const (
	// BeforeChildren runs the handler before the walker descends into the field.
	BeforeChildren Phase = iota
	// AfterChildren runs the handler once the field's children have been walked.
	AfterChildren
)

const defaultPriority = 100

// HandlerOption configures a single handler registration.
type HandlerOption func(*handler)

// WithPriority sets when the handler runs relative to the other handlers on a
// field. Lower runs first and the default is 100.
func WithPriority(p int) HandlerOption {
	return func(h *handler) { h.priority = p }
}

// WithFilter restricts the handler to fields where fn returns true.
func WithFilter(fn func(FieldContext) bool) HandlerOption {
	return func(h *handler) { h.filter = fn }
}

// WithPhase sets when the handler runs relative to the field's children.
func WithPhase(phase Phase) HandlerOption {
	return func(h *handler) { h.phase = phase }
}

type handler struct {
	key      string
	global   bool
	fn       HandlerFunc
	priority int
	phase    Phase
	filter   func(FieldContext) bool
}

func newHandler(key string, global bool, fn HandlerFunc, opts []HandlerOption) handler {
	h := handler{key: key, global: global, fn: fn, priority: defaultPriority}
	for _, opt := range opts {
		opt(&h)
	}
	return h
}
