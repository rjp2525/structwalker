package structwalker

import (
	"context"
	"errors"
	"testing"
)

func TestWalk_BasicStruct(t *testing.T) {
	type User struct {
		Name  string `json:"name"`
		Email string `json:"email"`
	}

	var visited []string
	w := New()
	w.Handle("json", func(ctx FieldContext) error {
		visited = append(visited, ctx.Tag.Name)
		return nil
	})

	err := w.Walk(&User{Name: "Alice", Email: "alice@example.com"})
	if err != nil {
		t.Fatal(err)
	}
	if len(visited) != 2 {
		t.Fatalf("visited %d fields, want 2", len(visited))
	}
	if visited[0] != "name" || visited[1] != "email" {
		t.Errorf("visited = %v, want [name email]", visited)
	}
}

func TestWalk_NestedStruct(t *testing.T) {
	type Address struct {
		Street string `json:"street"`
		City   string `json:"city"`
	}
	type User struct {
		Name    string  `json:"name"`
		Address Address `json:"address"`
	}

	var paths []string
	w := New()
	w.Handle("json", func(ctx FieldContext) error {
		paths = append(paths, ctx.Path)
		return nil
	})

	err := w.Walk(&User{})
	if err != nil {
		t.Fatal(err)
	}
	expected := []string{"Name", "Address", "Address.Street", "Address.City"}
	if len(paths) != len(expected) {
		t.Fatalf("paths = %v, want %v", paths, expected)
	}
	for i, p := range expected {
		if paths[i] != p {
			t.Errorf("paths[%d] = %q, want %q", i, paths[i], p)
		}
	}
}

func TestWalk_PointerToStruct(t *testing.T) {
	type Inner struct {
		Value string `tag:"val"`
	}
	type Outer struct {
		Inner *Inner `tag:"inner"`
	}

	var paths []string
	w := New()
	w.Handle("tag", func(ctx FieldContext) error {
		paths = append(paths, ctx.Path)
		return nil
	})

	err := w.Walk(&Outer{Inner: &Inner{Value: "hello"}})
	if err != nil {
		t.Fatal(err)
	}
	if len(paths) != 2 {
		t.Fatalf("paths = %v, want 2 entries", paths)
	}
}

func TestWalk_NilPointer(t *testing.T) {
	type Inner struct {
		Value string `tag:"val"`
	}
	type Outer struct {
		Inner *Inner `tag:"inner"`
	}

	var paths []string
	w := New()
	w.Handle("tag", func(ctx FieldContext) error {
		paths = append(paths, ctx.Path)
		return nil
	})

	err := w.Walk(&Outer{Inner: nil})
	if err != nil {
		t.Fatal(err)
	}
	// Should visit Outer.Inner but not recurse into nil pointer
	if len(paths) != 1 {
		t.Fatalf("paths = %v, want 1 entry", paths)
	}
}

func TestWalk_SkipField(t *testing.T) {
	type S struct {
		A string `tag:"a"`
		B string `tag:"b"`
		C string `tag:"c"`
	}

	var visited []string
	w := New()
	w.Handle("tag", func(ctx FieldContext) error {
		if ctx.Tag.Name == "b" {
			return SkipField
		}
		visited = append(visited, ctx.Tag.Name)
		return nil
	})

	err := w.Walk(&S{})
	if err != nil {
		t.Fatal(err)
	}
	if len(visited) != 2 || visited[0] != "a" || visited[1] != "c" {
		t.Errorf("visited = %v, want [a c]", visited)
	}
}

func TestWalk_SkipChildren(t *testing.T) {
	type Inner struct {
		Value string `tag:"value"`
	}
	type Outer struct {
		Inner Inner `tag:"inner"`
	}

	var paths []string
	w := New()
	w.Handle("tag", func(ctx FieldContext) error {
		paths = append(paths, ctx.Path)
		if ctx.Tag.Name == "inner" {
			return SkipChildren
		}
		return nil
	})

	err := w.Walk(&Outer{})
	if err != nil {
		t.Fatal(err)
	}
	// Should visit Inner but not recurse into it
	if len(paths) != 1 || paths[0] != "Inner" {
		t.Fatalf("paths = %v, want [Inner]", paths)
	}
}

