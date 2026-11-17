---
title: What gets walked
description: Nested structs, pointers, slices, maps, interfaces, embedded structs, cycles and depth. Which of them are on by default.
---

By default the walker visits every exported field and follows nested structs and pointers. Everything else is opt-in.

| | Default | Option |
|---|---|---|
| Exported fields | yes | |
| Nested structs | yes | |
| Pointers to structs | yes | `WithFollowPointers(false)` |
| Slices and arrays | no | `WithSliceElements(true)` |
| Maps | no | `WithMapEntries(true)` |
| Interface values | no | `WithInterfaces(true)` |
| Unexported fields | no | `WithUnexported(true)` |

## Nested structs and paths

```go
type Address struct{ Street, City string }

type User struct {
    Name    string
    Address Address
}
```

Handlers see `Name`, `Address`, `Address.Street` and `Address.City`, in that order. The struct field itself is visited before its children.

`ctx.Depth` is 0 for `Name` and `Address` and 1 for the fields inside `Address`.

## Pointers

Pointers are followed. A nil pointer is visited as a field and then there's nothing to go into.

```go
type Node struct {
    Name string
    Next *Node
}
```

Pointers to pointers and pointers to slices or maps work as well.

`WithFollowPointers(false)` stops the walker at every pointer, including pointers inside slices and maps.

## Cycles

A struct that points back to something the walk is already inside is visited as a field and not entered again.

```go
a := &Node{Name: "a"}
a.Next = &Node{Name: "b", Next: a}

// Name, Next, Next.Name, Next.Next
```

The same pointer appearing twice side by side is not a cycle and is walked both times.

## Slices and arrays

```go
w := structwalker.New(structwalker.WithSliceElements(true))

type Cart struct {
    Items []Item
}
// Items, Items[0].Name, Items[1].Name
```

Nesting is fine: `[][]Item`, `[]*Item` and `map[string][]Item` all reach the structs inside.

Handlers run on struct **fields**. A `[]string` field is one field. Its elements aren't visited one by one, so loop over `ctx.Value` yourself if you need them.

## Maps

```go
w := structwalker.New(structwalker.WithMapEntries(true))

type Config struct {
    Regions map[string]Region
}
// Regions, Regions[eu].Endpoint, Regions[us].Endpoint
```

Entries are visited in key order, so paths and errors come out the same on every run.

> [!NOTE]
> A struct stored directly in a map can be read but not set. That's a rule of Go's maps, not of this package. Use `map[string]*Region` when handlers need to change the values.

## Interfaces

```go
w := structwalker.New(structwalker.WithInterfaces(true))

type Event struct {
    Payload any
}
```

With the option on, the walker looks at what `Payload` holds and walks it if it's a struct or a pointer to one. It's off by default because an `any` or an `io.Reader` can hold anything, including types you'd never want to walk.

## Embedded structs

Embedded structs are walked like any other field and the path includes the type name: `Audit.CreatedBy`.

If the embedded type is unexported, its exported fields are still visited and can still be set. That matches how Go promotes them and how `encoding/json` treats them.

```go
type base struct{ ID int }

type User struct {
    base
    Name string
}
// base.ID, Name
```

## Unexported fields

`WithUnexported(true)` includes them. You can read them through the typed getters on `ctx.Value` but you can't set them. Calling `Value.Interface()` on one panics.

## Leaf types

Some structs are really single values: a timestamp, a decimal, a UUID. `time.Time` is treated that way out of the box. Add your own with `WithLeafTypes`.

```go
w := structwalker.New(
    structwalker.WithLeafTypes(reflect.TypeFor[decimal.Decimal]()),
)
```

Handlers still run on a field of a leaf type. The walker just doesn't go inside it.

## Depth

`WithMaxDepth(n)` limits how deep structs may nest. The default is 32. Reaching the limit is an error, `ErrMaxDepth`, because silently skipping part of a struct is how validators miss things.

```go
if errors.Is(err, structwalker.ErrMaxDepth) { ... }
```

## The root

`Walk` takes a struct or a pointer to one. A nil pointer is walked as nothing and returns `nil`. Anything else returns an error.
