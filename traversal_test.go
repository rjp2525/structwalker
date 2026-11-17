package structwalker

import (
	"reflect"
	"slices"
	"testing"
	"time"
)

func walkPaths(t *testing.T, v any, opts ...WalkerOption) []string {
	t.Helper()
	var paths []string
	w := New(opts...)
	w.HandleAll(func(ctx FieldContext) error {
		paths = append(paths, ctx.Path)
		return nil
	})
	if err := w.Walk(v); err != nil {
		t.Fatal(err)
	}
	return paths
}

func assertPaths(t *testing.T, got []string, want ...string) {
	t.Helper()
	if !slices.Equal(got, want) {
		t.Errorf("got  %v\nwant %v", got, want)
	}
}

type leaf struct {
	Name string
}

func TestTraversal_CycleVisitsEachStructOnce(t *testing.T) {
	type Node struct {
		Name string
		Next *Node
	}
	a := &Node{Name: "a"}
	a.Next = &Node{Name: "b", Next: a}

	assertPaths(t, walkPaths(t, a), "Name", "Next", "Next.Name", "Next.Next")
}

func TestTraversal_SelfPointer(t *testing.T) {
	type Node struct {
		Self *Node
	}
	n := &Node{}
	n.Self = n

	assertPaths(t, walkPaths(t, n), "Self")
}

func TestTraversal_SameAddressDifferentTypeIsNotACycle(t *testing.T) {
	type Outer struct {
		First leaf
		Again *leaf
	}
	o := &Outer{}
	o.Again = &o.First

	assertPaths(t, walkPaths(t, o), "First", "First.Name", "Again", "Again.Name")
}

func TestTraversal_SharedPointerIsWalkedEachTime(t *testing.T) {
	type Pair struct {
		A, B *leaf
	}
	shared := &leaf{}

	assertPaths(t, walkPaths(t, &Pair{A: shared, B: shared}), "A", "A.Name", "B", "B.Name")
}

func TestTraversal_NestedContainers(t *testing.T) {
	type S struct {
		Grid    [][]leaf
		Groups  map[string][]*leaf
		Deep    **leaf
		Ptr     *[]leaf
		Array   [2]leaf
		Scalars []int
	}
	l := &leaf{}
	s := &S{
		Grid:   [][]leaf{{{}}, {{}, {}}},
		Groups: map[string][]*leaf{"b": {{}}, "a": {{}, nil}},
		Deep:   &l,
		Ptr:    &[]leaf{{}},
	}

	got := walkPaths(t, s, WithSliceElements(true), WithMapEntries(true))
	assertPaths(t, got,
		"Grid", "Grid[0][0].Name", "Grid[1][0].Name", "Grid[1][1].Name",
		"Groups", "Groups[a][0].Name", "Groups[b][0].Name",
		"Deep", "Deep.Name",
		"Ptr", "Ptr[0].Name",
		"Array", "Array[0].Name", "Array[1].Name",
		"Scalars",
	)
}

func TestTraversal_ContainersAreOptIn(t *testing.T) {
	type S struct {
		Items []leaf
		ByKey map[string]leaf
		Any   any
	}
	s := &S{Items: []leaf{{}}, ByKey: map[string]leaf{"a": {}}, Any: leaf{}}

	assertPaths(t, walkPaths(t, s), "Items", "ByKey", "Any")
}

func TestTraversal_FollowPointersFalseAppliesEverywhere(t *testing.T) {
	type S struct {
		Direct *leaf
		Items  []*leaf
		ByKey  map[string]*leaf
		Values []leaf
	}
	s := &S{
		Direct: &leaf{},
		Items:  []*leaf{{}},
		ByKey:  map[string]*leaf{"a": {}},
		Values: []leaf{{}},
	}

	got := walkPaths(t, s, WithFollowPointers(false), WithSliceElements(true), WithMapEntries(true))
	assertPaths(t, got, "Direct", "Items", "ByKey", "Values", "Values[0].Name")
}

func TestTraversal_Interfaces(t *testing.T) {
	type S struct {
		Value   any
		Pointer any
		Scalar  any
		Empty   any
		List    []any
	}
	s := &S{Value: leaf{}, Pointer: &leaf{}, Scalar: 42, List: []any{leaf{}, "x", &leaf{}}}

	got := walkPaths(t, s, WithInterfaces(true), WithSliceElements(true))
	assertPaths(t, got,
		"Value", "Value.Name",
		"Pointer", "Pointer.Name",
		"Scalar", "Empty",
		"List", "List[0].Name", "List[2].Name",
	)
}