func TestWalk_IgnoredTag(t *testing.T) {
	type S struct {
		A string `json:"a"`
		B string `json:"-"`
		C string `json:"c"`
	}

	var visited []string
	w := New()
	w.Handle("json", func(ctx FieldContext) error {
		visited = append(visited, ctx.Tag.Name)
		return nil
	})

	err := w.Walk(&S{})
	if err != nil {
		t.Fatal(err)
	}
	if len(visited) != 2 || visited[0] != "a" || visited[1] != "c" {
		t.Errorf("visited = %v, want [a c]", visited)
	}
}

func TestWalk_HandleAll(t *testing.T) {
	type S struct {
		A string
		B int
		C bool
	}

	count := 0
	w := New()
	w.HandleAll(func(ctx FieldContext) error {
		count++
		return nil
	})

	err := w.Walk(&S{})
	if err != nil {
		t.Fatal(err)
	}
	if count != 3 {
		t.Errorf("count = %d, want 3", count)
	}
}

func TestWalk_StopOnError(t *testing.T) {
	type S struct {
		A string `tag:"a"`
		B string `tag:"b"`
	}

	w := New()
	w.Handle("tag", func(ctx FieldContext) error {
		return errors.New("fail")
	})

	err := w.Walk(&S{})
	if err == nil {
		t.Fatal("expected error")
	}
	var walkErr *WalkError
	if !errors.As(err, &walkErr) {
		t.Fatalf("expected WalkError, got %T", err)
	}
	if walkErr.Path != "A" {
		t.Errorf("WalkError.Path = %q, want %q", walkErr.Path, "A")
	}
}

func TestWalk_CollectErrors(t *testing.T) {
	type S struct {
		A string `tag:"a"`
		B string `tag:"b"`
		C string `tag:"c"`
	}

	w := New(WithErrorMode(CollectErrors))
	w.Handle("tag", func(ctx FieldContext) error {
		return errors.New("fail")
	})

	err := w.Walk(&S{})
	if err == nil {
		t.Fatal("expected error")
	}
	var multi *MultiError
	if !errors.As(err, &multi) {
		t.Fatalf("expected MultiError, got %T", err)
	}
	if len(multi.Errors) != 3 {
		t.Errorf("len(Errors) = %d, want 3", len(multi.Errors))
	}
}

func TestWalk_SkipOnError(t *testing.T) {
	type S struct {
		A string `tag:"a"`
		B string `tag:"b"`
	}

	visited := 0
	w := New(WithErrorMode(SkipOnError))
	w.Handle("tag", func(ctx FieldContext) error {
		visited++
		return errors.New("fail")
	})

	err := w.Walk(&S{})
	if err != nil {
		t.Fatalf("expected nil error with SkipOnError, got %v", err)
	}
	if visited != 2 {
		t.Errorf("visited = %d, want 2", visited)
	}
}

func TestWalk_MaxDepth(t *testing.T) {
	type Recursive struct {
		Child *Recursive `tag:"child"`
	}

	w := New(WithMaxDepth(3))
	w.Handle("tag", func(ctx FieldContext) error {
		return nil
	})

	s := &Recursive{Child: &Recursive{Child: &Recursive{Child: &Recursive{Child: &Recursive{}}}}}
	err := w.Walk(s)
	if err == nil {
		t.Fatal("expected max depth error")
	}
}

func TestWalk_ContextCancellation(t *testing.T) {
	type S struct {
		A string `tag:"a"`
		B string `tag:"b"`
	}

	ctx, cancel := context.WithCancel(context.Background())
	w := New(WithContext(ctx))
	w.Handle("tag", func(fc FieldContext) error {
		cancel()
		return nil
	})

	err := w.Walk(&S{})
	if !errors.Is(err, context.Canceled) {
		t.Errorf("expected context.Canceled, got %v", err)
	}
}

func TestWalk_Priority(t *testing.T) {
	type S struct {
		Name string `tag:"name"`
	}

	var order []int
	w := New()
	w.Handle("tag", func(ctx FieldContext) error {
		order = append(order, 2)
		return nil
	}, WithPriority(200))
	w.Handle("tag", func(ctx FieldContext) error {
		order = append(order, 1)
		return nil
	}, WithPriority(50))

	err := w.Walk(&S{})
	if err != nil {
		t.Fatal(err)
	}
	if len(order) != 2 || order[0] != 1 || order[1] != 2 {
		t.Errorf("order = %v, want [1 2]", order)
	}
}

