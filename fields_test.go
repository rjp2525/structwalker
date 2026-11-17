package structwalker

import (
	"fmt"
	"slices"
	"testing"
)

type fieldsAddress struct {
	Street string `json:"street"`
	City   string `json:"city"`
}

type fieldsUser struct {
	base
	Name    string `json:"name" env:"USER_NAME"`
	Address *fieldsAddress
	Tags    []fieldsAddress
	secret  string
}

func collect(seq func(func(FieldContext) bool)) []string {
	var paths []string
	for f := range seq {
		paths = append(paths, f.Path)
	}
	return paths
}

func TestFields_VisitsWhatWalkVisits(t *testing.T) {
	u := &fieldsUser{Address: &fieldsAddress{}, Tags: []fieldsAddress{{}}, secret: "x"}

	for _, opts := range [][]WalkerOption{
		nil,
		{WithSliceElements(true)},
		{WithUnexported(true), WithSliceElements(true)},
		{WithFollowPointers(false)},
	} {
		assertPaths(t, collect(Fields(u, opts...)), walkPaths(t, u, opts...)...)
	}

	assertPaths(t, collect(Fields(u)),
		"base.ID", "Name", "Address", "Address.Street", "Address.City", "Tags")
}

func TestFields_BreakStopsTheWalk(t *testing.T) {
	u := &fieldsUser{Address: &fieldsAddress{}}

	var paths []string
	for f := range Fields(u) {
		paths = append(paths, f.Path)
		if f.Path == "Address" {
			break
		}
	}
	assertPaths(t, paths, "base.ID", "Name", "Address")
}

func TestFields_SetAndLookup(t *testing.T) {
	env := map[string]string{"USER_NAME": "reno"}
	u := &fieldsUser{}

	for f := range Fields(u) {
		if f.Tag.Key != "" {
			t.Errorf("%s: Tag should be empty, got %q", f.Path, f.Tag.Key)
		}
		if tag, ok := f.Lookup("env"); ok {
			if err := f.SetValue(env[tag.Name]); err != nil {
				t.Fatal(err)
			}
		}
	}
	if u.Name != "reno" {
		t.Errorf("Name = %q", u.Name)
	}
}

func TestFields_HandlersDoNotRun(t *testing.T) {
	w := New()
	w.HandleAll(func(FieldContext) error { return fmt.Errorf("handler ran") })
	w.Use(func(ctx FieldContext, next HandlerFunc) error { return fmt.Errorf("middleware ran") })

	if got := len(collect(w.Fields(&fieldsAddress{}))); got != 2 {
		t.Errorf("got %d fields, want 2", got)
	}
}

func TestFields_NothingToWalk(t *testing.T) {
	var nilUser *fieldsUser
	for _, v := range []any{nil, 42, "text", nilUser, []fieldsUser{{}}} {
		if paths := collect(Fields(v)); len(paths) != 0 {
			t.Errorf("Fields(%#v) yielded %v", v, paths)
		}
	}
}

func TestFields_StopsDescendingAtMaxDepth(t *testing.T) {
	type C struct{ Leaf string }
	type B struct{ C C }
	type A struct {
		B    B
		Name string
	}

	assertPaths(t, collect(Fields(&A{}, WithMaxDepth(1))), "B", "B.C", "Name")
}

func TestFields_Cycle(t *testing.T) {
	type Node struct {
		Name string
		Next *Node
	}
	n := &Node{}
	n.Next = n

	assertPaths(t, collect(Fields(n)), "Name", "Next")
}

func TestFields_WorksWithSlicesPackage(t *testing.T) {
	names := slices.Collect(func(yield func(string) bool) {
		for f := range Fields(&fieldsAddress{}) {
			if tag, ok := f.Lookup("json"); ok && !yield(tag.Name) {
				return
			}
		}
	})
	if !slices.Equal(names, []string{"street", "city"}) {
		t.Errorf("got %v", names)
	}
}

func BenchmarkFields(b *testing.B) {
	v := &BenchStruct{}
	for b.Loop() {
		for f := range Fields(v) {
			_ = f.Path
		}
	}
}
