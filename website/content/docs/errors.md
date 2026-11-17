---
title: Errors
description: The three error modes, what a WalkError carries and how to get at the error your handler returned.
---

When a handler returns an error, the walker wraps it in a `*WalkError` that says where it happened.

```go
type WalkError struct {
    Path   string // "Address.City"
    TagKey string // "validate", empty for HandleAll handlers
    Field  string // "City"
    Err    error  // what the handler returned
}
```

```text
structwalker: Address.City (tag "validate"): is required
```

## Three modes

`WithErrorMode` decides what happens next.

| Mode | Behaviour | Good for |
|---|---|---|
| `StopOnError` | The walk ends and `Walk` returns the error. The default | defaults, loading config |
| `CollectErrors` | The walk continues. `Walk` returns every error as a `*MultiError` | validation |
| `SkipOnError` | Handler errors are dropped | best-effort passes such as redaction |

## Collecting

```go
w := structwalker.New(structwalker.WithErrorMode(structwalker.CollectErrors))
w.Handle("validate", validate)

err := w.Walk(&signup)
```

```text
structwalker: 3 errors:
  - Email (tag "validate"): is required
  - Age (tag "validate"): must be at least 18
  - Address.City (tag "validate"): is required
```

`MultiError.Errors` is a flat list in the order the walker met them, however deeply nested the fields were.

```go
if multi, ok := errors.AsType[*structwalker.MultiError](err); ok {
    for _, e := range multi.Errors {
        problems[e.Path] = e.Err.Error()
    }
}
```

That loop is most of an API's validation response.

## Getting your own error back

Both types unwrap, so `errors.Is` and `errors.As` reach whatever the handler returned.

```go
var ErrRequired = errors.New("is required")

err := w.Walk(&signup)

errors.Is(err, ErrRequired) // true if any field failed with it
```

```go
if walkErr, ok := errors.AsType[*structwalker.WalkError](err); ok {
    fmt.Println(walkErr.Path)
}
```

## Errors that aren't from handlers

| Error | When |
|---|---|
| `ErrMaxDepth` | A struct is nested deeper than [`WithMaxDepth`](/docs/traversal/#depth) allows. It arrives as a `*WalkError` with the path |
| the context's error | The [context](/docs/context/) was cancelled or timed out |
| `expected struct` | `Walk` was given something that isn't a struct or a pointer to one |

In `CollectErrors` mode `ErrMaxDepth` is collected like any other error and the rest of the walk carries on. A cancelled context always ends the walk and is returned on its own.

## Panics

A panic in a handler is a panic. Add the [recovery middleware](/docs/middleware/#recovery) to turn it into an error with a stack trace.