func TestWalk_Filter(t *testing.T) {
	type S struct {
		Name  string `tag:"name"`
		Email string `tag:"email"`
	}

	var visited []string
	w := New()
	w.Handle("tag", func(ctx FieldContext) error {
		visited = append(visited, ctx.Tag.Name)
		return nil
	}, WithFilter(func(ctx FieldContext) bool {
		return ctx.Tag.Name == "email"
	}))

	err := w.Walk(&S{})
	if err != nil {
		t.Fatal(err)
	}
	if len(visited) != 1 || visited[0] != "email" {
		t.Errorf("visited = %v, want [email]", visited)
	}
}

func TestWalk_AfterChildren(t *testing.T) {
	type Inner struct {
		Value string `tag:"value"`
	}
	type Outer struct {
		Inner Inner `tag:"inner"`
	}

	var order []string
	w := New()
	w.Handle("tag", func(ctx FieldContext) error {
		order = append(order, "before:"+ctx.Path)
		return nil
	})
	w.Handle("tag", func(ctx FieldContext) error {
		order = append(order, "after:"+ctx.Path)
		return nil
	}, WithPhase(AfterChildren))

	err := w.Walk(&Outer{})
	if err != nil {
		t.Fatal(err)
	}
	expected := []string{"before:Inner", "before:Inner.Value", "after:Inner.Value", "after:Inner"}
	if len(order) != len(expected) {
		t.Fatalf("order = %v, want %v", order, expected)
	}
	for i, v := range expected {
		if order[i] != v {
			t.Errorf("order[%d] = %q, want %q", i, order[i], v)
		}
	}
}

func TestWalk_SliceElements(t *testing.T) {
	type Item struct {
		Name string `tag:"name"`
	}
	type Container struct {
		Items []Item `tag:"items"`
	}

	var paths []string
	w := New(WithSliceElements(true))
	w.Handle("tag", func(ctx FieldContext) error {
		paths = append(paths, ctx.Path)
		return nil
	})

	err := w.Walk(&Container{Items: []Item{{Name: "a"}, {Name: "b"}}})
	if err != nil {
		t.Fatal(err)
	}
	expected := []string{"Items", "Items[0].Name", "Items[1].Name"}
	if len(paths) != len(expected) {
		t.Fatalf("paths = %v, want %v", paths, expected)
	}
	for i, p := range expected {
		if paths[i] != p {
			t.Errorf("paths[%d] = %q, want %q", i, paths[i], p)
		}
	}
}

func TestWalk_MapEntries(t *testing.T) {
	type Value struct {
		Data string `tag:"data"`
	}
	type Container struct {
		M map[string]Value `tag:"m"`
	}

	var paths []string
	w := New(WithMapEntries(true))
	w.Handle("tag", func(ctx FieldContext) error {
		paths = append(paths, ctx.Path)
		return nil
	})

	err := w.Walk(&Container{M: map[string]Value{"key1": {Data: "hello"}}})
	if err != nil {
		t.Fatal(err)
	}
	// Should visit M and M[key1].Data
	if len(paths) < 2 {
		t.Fatalf("paths = %v, want at least 2", paths)
	}
}

func TestWalk_SetValue(t *testing.T) {
	type S struct {
		Name string `tag:"name"`
		Age  int    `tag:"age"`
	}

	w := New()
	w.Handle("tag", func(ctx FieldContext) error {
		switch ctx.Tag.Name {
		case "name":
			return ctx.SetValue("modified")
		case "age":
			return ctx.SetValue(42)
		}
		return nil
	})

	s := &S{Name: "original", Age: 0}
	err := w.Walk(s)
	if err != nil {
		t.Fatal(err)
	}
	if s.Name != "modified" {
		t.Errorf("Name = %q, want %q", s.Name, "modified")
	}
	if s.Age != 42 {
		t.Errorf("Age = %d, want 42", s.Age)
	}
}

func TestWalk_SetValueFromString(t *testing.T) {
	type S struct {
		Count   int     `tag:"count"`
		Rate    float64 `tag:"rate"`
		Enabled bool    `tag:"enabled"`
	}

	w := New()
	w.Handle("tag", func(ctx FieldContext) error {
		switch ctx.Tag.Name {
		case "count":
			return ctx.SetValue("42")
		case "rate":
			return ctx.SetValue("3.14")
		case "enabled":
			return ctx.SetValue("true")
		}
		return nil
	})

	s := &S{}
	err := w.Walk(s)
	if err != nil {
		t.Fatal(err)
	}
	if s.Count != 42 {
		t.Errorf("Count = %d, want 42", s.Count)
	}
	if s.Rate != 3.14 {
		t.Errorf("Rate = %f, want 3.14", s.Rate)
	}
	if !s.Enabled {
		t.Error("Enabled = false, want true")
	}
}

