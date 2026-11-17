package structwalker

import (
	"reflect"
	"testing"
)

func TestTypeCache_GetAndCacheHit(t *testing.T) {
	cache := newTypeCache()

	type Sample struct {
		Name  string `json:"name" validate:"required"`
		Email string `json:"email"`
	}

	st := reflect.TypeFor[Sample]()
	keys := []string{"json", "validate"}

	meta1 := cache.get(st, keys, nil)
	if meta1 == nil {
		t.Fatal("expected non-nil meta")
	}
	if len(meta1.Fields) != 2 {
		t.Errorf("len(Fields) = %d, want 2", len(meta1.Fields))
	}

	// Second call should return cached value
	meta2 := cache.get(st, keys, nil)
	if meta1 != meta2 {
		t.Error("expected same pointer on cache hit")
	}
}

func TestTypeCache_FieldMeta(t *testing.T) {
	cache := newTypeCache()

	type Inner struct {
		Value int `json:"value"`
	}

	type Outer struct {
		Name   string            `json:"name"`
		Inner  Inner             `json:"inner"`
		Ptr    *Inner            `json:"ptr"`
		Items  []Inner           `json:"items"`
		Lookup map[string]string `json:"lookup"`
	}

	st := reflect.TypeFor[Outer]()
	meta := cache.get(st, []string{"json"}, nil)

	if len(meta.Fields) != 5 {
		t.Fatalf("len(Fields) = %d, want 5", len(meta.Fields))
	}

	// Name: string, not struct/ptr/slice/map
	name := meta.Fields[0]
	if name.IsStruct || name.IsPtr || name.IsSlice || name.IsMap {
		t.Errorf("Name field: unexpected flags: struct=%v ptr=%v slice=%v map=%v",
			name.IsStruct, name.IsPtr, name.IsSlice, name.IsMap)
	}

	// Inner: struct
	inner := meta.Fields[1]
	if !inner.IsStruct {
		t.Error("Inner should be IsStruct")
	}

	// Ptr: pointer to struct
	ptr := meta.Fields[2]
	if !ptr.IsPtr || !ptr.IsStruct {
		t.Errorf("Ptr: IsPtr=%v IsStruct=%v, want both true", ptr.IsPtr, ptr.IsStruct)
	}

	// Items: slice
	items := meta.Fields[3]
	if !items.IsSlice {
		t.Error("Items should be IsSlice")
	}

	// Lookup: map
	lookup := meta.Fields[4]
	if !lookup.IsMap {
		t.Error("Lookup should be IsMap")
	}
}

func TestTypeCache_HandlerKeys(t *testing.T) {
	cache := newTypeCache()

	type Sample struct {
		Name  string `json:"name" validate:"required"`
		Email string `validate:"email"`
		Age   int    `json:"age"`
	}

	st := reflect.TypeFor[Sample]()
	meta := cache.get(st, []string{"json", "validate"}, nil)

	// Name has both json and validate
	if len(meta.Fields[0].HandlerKeys) != 2 {
		t.Errorf("Name HandlerKeys = %v, want 2 keys", meta.Fields[0].HandlerKeys)
	}

	// Email has only validate
	if len(meta.Fields[1].HandlerKeys) != 1 || meta.Fields[1].HandlerKeys[0] != "validate" {
		t.Errorf("Email HandlerKeys = %v, want [validate]", meta.Fields[1].HandlerKeys)
	}

	// Age has only json
	if len(meta.Fields[2].HandlerKeys) != 1 || meta.Fields[2].HandlerKeys[0] != "json" {
		t.Errorf("Age HandlerKeys = %v, want [json]", meta.Fields[2].HandlerKeys)
	}
}

func TestTypeCache_Clear(t *testing.T) {
	cache := newTypeCache()

	type Sample struct {
		Name string `json:"name"`
	}

	st := reflect.TypeFor[Sample]()
	cache.get(st, []string{"json"}, nil)
	cache.clear()

	// After clear, should rebuild
	meta := cache.get(st, []string{}, nil)
	if meta == nil {
		t.Fatal("expected non-nil meta after clear")
	}
	// With no handler keys, no fields should have HandlerKeys
	if len(meta.Fields[0].HandlerKeys) != 0 {
		t.Errorf("expected no HandlerKeys after clear with empty keys")
	}
}

func TestTypeCache_Invalidate(t *testing.T) {
	cache := newTypeCache()

	type Sample struct {
		Name string `json:"name"`
	}

	st := reflect.TypeFor[Sample]()
	meta1 := cache.get(st, []string{"json"}, nil)
	cache.invalidate(st)
	meta2 := cache.get(st, []string{"json"}, nil)

	if meta1 == meta2 {
		t.Error("expected different pointer after invalidate")
	}
}
