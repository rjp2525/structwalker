---
title: The Fields iterator
description: Loop over a struct's fields with for range, without a walker or handlers.
---

Handlers earn their keep when several concerns share a walk. For a one-off they're ceremony. `Fields` is the short version.

```go
for f := range structwalker.Fields(&cfg) {
    fmt.Println(f.Path, f.Value)
}
```

It visits the same fields as `Walk` in the same order. It gives you the same [`FieldContext`](/docs/handlers/#what-a-handler-receives).

## A real loop

Loading config from the environment:

```go
for f := range structwalker.Fields(&cfg) {
    tag, ok := f.Lookup("env")
    if !ok {
        continue
    }
    if v, ok := os.LookupEnv(tag.Name); ok {
        if err := f.SetValue(v); err != nil {
            return err
        }
    }
}
```

`continue`, `break` and `return` all do what you'd expect. `break` stops the walk.

## Tags in a loop

No handler was matched, so `f.Tag` is empty. Use `f.Lookup(key)`:

```go
tag, ok := f.Lookup("json")
```

## Options

Pass the same [options](/docs/options/) you'd give `New`:

```go
for f := range structwalker.Fields(&order, structwalker.WithSliceElements(true)) {
    // also visits order.Items[0].Name, order.Items[1].Name ...
}
```

If you loop over the same types often, keep a walker and call its method. It reuses the cached reflection work.

```go
var w = structwalker.New(structwalker.WithSliceElements(true))

for f := range w.Fields(&order) { ... }
```

Calling `structwalker.Fields` with no options shares one built-in walker, so that's cached too.

## How it differs from Walk

A `for range` loop has nowhere to return an error, so `Fields` doesn't produce any.

| | `Walk` | `Fields` |
|---|---|---|
| Not a struct | returns an error | yields nothing |
| Nested past the depth limit | `ErrMaxDepth` | stops descending |
| Registered handlers | run | don't run |
| Middleware | runs | doesn't run |
| Skipping children | `ErrSkipChildren` | not available |

If you need to prune parts of the struct or you want middleware, use a [handler](/docs/handlers/).

## Collecting

It's a standard `iter.Seq`, so it works with the rest of the standard library:

```go
var paths []string
for f := range structwalker.Fields(&cfg) {
    paths = append(paths, f.Path)
}
```
