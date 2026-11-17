---
title: Options
description: Every walker option and handler option in one place, with defaults.
---

## Walker options

Passed to `structwalker.New` or to `structwalker.Fields`.

| Option | Default | What it does |
|---|---|---|
| `WithErrorMode(mode)` | `StopOnError` | What happens when a handler returns an error. See [errors](/docs/errors/) |
| `WithFollowPointers(bool)` | `true` | Follow pointers to the structs behind them |
| `WithSliceElements(bool)` | `false` | Walk structs inside slices and arrays |
| `WithMapEntries(bool)` | `false` | Walk structs inside map values, in key order |
| `WithInterfaces(bool)` | `false` | Walk the struct or pointer an interface field holds |
| `WithUnexported(bool)` | `false` | Visit unexported fields. They can be read but not set |
| `WithLeafTypes(types...)` | `time.Time` | Struct types to treat as single values |
| `WithMaxDepth(n)` | `32` | How deep structs may nest before `ErrMaxDepth` |
| `WithTagParser(fn)` | `ParseTag` | A [custom tag parser](/docs/tag-parsers/) |
| `WithContext(ctx)` | | Deprecated. Use [`WalkContext`](/docs/context/) |

```go
w := structwalker.New(
    structwalker.WithErrorMode(structwalker.CollectErrors),
    structwalker.WithSliceElements(true),
    structwalker.WithMapEntries(true),
    structwalker.WithLeafTypes(reflect.TypeFor[uuid.UUID]()),
)
```

Options are fixed once the walker is created. If you need two configurations, make two walkers. They're cheap.

What each traversal option reaches is covered in [what gets walked](/docs/traversal/).

## Handler options

Passed to `Handle` or `HandleAll`.

| Option | Default | What it does |
|---|---|---|
| `WithPriority(n)` | `100` | Lower runs first, across every handler on the field |
| `WithPhase(phase)` | `BeforeChildren` | `AfterChildren` runs the handler once the field's children are done |
| `WithFilter(fn)` | | Only run on fields where `fn` returns true |

```go
w.Handle("default", setDefault, structwalker.WithPriority(10))
w.Handle("section", closeSection, structwalker.WithPhase(structwalker.AfterChildren))
w.Handle("validate", checkString, structwalker.WithFilter(isString))
```

See [handlers](/docs/handlers/#order) for how ties are broken.

## Walking

| Method | |
|---|---|
| `w.Walk(v)` | Walk a struct or a pointer to one |
| `w.WalkContext(ctx, v)` | The same with a [context](/docs/context/) |
| `w.WalkValue(rv)` | Walk a `reflect.Value` you already have |
| `w.WalkValueContext(ctx, rv)` | The same with a context |
| `w.Fields(v)` | An [iterator](/docs/fields/) using the walker's options |
| `structwalker.Fields(v, opts...)` | An iterator without a walker |

## Errors and sentinels

| | |
|---|---|
| `ErrSkipField` | Returned by a handler to [leave a field](/docs/flow-control/) |
| `ErrSkipChildren` | Returned by a handler to not descend into a field |
| `ErrMaxDepth` | The depth limit was reached |
| `ErrNotSettable` | `SetValue` on a field that [can't be changed](/docs/setting-values/#when-a-field-cant-be-set) |
| `*WalkError` | A handler error with its path |
| `*MultiError` | Every error from a `CollectErrors` walk |
| `*PanicError` | A handler panic caught by `WithRecovery` |

The full API reference is on [pkg.go.dev](https://pkg.go.dev/github.com/rjp2525/structwalker).