func TestWalk_NonPointerStruct(t *testing.T) {
	type S struct {
		Name string `tag:"name"`
	}

	var visited []string
	w := New()
	w.Handle("tag", func(ctx FieldContext) error {
		visited = append(visited, ctx.Tag.Name)
		return nil
	})

	// Walk with value (not pointer) should still work for reading
	err := w.Walk(S{Name: "test"})
	if err != nil {
		t.Fatal(err)
	}
	if len(visited) != 1 {
		t.Errorf("visited = %v, want 1 entry", visited)
	}
}

func TestWalk_NonStruct(t *testing.T) {
	w := New()
	err := w.Walk("not a struct")
	if err == nil {
		t.Fatal("expected error for non-struct")
	}
}

func TestWalk_NilInput(t *testing.T) {
	w := New()
	err := w.Walk((*struct{})(nil))
	if err != nil {
		t.Fatalf("expected nil error for nil pointer, got %v", err)
	}
}

func TestWalk_MultipleTagHandlers(t *testing.T) {
	type S struct {
		Name string `json:"name" validate:"required"`
	}

	var tags []string
	w := New()
	w.Handle("json", func(ctx FieldContext) error {
		tags = append(tags, "json:"+ctx.Tag.Name)
		return nil
	})
	w.Handle("validate", func(ctx FieldContext) error {
		tags = append(tags, "validate:"+ctx.Tag.Name)
		return nil
	})

	err := w.Walk(&S{})
	if err != nil {
		t.Fatal(err)
	}
	if len(tags) != 2 {
		t.Fatalf("tags = %v, want 2 entries", tags)
	}
}

func TestWalk_Depth(t *testing.T) {
	type Level2 struct {
		Value string `tag:"val"`
	}
	type Level1 struct {
		L2 Level2 `tag:"l2"`
	}
	type Root struct {
		L1 Level1 `tag:"l1"`
	}

	var depths []int
	w := New()
	w.Handle("tag", func(ctx FieldContext) error {
		depths = append(depths, ctx.Depth)
		return nil
	})

	err := w.Walk(&Root{})
	if err != nil {
		t.Fatal(err)
	}
	expected := []int{0, 1, 2}
	if len(depths) != len(expected) {
		t.Fatalf("depths = %v, want %v", depths, expected)
	}
	for i, d := range expected {
		if depths[i] != d {
			t.Errorf("depths[%d] = %d, want %d", i, depths[i], d)
		}
	}
}

func TestWalk_WithUnexported(t *testing.T) {
	type S struct {
		Public  string `tag:"public"`
		private string `tag:"private"` //nolint:unused
	}

	var paths []string
	w := New(WithUnexported(true))
	w.Handle("tag", func(ctx FieldContext) error {
		paths = append(paths, ctx.Path)
		return nil
	})

	s := S{Public: "yes"}
	err := w.Walk(&s)
	if err != nil {
		t.Fatal(err)
	}
	if len(paths) != 2 {
		t.Errorf("paths = %v, want 2 entries (including unexported)", paths)
	}
}

func TestWalk_WithFollowPointersFalse(t *testing.T) {
	type Inner struct {
		Value string `tag:"value"`
	}
	type Outer struct {
		Inner *Inner `tag:"inner"`
	}

	var paths []string
	w := New(WithFollowPointers(false))
	w.Handle("tag", func(ctx FieldContext) error {
		paths = append(paths, ctx.Path)
		return nil
	})

	err := w.Walk(&Outer{Inner: &Inner{Value: "test"}})
	if err != nil {
		t.Fatal(err)
	}
	// Should visit Inner field but not recurse into it
	if len(paths) != 1 {
		t.Errorf("paths = %v, want [Inner]", paths)
	}
}

