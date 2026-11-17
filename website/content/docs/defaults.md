---
title: Default values
description: A default tag that fills empty fields, including durations, times and pointers.
---

The smallest useful thing you can build: a `default` tag.

```go
type Config struct {
    Port     int           `default:"8080"`
    Timeout  time.Duration `default:"30s"`
    Started  time.Time     `default:"2026-04-07T00:00:00Z"`
    MaxConns *int          `default:"10"`
    Name     string
}
```

## The handler

```go
w := structwalker.New()

w.Handle("default", func(ctx structwalker.FieldContext) error {
    if ctx.IsZero() {
        return ctx.SetValue(ctx.Tag.Raw)
    }
    return nil
})
```

```go
cfg := Config{Port: 9000}
if err := w.Walk(&cfg); err != nil {
    return err
}

fmt.Println(cfg.Port, cfg.Timeout, cfg.Started.Year(), *cfg.MaxConns)
// 9000 30s 2026 10
```

`Port` was already set, so it's left alone. Everything else with a tag is filled in. `Name` has no tag and isn't touched.

## Why it's this short

- `ctx.IsZero()` is true for `0`, `""`, a nil pointer and a zero `time.Time`. It's the "nobody set this" check.
- `ctx.Tag.Raw` is the tag value exactly as written. Using `Raw` instead of `Name` means a default containing a comma works.
- [`SetValue`](/docs/setting-values/) converts the string to whatever the field is. It allocates the `*int` for you.

## A bad default is an error

```go
type Config struct {
    Port int `default:"eighty"`
}
```

```text
structwalker: Port (tag "default"): field Port: cannot parse "eighty" as int: invalid syntax
```

You find out the first time the struct is walked, not when the wrong port is used. A one-line test that walks your config struct is enough to catch a typo in a tag.

## Zero is a real value sometimes

`IsZero` can't tell "not set" from "set to zero". If `0` or `false` is a meaningful setting, make the field a pointer. Nil then means "not set":

```go
type Config struct {
    Retries *int `default:"3"` // nil gets 3, a pointer to 0 stays 0
}
```

## Running it with other handlers

Give defaults a low priority number so they run before anything that reads the value.

```go
w.Handle("default", setDefault, structwalker.WithPriority(10))
w.Handle("validate", validate)
```

This recipe runs as `ExampleFieldContext_SetValue` in the repository's tests.
