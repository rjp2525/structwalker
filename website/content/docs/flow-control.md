---
title: Steering the walk
description: Skip a field, skip its children and understand exactly what each choice stops.
---

A handler steers the walk with its return value.

| Return | Effect |
|---|---|
| `nil` | Carry on |
| `structwalker.ErrSkipField` | Leave this field: no more handlers, no children |
| `structwalker.ErrSkipChildren` | Don't descend into this field. Its other handlers still run |
| any other error | A failure. What happens next depends on the [error mode](/docs/errors/) |

## Skip a field

```go
w.Handle("audit", func(ctx structwalker.FieldContext) error {
    if ctx.Tag.Name == "ignore" {
        return structwalker.ErrSkipField
    }
    return nil
}, structwalker.WithPriority(1))
```

`ErrSkipField` ends the field. Handlers that would have run later on it don't run, in either phase. The walker doesn't go into it. The walk continues with the next field.

Give the handler that decides a low priority number so it runs before the ones it's meant to stop.

## Skip the children

```go
type Config struct {
    Name    string
    Secrets Secrets `walk:"opaque"`
}

w.Handle("walk", func(ctx structwalker.FieldContext) error {
    if ctx.Tag.Name == "opaque" {
        return structwalker.ErrSkipChildren
    }
    return nil
})
```

The walker visits `Secrets` itself but none of its fields. Every other handler on `Secrets` still runs, including after-children handlers.

Returned from an [after-children](/docs/handlers/#before-or-after-children) handler, `ErrSkipChildren` does nothing. The children have already been walked.

## Types you never want to walk into

If a struct type should always be treated as a single value, say so once with [`WithLeafTypes`](/docs/traversal/#leaf-types) and skip the handler.

## Wrapping is fine

Both are checked with `errors.Is`, so a wrapped sentinel still works.

```go
return fmt.Errorf("%s is deprecated: %w", ctx.Path, structwalker.ErrSkipField)
```

## They aren't errors

Neither value ever comes back from `Walk`. The [logging middleware](/docs/middleware/#logging) doesn't report them as errors either.

## In a Fields loop

A [`Fields`](/docs/fields/) loop has `continue` and `break`. There's no way to skip a field's children from inside a loop. Use a handler when you need that.
