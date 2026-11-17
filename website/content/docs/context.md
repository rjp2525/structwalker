---
title: Context and cancellation
description: Pass a context to a walk, stop a walk early and use the context inside handlers.
---

`WalkContext` is `Walk` with a `context.Context`.

```go
err := w.WalkContext(ctx, &v)
```

## Cancellation

The walker checks the context before each field. Once it's done the walk stops and returns the context's error.

```go
ctx, cancel := context.WithTimeout(r.Context(), 50*time.Millisecond)
defer cancel()

err := w.WalkContext(ctx, &payload)
if errors.Is(err, context.DeadlineExceeded) {
    // took too long
}
```

A handler that's already running isn't interrupted. It has to watch the context itself, which is the next section.

## Inside a handler

`ctx.Context()` returns the context the walk was started with. Pass it to anything that does I/O.

```go
w.Handle("setting", func(fc structwalker.FieldContext) error {
    value, err := settings.Get(fc.Context(), fc.Tag.Name)
    if err != nil {
        return err
    }
    return fc.SetValue(value)
})
```

It also carries request-scoped values, so a handler can read a trace ID or the current user without them being threaded through by hand.

```go
user := auth.UserFrom(fc.Context())
```

`Context()` never returns nil. In a plain `Walk` it's `context.Background()`.

> [!NOTE]
> The handler argument is conventionally named `ctx` in these docs. When you also use `context.Context` in the same function, calling it `fc` keeps the two apart.

## One walker, many requests

A walker is built once and shared. The context belongs to a single walk, so pass it per call.

```go
var validator = newValidator() // a *structwalker.Walker, built at startup

func handle(w http.ResponseWriter, r *http.Request) {
    var req CreateOrder
    // decode ...
    if err := validator.WalkContext(r.Context(), &req); err != nil {
        // respond 422
    }
}
```

## WithContext is deprecated

Earlier versions set the context on the walker with `WithContext`. That context was shared by every walk, so a walker built with a five second timeout stopped working after five seconds.

It still works. To move off it, drop the option and pass the context to `WalkContext`:

```go
// before
w := structwalker.New(structwalker.WithContext(ctx))
err := w.Walk(&v)

// after
w := structwalker.New()
err := w.WalkContext(ctx, &v)
```
