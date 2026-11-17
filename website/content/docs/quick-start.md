---
title: Quick start
description: Build a default tag and a validator in about five minutes. See how handlers share one walk.
---

We'll give two tags a meaning: `default` fills in empty fields and `validate` checks them.

## 1. A struct with tags

```go
type Server struct {
    Host    string        `default:"localhost" validate:"required"`
    Port    int           `default:"8080"`
    Timeout time.Duration `default:"30s"`
    TLS     struct {
        Cert string `validate:"required"`
        Key  string `validate:"required"`
    }
}
```

Nothing here knows about structwalker. They're ordinary struct tags.

## 2. A walker and two handlers

```go
w := structwalker.New(
    structwalker.WithErrorMode(structwalker.CollectErrors),
)

w.Handle("default", func(ctx structwalker.FieldContext) error {
    if ctx.IsZero() {
        return ctx.SetValue(ctx.Tag.Raw)
    }
    return nil
}, structwalker.WithPriority(10))

w.Handle("validate", func(ctx structwalker.FieldContext) error {
    if ctx.Tag.Name == "required" && ctx.IsZero() {
        return errors.New("is required")
    }
    return nil
})
```

Three things to notice:

- A handler is registered for a **tag key**. It runs on every field that has that key and on no others.
- `ctx.Tag` is the parsed tag. `ctx.SetValue` converts the string `"8080"` to the field's `int` and `"30s"` to a `time.Duration`.
- `WithPriority(10)` makes defaults run before validation, whose priority is the default 100. `Host` gets its default first and then passes the `required` check.

## 3. Walk

```go
var srv Server
err := w.Walk(&srv)
```

Pass a pointer. Handlers can only set fields they can reach through one.

```go
fmt.Println(srv.Host, srv.Port, srv.Timeout)
// localhost 8080 30s

fmt.Println(err)
// structwalker: 2 errors:
//   - TLS.Cert (tag "validate"): is required
//   - TLS.Key (tag "validate"): is required
```

The walker went into the nested `TLS` struct on its own and the errors say where the problem is. `CollectErrors` is why you got both instead of only the first.

## 4. Use the errors

```go
if multi, ok := errors.AsType[*structwalker.MultiError](err); ok {
    for _, e := range multi.Errors {
        fmt.Println(e.Path, e.Err)
    }
}
// TLS.Cert is required
// TLS.Key is required
```

## Where next

- [Handlers](/docs/handlers/): priorities, filters and running after children.
- [Tags](/docs/tags/): the tag format and what you get from a parsed tag.
- [Setting values](/docs/setting-values/): what `SetValue` converts.
- [Recipes](/docs/defaults/): complete, tested versions of the common jobs.
