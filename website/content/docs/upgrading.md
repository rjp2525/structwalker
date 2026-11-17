---
title: Upgrading to v1.2
description: What changed since v1.0, the handful of behaviours to check and how to move off the one deprecated option.
---

v1.2 is a drop-in upgrade. Nothing was removed and no signature changed.

```bash
go get github.com/rjp2525/structwalker@latest
```

It does fix several bugs. A fix changes behaviour by definition. This page lists the ones you could notice. The [changelog](https://github.com/rjp2525/structwalker/blob/main/CHANGELOG.md) has everything.

## Check these

### Handler order across tag keys

Priority used to order handlers within one tag key. It now orders every handler on a field.

```go
w.Handle("validate", validate)                           // 100
w.Handle("default", setDefault, structwalker.WithPriority(10))
```

In v1.0 which ran first depended on the order of the tags on each field. In v1.2 `default` always runs first. If you never mixed priorities across keys, nothing changes.

### ErrSkipChildren

It used to stop the remaining tag handlers on the field as well as skipping the children. Now it only skips the children. Return [`ErrSkipField`](/docs/flow-control/) if you want the field left alone entirely.

### SetValue is strict

These used to succeed with the wrong value. They're errors now.

| Call | v1.0 stored | v1.2 |
|---|---|---|
| `SetValue(65)` on a `string` | `"A"` | error |
| `SetValue("300")` on an `int8` | `44` | error |
| `SetValue(-1)` on a `uint8` | `255` | error |
| `SetValue(1.5)` on an `int` | `1` | error |

### CollectErrors is flat

Errors from nested structs used to be wrapped inside another error. `MultiError.Errors` is now one flat list. If you wrote code to dig nested errors out, you can delete it.

### Map order

Map entries are visited in key order. They used to come in Go's random map order.

### Tags in single quotes

A value wrapped in single quotes now has them removed, because quotes are how you [keep a comma in a value](/docs/tags/#commas-inside-a-value). `default='hi'` gives the option value `hi`. `tag.Raw` is unchanged.

### time.Time with WithUnexported

`time.Time` is a [leaf type](/docs/traversal/#leaf-types). With `WithUnexported(true)` the walker no longer visits its internal fields.

### Error text

If you matched on message strings: `MultiError`, recovered panics, the depth error and `SetValue` errors all read differently. Match with `errors.Is` and `errors.As` instead. There are sentinels for everything now.

## Deprecated

### WithContext

It still works. A context on the walker is shared by every walk, which is rarely what you want.

```go
// before
w := structwalker.New(structwalker.WithContext(ctx))
err := w.Walk(&v)

// after
w := structwalker.New()
err := w.WalkContext(ctx, &v)
```

## New things worth a look

- [`Fields`](/docs/fields/): a `for range` loop over a struct's fields.
- [`WalkContext`](/docs/context/) and `ctx.Context()` in handlers.
- [`SetValue`](/docs/setting-values/) fills durations, pointer fields and any `TextUnmarshaler`.
- [`ctx.Lookup`](/docs/tags/#reading-other-tags) reads a field's other tags.
- [`WithInterfaces` and `WithLeafTypes`](/docs/traversal/), plus containers at any nesting depth.
- `ErrMaxDepth`, `ErrNotSettable` and `PanicError`.
- A walker is safe to share between goroutines from its first walk and walks about four times faster.
