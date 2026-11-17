---
title: Handlers
description: Register functions for tag keys, order them, filter them and choose whether they run before or after a field's children.
---

A handler is a function that runs on a field.

```go
type HandlerFunc func(ctx structwalker.FieldContext) error
```

## Tag handlers

`Handle` registers a handler for one tag key. It runs on every field that has that key.

```go
w := structwalker.New()

w.Handle("mask", func(ctx structwalker.FieldContext) error {
    return ctx.SetValue("***")
})

type Login struct {
    User     string
    Password string `mask:"true"`
}
```

Only `Password` reaches the handler. `User` has no `mask` tag.

A field tagged `mask:"-"` is skipped as well. `-` is Go's usual way to opt a field out and tag handlers respect it.

## Handlers for every field

`HandleAll` registers a handler that runs on every field, tagged or not.

```go
w.HandleAll(func(ctx structwalker.FieldContext) error {
    fmt.Println(ctx.Path)
    return nil
})
```

In a `HandleAll` handler `ctx.Tag` is empty, because the handler wasn't matched to a tag. Read tags with [`ctx.Lookup`](/docs/tags/#reading-other-tags).

## What a handler receives

| | |
|---|---|
| `ctx.Tag` | The parsed tag the handler was registered for |
| `ctx.AllTags` | Every tag on the field |
| `ctx.Lookup(key)` | Another tag on the same field |
| `ctx.Field` | The `reflect.StructField` |
| `ctx.Value` | The field's `reflect.Value` |
| `ctx.Path` | Where the field is, such as `Address.Street` |
| `ctx.Depth` | How many structs deep. Root fields are 0 |
| `ctx.Parent` | The struct holding the field |
| `ctx.Index` | The field's position in its parent |
| `ctx.Context()` | The walk's [`context.Context`](/docs/context/) |
| `ctx.SetValue(v)` | [Sets the field](/docs/setting-values/) |
| `ctx.IsZero()` | The field holds its zero value |
| `ctx.IsNil()` | The field is a nil pointer, interface, slice, map, channel or func |
| `ctx.IsExported()` | The field is exported |

## Order

Several handlers can run on one field. They run in priority order, lowest first. The default priority is 100.

```go
w.Handle("default", setDefault, structwalker.WithPriority(10))
w.Handle("validate", validate) // 100, so after defaults
w.HandleAll(audit, structwalker.WithPriority(1000))
```

Priority orders everything on the field: handlers for different tag keys and `HandleAll` handlers alike.

When priorities tie:

1. Tag handlers run in the order their tags are written on the field.
2. `HandleAll` handlers come after tag handlers.
3. Anything still tied runs in the order it was registered.

So with no priorities set, `json:"a" validate:"b"` runs the `json` handler before the `validate` handler.

## Filters

`WithFilter` narrows a handler to the fields where a function returns true.

```go
w.Handle("validate", validateString, structwalker.WithFilter(
    func(ctx structwalker.FieldContext) bool {
        return ctx.Value.Kind() == reflect.String
    },
))
```

A filtered-out handler is skipped. The other handlers on the field still run.

## Before or after children

By default a handler runs on a struct field before the walker goes into it. `WithPhase(AfterChildren)` runs it afterwards.

```go
w.Handle("section", open)
w.Handle("section", closeSection, structwalker.WithPhase(structwalker.AfterChildren))
```

```text
open:  Database
  ...  Database.Host
  ...  Database.Port
close: Database
```

After-children handlers are useful when a field's meaning depends on what's inside it: totals, "at least one of these is set" checks or closing something you opened.

## Returning errors

Returning an error stops the walk and you get it back from `Walk`, wrapped with the field's path. Two errors are special and steer the walk instead: [`ErrSkipField` and `ErrSkipChildren`](/docs/flow-control/). Collecting every error instead of stopping is covered in [errors](/docs/errors/).

## Registering late

You can call `Handle`, `HandleAll` and `Use` at any time, even while other goroutines are walking. A walk uses the handlers that were registered when it started.
