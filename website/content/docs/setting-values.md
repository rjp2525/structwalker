---
title: Setting values
description: What SetValue converts, what it refuses and why a field sometimes can't be set at all.
---

`ctx.SetValue(v)` writes to the field a handler is looking at.

```go
w.Handle("default", func(ctx structwalker.FieldContext) error {
    if ctx.IsZero() {
        return ctx.SetValue(ctx.Tag.Raw)
    }
    return nil
})
```

Tag values are strings and fields usually aren't, so `SetValue` converts.

## From a string

| Field type | Example | Notes |
|---|---|---|
| `string` and named string types | `"hello"` | |
| `int`, `int8` … `int64` | `"42"` | must fit the field's size |
| `uint`, `uint8` … `uint64` | `"42"` | must fit, no negatives |
| `float32`, `float64` | `"1.5"` | |
| `bool` | `"true"`, `"1"`, `"f"` | anything `strconv.ParseBool` takes |
| `time.Duration` | `"1m30s"` | |
| `[]byte` | `"raw"` | |
| `time.Time` | `"2026-04-07T12:00:00Z"` | RFC 3339 |
| Any `encoding.TextUnmarshaler` | `"10.0.0.1"` into a `netip.Addr` | |

The last row does a lot of work. `netip.Addr`, `big.Int`, `slog.Level`, UUID and decimal types from the usual libraries and your own types all implement `UnmarshalText`. They work without structwalker knowing about them.

```go
type Level int

func (l *Level) UnmarshalText(b []byte) error { ... }

type Config struct {
    LogLevel Level `default:"warn"` // just works
}
```

## From other values

- A value of the field's own type is stored as it is.
- Numbers convert between numeric types when the value survives: `SetValue(42)` on an `int64` or a `float64` field is fine.
- `nil` resets the field to its zero value.

## Pointer fields

If the field is a pointer, `SetValue` allocates what it points to.

```go
type Limits struct {
    MaxConns *int `default:"10"`
}
// after the walk, *limits.MaxConns == 10
```

## What it refuses

`SetValue` never stores a value that changed on the way in.

```go
ctx.SetValue("300")  // int8 field:   cannot parse "300" as int8: value out of range
ctx.SetValue(-1)     // uint field:   -1 does not fit in uint
ctx.SetValue(1.5)    // int field:    1.5 does not fit in int
ctx.SetValue(65)     // string field: cannot assign int to string
```

When it returns an error the field is untouched.

Errors name the field first:

```text
field Port: cannot parse "abc" as int: invalid syntax
```

Return the error from your handler and the walker adds the tag: `structwalker: Port (tag "default"): field Port: cannot parse ...`.

## When a field can't be set

`SetValue` returns `ErrNotSettable` when Go's reflection rules say the field can't be changed.

```go
if errors.Is(err, structwalker.ErrNotSettable) { ... }
```

There are three ways to end up there:

- **You passed the struct by value.** `w.Walk(cfg)` gives handlers a copy. Use `w.Walk(&cfg)`.
- **The field is unexported.** With [`WithUnexported`](/docs/options/) you can read unexported fields but Go won't let you set them.
- **The struct lives directly in a map.** `map[string]Item` values aren't addressable. Use `map[string]*Item` if handlers need to change them.

## Reading the value

`ctx.Value` is a `reflect.Value`, so everything in the `reflect` package works on it.

```go
ctx.Value.String()
ctx.Value.Int()
ctx.Value.Interface() // the value as an any
```

> [!WARNING]
> `Value.Interface()` panics on an unexported field. If you walk with `WithUnexported(true)`, check `ctx.IsExported()` first or use the typed getters such as `Value.String()`.
