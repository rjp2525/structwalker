---
title: Middleware
description: Wrap every handler call for recovery, logging, timing or anything of your own.
---

Middleware is a function that wraps every handler call.

```go
type Middleware func(ctx structwalker.FieldContext, next structwalker.HandlerFunc) error
```

Call `next(ctx)` to run the handler. Do things before and after it. Or don't call it at all and the handler doesn't run.

```go
w := structwalker.New()
w.Use(structwalker.WithRecovery())
```

## Order

The first middleware you add is the outermost.

```go
w.Use(first, second)
```

```text
first → second → handler → second → first
```

`Use` applies to every handler on the walker, including ones registered before you called it. It doesn't matter what order you set things up in.

## Recovery

A panic in a handler normally takes the program down. `WithRecovery` turns it into an error.

```go
w.Use(structwalker.WithRecovery())

err := w.Walk(&v)
// structwalker: Name (tag "validate"): panic: runtime error: index out of range
```

The error is a `*PanicError` with the value that was panicked and the stack at the time.

```go
if p, ok := errors.AsType[*structwalker.PanicError](err); ok {
    log.Printf("handler panicked: %v\n%s", p.Value, p.Stack)
}
```

If the handler panicked with an `error`, `errors.Is` and `errors.As` see through to it.

Put recovery first so it wraps everything else.

## Logging

`WithLogging` takes a `*slog.Logger`. It logs each handler call at debug level and each failure at error level.

```go
w.Use(structwalker.WithLogging(slog.Default()))
```

```text
level=DEBUG msg="structwalker: visiting field" path=Address.City tag=validate depth=1
level=ERROR msg="structwalker: handler error" path=Address.City tag=validate error="is required"
```

`ErrSkipField` and `ErrSkipChildren` aren't logged as errors. They're [instructions](/docs/flow-control/), not failures.

## Timing

`WithTiming` calls you back with how long each handler took.

```go
w.Use(structwalker.WithTiming(func(path, tagKey string, d time.Duration) {
    handlerSeconds.WithLabelValues(tagKey).Observe(d.Seconds())
}))
```

## Writing your own

A middleware that only lets handlers touch fields that are still empty:

```go
func OnlyZero() structwalker.Middleware {
    return func(ctx structwalker.FieldContext, next structwalker.HandlerFunc) error {
        if !ctx.IsZero() {
            return nil
        }
        return next(ctx)
    }
}
```

One that adds detail to every error:

```go
func Describe() structwalker.Middleware {
    return func(ctx structwalker.FieldContext, next structwalker.HandlerFunc) error {
        if err := next(ctx); err != nil {
            return fmt.Errorf("%s field: %w", ctx.Field.Type, err)
        }
        return nil
    }
}
```

Wrap with `%w` so `ErrSkipField` and `ErrSkipChildren` keep working when they pass through.

## What middleware sees

`ctx.Tag` is the tag of the handler being wrapped. It's empty for `HandleAll` handlers.

Middleware runs once per handler call, not once per field. A field with three handlers goes through the chain three times.

Middleware doesn't run in a [`Fields`](/docs/fields/) loop. There are no handlers to wrap.