func TestWalk_CycleDetection(t *testing.T) {
	type Node struct {
		Name string `tag:"name"`
		Next *Node  `tag:"next"`
	}

	a := &Node{Name: "a"}
	b := &Node{Name: "b"}
	a.Next = b
	b.Next = a // cycle

	var paths []string
	w := New()
	w.Handle("tag", func(ctx FieldContext) error {
		paths = append(paths, ctx.Path)
		return nil
	})

	err := w.Walk(a)
	if err != nil {
		t.Fatal(err)
	}
	// Should not infinite loop; cycle detection should break the loop
	if len(paths) > 10 {
		t.Errorf("visited %d paths; likely infinite loop", len(paths))
	}
}

func TestFieldContext_IsZero(t *testing.T) {
	type S struct {
		Name  string `tag:"name"`
		Count int    `tag:"count"`
	}

	var results []bool
	w := New()
	w.Handle("tag", func(ctx FieldContext) error {
		results = append(results, ctx.IsZero())
		return nil
	})

	err := w.Walk(&S{Name: "", Count: 0})
	if err != nil {
		t.Fatal(err)
	}
	if len(results) != 2 || !results[0] || !results[1] {
		t.Errorf("IsZero results = %v, want [true true]", results)
	}
}

func TestFieldContext_IsNil(t *testing.T) {
	type S struct {
		Ptr   *string `tag:"ptr"`
		Slice []int   `tag:"slice"`
	}

	var results []bool
	w := New()
	w.Handle("tag", func(ctx FieldContext) error {
		results = append(results, ctx.IsNil())
		return nil
	})

	err := w.Walk(&S{})
	if err != nil {
		t.Fatal(err)
	}
	if len(results) != 2 || !results[0] || !results[1] {
		t.Errorf("IsNil results = %v, want [true true]", results)
	}
}

func TestFieldContext_IsExported(t *testing.T) {
	type S struct {
		Public  string `tag:"public"`
		private string `tag:"private"` //nolint:unused
	}

	var results []bool
	w := New(WithUnexported(true))
	w.Handle("tag", func(ctx FieldContext) error {
		results = append(results, ctx.IsExported())
		return nil
	})

	err := w.Walk(&S{})
	if err != nil {
		t.Fatal(err)
	}
	if len(results) != 2 || !results[0] || results[1] {
		t.Errorf("IsExported results = %v, want [true false]", results)
	}
}

func TestWalk_CustomTagParser(t *testing.T) {
	type S struct {
		Name string `custom:"hello-world"`
	}

	w := New(WithTagParser(func(key, raw string) Tag {
		return Tag{Key: key, Name: "custom_" + raw, Raw: raw}
	}))

	var name string
	w.Handle("custom", func(ctx FieldContext) error {
		name = ctx.Tag.Name
		return nil
	})

	err := w.Walk(&S{})
	if err != nil {
		t.Fatal(err)
	}
	if name != "custom_hello-world" {
		t.Errorf("name = %q, want %q", name, "custom_hello-world")
	}
}

func TestWalkError_Unwrap(t *testing.T) {
	inner := errors.New("inner error")
	we := &WalkError{Path: "A.B", TagKey: "json", Field: "B", Err: inner}
	if !errors.Is(we, inner) {
		t.Error("WalkError should unwrap to inner error")
	}
}

func TestMultiError_Unwrap(t *testing.T) {
	inner1 := errors.New("err1")
	inner2 := errors.New("err2")
	me := &MultiError{
		Errors: []*WalkError{
			{Path: "A", Err: inner1},
			{Path: "B", Err: inner2},
		},
	}
	if !errors.Is(me, inner1) {
		t.Error("MultiError should unwrap to inner1")
	}
	if !errors.Is(me, inner2) {
		t.Error("MultiError should unwrap to inner2")
	}
}

func TestWalk_SliceOfPointers(t *testing.T) {
	type Item struct {
		Name string `tag:"name"`
	}
	type Container struct {
		Items []*Item `tag:"items"`
	}

	var paths []string
	w := New(WithSliceElements(true))
	w.Handle("tag", func(ctx FieldContext) error {
		paths = append(paths, ctx.Path)
		return nil
	})

	err := w.Walk(&Container{Items: []*Item{{Name: "a"}, nil, {Name: "c"}}})
	if err != nil {
		t.Fatal(err)
	}
	// Should visit Items, Items[0].Name, Items[2].Name (skip nil)
	if len(paths) != 3 {
		t.Fatalf("paths = %v, want 3 entries", paths)
	}
}
