---
title: Diffs and audit logs
description: List what changed between two values of the same type, by path.
---

"What changed?" is two walks and a map.

```go
type Limits struct {
    Requests int
    Burst    int
}

type Plan struct {
    Name   string
    Price  int
    Limits Limits
}
```

## The diff

```go
type Change struct {
    Path     string
    Old, New any
}

func Diff[T any](before, after *T) []Change {
    old := map[string]any{}
    for f := range structwalker.Fields(before) {
        if f.Value.Kind() != reflect.Struct {
            old[f.Path] = f.Value.Interface()
        }
    }

    var changes []Change
    for f := range structwalker.Fields(after) {
        was, tracked := old[f.Path]
        if now := f.Value.Interface(); tracked && !reflect.DeepEqual(was, now) {
            changes = append(changes, Change{f.Path, was, now})
        }
    }
    return changes
}
```

```go
before := Plan{Name: "Pro", Price: 20, Limits: Limits{Requests: 1000, Burst: 50}}
after := Plan{Name: "Pro", Price: 25, Limits: Limits{Requests: 5000, Burst: 50}}

for _, c := range Diff(&before, &after) {
    fmt.Printf("%s: %v -> %v\n", c.Path, c.Old, c.New)
}
```

```text
Price: 20 -> 25
Limits.Requests: 1000 -> 5000
```

## Why skip struct fields

The walker visits `Limits` and then `Limits.Requests`. Recording both would report the change twice, once for the whole struct and once for the field. Skipping fields whose kind is `Struct` keeps only the leaves.

`time.Time` is a struct too, so with this check it would be left out. Compare on the [leaf types](/docs/traversal/#leaf-types) you care about instead:

```go
leaf := f.Value.Kind() != reflect.Struct || f.Field.Type == reflect.TypeFor[time.Time]()
```

## Leaving fields out

Use a tag to keep noise out of the audit log:

```go
type Plan struct {
    Name      string
    UpdatedAt time.Time `audit:"-"`
    APIKey    string    `audit:"redact"`
}
```

```go
tag, _ := f.Lookup("audit")
switch {
case tag.IsIgnored():
    continue
case tag.Name == "redact":
    changes = append(changes, Change{f.Path, "[redacted]", "[redacted]"})
    continue
}
```

## Slices and maps

With no options a slice or map field is one leaf and `reflect.DeepEqual` compares it whole. You learn that `Tags` changed, not which element.

To go inside, pass `WithSliceElements(true)` or `WithMapEntries(true)` to both loops. Paths then look like `Items[2].Price`. Elements that only exist on one side show up in one loop and not the other, so handle the "not tracked" case if you need additions and removals.

## Writing the audit entry

```go
if changes := Diff(&before, &after); len(changes) > 0 {
    audit.Record(ctx, user, "plan.updated", changes)
}
```

Stored as JSON, `changes` is a readable history: which field, what it was and what it became.

This recipe runs as `Example_diff` in the repository's tests.
