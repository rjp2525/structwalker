package structwalker

import (
	"testing"
)

func TestHandlerRegistry_Add(t *testing.T) {
	r := newHandlerRegistry()
	r.add("json", func(ctx FieldContext) error { return nil })
	r.add("json", func(ctx FieldContext) error { return nil })
	r.add("validate", func(ctx FieldContext) error { return nil })

	if len(r.tagged["json"]) != 2 {
		t.Errorf("json handlers = %d, want 2", len(r.tagged["json"]))
	}
	if len(r.tagged["validate"]) != 1 {
		t.Errorf("validate handlers = %d, want 1", len(r.tagged["validate"]))
	}
}

func TestHandlerRegistry_Keys(t *testing.T) {
	r := newHandlerRegistry()
	r.add("json", func(ctx FieldContext) error { return nil })
	r.add("validate", func(ctx FieldContext) error { return nil })

	keys := r.keys()
	if len(keys) != 2 {
		t.Errorf("keys = %v, want 2 entries", keys)
	}
}

func TestHandlerRegistry_EntriesForPhase(t *testing.T) {
	r := newHandlerRegistry()
	r.add("json", func(ctx FieldContext) error { return nil }, WithPriority(200))
	r.add("json", func(ctx FieldContext) error { return nil }, WithPriority(50))
	r.add("json", func(ctx FieldContext) error { return nil }, WithPhase(AfterChildren))

	before := r.entriesForPhase("json", BeforeChildren)
	if len(before) != 2 {
		t.Fatalf("BeforeChildren entries = %d, want 2", len(before))
	}
	// Should be sorted by priority
	if before[0].config.priority != 50 || before[1].config.priority != 200 {
		t.Errorf("priorities = [%d, %d], want [50, 200]",
			before[0].config.priority, before[1].config.priority)
	}

	after := r.entriesForPhase("json", AfterChildren)
	if len(after) != 1 {
		t.Errorf("AfterChildren entries = %d, want 1", len(after))
	}
}

func TestHandlerRegistry_GlobalForPhase(t *testing.T) {
	r := newHandlerRegistry()
	r.addGlobal(func(ctx FieldContext) error { return nil }, WithPriority(100))
	r.addGlobal(func(ctx FieldContext) error { return nil }, WithPriority(10), WithPhase(AfterChildren))

	before := r.globalForPhase(BeforeChildren)
	if len(before) != 1 {
		t.Errorf("BeforeChildren global = %d, want 1", len(before))
	}

	after := r.globalForPhase(AfterChildren)
	if len(after) != 1 {
		t.Errorf("AfterChildren global = %d, want 1", len(after))
	}
}

func TestHandlerConfig_Defaults(t *testing.T) {
	r := newHandlerRegistry()
	r.add("test", func(ctx FieldContext) error { return nil })

	entries := r.entriesForPhase("test", BeforeChildren)
	if len(entries) != 1 {
		t.Fatal("expected 1 entry")
	}
	if entries[0].config.priority != 100 {
		t.Errorf("default priority = %d, want 100", entries[0].config.priority)
	}
	if entries[0].config.phase != BeforeChildren {
		t.Errorf("default phase = %d, want BeforeChildren", entries[0].config.phase)
	}
}
