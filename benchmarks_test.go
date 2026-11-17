package structwalker

import (
	"reflect"
	"testing"
)

type BenchStruct struct {
	Name    string `json:"name" validate:"required"`
	Email   string `json:"email" validate:"email"`
	Age     int    `json:"age" validate:"min=0,max=150"`
	Active  bool   `json:"active"`
	Country string `json:"country"`
}

type BenchNested struct {
	User    BenchStruct `json:"user"`
	Admin   BenchStruct `json:"admin"`
	Comment string      `json:"comment"`
}

type BenchDeep struct {
	Level1 struct {
		Level2 struct {
			Level3 struct {
				Value string `json:"value"`
			} `json:"level3"`
		} `json:"level2"`
	} `json:"level1"`
}

func BenchmarkWalk_Flat(b *testing.B) {
	w := New()
	w.Handle("json", func(ctx FieldContext) error { return nil })

	s := &BenchStruct{Name: "Alice", Email: "alice@example.com", Age: 30, Active: true, Country: "US"}
	b.ResetTimer()
	for b.Loop() {
		_ = w.Walk(s)
	}
}

func BenchmarkWalk_Nested(b *testing.B) {
	w := New()
	w.Handle("json", func(ctx FieldContext) error { return nil })

	s := &BenchNested{
		User:    BenchStruct{Name: "Alice", Email: "alice@example.com", Age: 30},
		Admin:   BenchStruct{Name: "Bob", Email: "bob@example.com", Age: 40},
		Comment: "test",
	}
	b.ResetTimer()
	for b.Loop() {
		_ = w.Walk(s)
	}
}

func BenchmarkWalk_Deep(b *testing.B) {
	w := New()
	w.Handle("json", func(ctx FieldContext) error { return nil })

	s := &BenchDeep{}
	b.ResetTimer()
	for b.Loop() {
		_ = w.Walk(s)
	}
}

func BenchmarkWalk_MultipleHandlers(b *testing.B) {
	w := New()
	w.Handle("json", func(ctx FieldContext) error { return nil })
	w.Handle("validate", func(ctx FieldContext) error { return nil })

	s := &BenchStruct{Name: "Alice", Email: "alice@example.com", Age: 30}
	b.ResetTimer()
	for b.Loop() {
		_ = w.Walk(s)
	}
}

func BenchmarkWalk_WithMiddleware(b *testing.B) {
	w := New()
	w.Use(func(ctx FieldContext, next HandlerFunc) error { return next(ctx) })
	w.Handle("json", func(ctx FieldContext) error { return nil })

	s := &BenchStruct{Name: "Alice", Email: "alice@example.com", Age: 30}
	b.ResetTimer()
	for b.Loop() {
		_ = w.Walk(s)
	}
}

func BenchmarkWalk_HandleAll(b *testing.B) {
	w := New()
	w.HandleAll(func(ctx FieldContext) error { return nil })

	s := &BenchStruct{Name: "Alice", Email: "alice@example.com", Age: 30}
	b.ResetTimer()
	for b.Loop() {
		_ = w.Walk(s)
	}
}

func BenchmarkParseTag(b *testing.B) {
	for b.Loop() {
		ParseTag("validate", "email,required,max=255,format=rfc5322")
	}
}

func BenchmarkParseAllTags(b *testing.B) {
	tag := reflect.StructTag(`json:"name,omitempty" validate:"required,max=100" db:"user_name"`)
	b.ResetTimer()
	for b.Loop() {
		ParseAllTags(tag)
	}
}

// BenchmarkRawReflection provides a baseline for comparison.
func BenchmarkRawReflection(b *testing.B) {
	s := &BenchStruct{Name: "Alice", Email: "alice@example.com", Age: 30, Active: true, Country: "US"}
	t := reflect.TypeOf(*s)
	b.ResetTimer()
	for b.Loop() {
		v := reflect.ValueOf(s).Elem()
		for i := range t.NumField() {
			_ = t.Field(i)
			_ = v.Field(i)
		}
	}
}

func BenchmarkPlan_Hit(b *testing.B) {
	w := New()
	w.Handle("json", func(FieldContext) error { return nil })
	r := w.registry.Load()
	t := reflect.TypeFor[BenchStruct]()
	r.plan(t, &w.opts)

	for b.Loop() {
		r.plan(t, &w.opts)
	}
}
