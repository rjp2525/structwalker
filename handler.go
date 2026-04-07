package structwalker

import "sort"

// HandlerFunc is the callback signature for tag processors.
type HandlerFunc func(ctx FieldContext) error

// Phase determines when a handler runs relative to child field processing.
type Phase int

const (
	// BeforeChildren runs the handler before recursing into nested struct fields.
	BeforeChildren Phase = iota
	// AfterChildren runs the handler after nested struct fields have been processed.
	AfterChildren
)

// HandlerOption configures individual handler behavior.
type HandlerOption func(*handlerConfig)

// WithPriority sets execution order (lower runs first). Default: 100.
func WithPriority(p int) HandlerOption {
	return func(c *handlerConfig) {
		c.priority = p
	}
}

// WithFilter restricts the handler to fields where fn returns true.
func WithFilter(fn func(FieldContext) bool) HandlerOption {
	return func(c *handlerConfig) {
		c.filter = fn
	}
}

// WithPhase sets when the handler runs relative to child processing.
func WithPhase(phase Phase) HandlerOption {
	return func(c *handlerConfig) {
		c.phase = phase
	}
}

type handlerConfig struct {
	priority int
	phase    Phase
	filter   func(FieldContext) bool
}

func buildConfig(opts []HandlerOption) handlerConfig {
	cfg := handlerConfig{priority: 100}
	for _, opt := range opts {
		opt(&cfg)
	}
	return cfg
}

type handlerEntry struct {
	fn     HandlerFunc
	config handlerConfig
}

type handlerRegistry struct {
	tagged map[string][]handlerEntry
	global []handlerEntry
	keys_  []string // cached key list, invalidated on add
}

func newHandlerRegistry() *handlerRegistry {
	return &handlerRegistry{
		tagged: make(map[string][]handlerEntry),
	}
}

func (r *handlerRegistry) add(tagKey string, fn HandlerFunc, opts ...HandlerOption) {
	r.tagged[tagKey] = append(r.tagged[tagKey], handlerEntry{fn: fn, config: buildConfig(opts)})
	r.keys_ = nil
}

func (r *handlerRegistry) addGlobal(fn HandlerFunc, opts ...HandlerOption) {
	r.global = append(r.global, handlerEntry{fn: fn, config: buildConfig(opts)})
}

func (r *handlerRegistry) keys() []string {
	if r.keys_ != nil {
		return r.keys_
	}
	r.keys_ = make([]string, 0, len(r.tagged))
	for k := range r.tagged {
		r.keys_ = append(r.keys_, k)
	}
	return r.keys_
}

func filterByPhase(entries []handlerEntry, phase Phase) []handlerEntry {
	var filtered []handlerEntry
	for _, e := range entries {
		if e.config.phase == phase {
			filtered = append(filtered, e)
		}
	}
	sort.Slice(filtered, func(i, j int) bool {
		return filtered[i].config.priority < filtered[j].config.priority
	})
	return filtered
}

func (r *handlerRegistry) entriesForPhase(tagKey string, phase Phase) []handlerEntry {
	return filterByPhase(r.tagged[tagKey], phase)
}

func (r *handlerRegistry) globalForPhase(phase Phase) []handlerEntry {
	return filterByPhase(r.global, phase)
}