func TestTraversal_InterfacePointerIsSettable(t *testing.T) {
	type S struct {
		Any any
	}
	target := &leaf{}

	w := New(WithInterfaces(true))
	w.HandleAll(func(ctx FieldContext) error {
		if ctx.Path == "Any.Name" {
			return ctx.SetValue("set")
		}
		return nil
	})
	if err := w.Walk(&S{Any: target}); err != nil {
		t.Fatal(err)
	}
	if target.Name != "set" {
		t.Errorf("Name = %q", target.Name)
	}
}

func TestTraversal_SelfContainingSliceTerminates(t *testing.T) {
	type S struct {
		List []any
	}
	s := &S{List: make([]any, 2)}
	s.List[0] = s.List
	s.List[1] = leaf{}

	got := walkPaths(t, s, WithInterfaces(true), WithSliceElements(true))
	assertPaths(t, got, "List", "List[1].Name")
}

type base struct {
	ID int `tag:"id"`
}

type hidden struct {
	Secret string
}

func TestTraversal_UnexportedEmbeddedStructPromotesFields(t *testing.T) {
	type S struct {
		base
		*hidden
		Name string
	}
	s := &S{hidden: &hidden{}}

	w := New()
	var paths []string
	w.HandleAll(func(ctx FieldContext) error {
		paths = append(paths, ctx.Path)
		if ctx.Path == "base.ID" {
			return ctx.SetValue(7)
		}
		return nil
	})
	if err := w.Walk(s); err != nil {
		t.Fatal(err)
	}

	assertPaths(t, paths, "base.ID", "hidden.Secret", "Name")
	if s.ID != 7 {
		t.Errorf("ID = %d, want 7", s.ID)
	}
}

func TestTraversal_UnexportedEmbeddedStructWithUnexportedOption(t *testing.T) {
	type S struct {
		base
	}

	assertPaths(t, walkPaths(t, &S{}, WithUnexported(true)), "base", "base.ID")
}

func TestTraversal_MapKeysAreSorted(t *testing.T) {
	type S struct {
		ByInt  map[int]leaf
		ByName map[string]leaf
		ByBool map[bool]leaf
	}
	s := &S{
		ByInt:  map[int]leaf{10: {}, 2: {}, -1: {}},
		ByName: map[string]leaf{"b": {}, "a": {}, "c": {}},
		ByBool: map[bool]leaf{true: {}, false: {}},
	}

	for range 20 {
		assertPaths(t, walkPaths(t, s, WithMapEntries(true)),
			"ByInt", "ByInt[-1].Name", "ByInt[2].Name", "ByInt[10].Name",
			"ByName", "ByName[a].Name", "ByName[b].Name", "ByName[c].Name",
			"ByBool", "ByBool[false].Name", "ByBool[true].Name",
		)
	}
}

func TestTraversal_UnexportedMapDoesNotPanic(t *testing.T) {
	type S struct {
		byKey map[string]leaf
	}

	got := walkPaths(t, &S{byKey: map[string]leaf{"a": {}}}, WithUnexported(true), WithMapEntries(true))
	assertPaths(t, got, "byKey", "byKey[a].Name")
}

func TestTraversal_TimeIsALeaf(t *testing.T) {
	type S struct {
		At time.Time
	}

	assertPaths(t, walkPaths(t, &S{At: time.Now()}, WithUnexported(true)), "At")
}

func TestTraversal_WithLeafTypes(t *testing.T) {
	type S struct {
		Leaf  leaf
		Ptr   *leaf
		Other base
	}

	got := walkPaths(t, &S{Ptr: &leaf{}}, WithLeafTypes(reflect.TypeFor[leaf]()))
	assertPaths(t, got, "Leaf", "Ptr", "Other", "Other.ID")
}

func TestTraversal_DepthCountsStructsOnly(t *testing.T) {
	type S struct {
		Grid [][]*leaf
	}

	depths := map[string]int{}
	w := New(WithSliceElements(true))
	w.HandleAll(func(ctx FieldContext) error {
		depths[ctx.Path] = ctx.Depth
		return nil
	})
	if err := w.Walk(&S{Grid: [][]*leaf{{{}}}}); err != nil {
		t.Fatal(err)
	}
	if depths["Grid"] != 0 || depths["Grid[0][0].Name"] != 1 {
		t.Errorf("depths = %v", depths)
	}
}
