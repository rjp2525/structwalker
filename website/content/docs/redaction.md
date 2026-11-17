---
title: Redaction
description: Mask sensitive fields before a value is logged, without touching the original.
---

Mark the sensitive fields and scrub them before the struct reaches a log line.

```go
type Card struct {
    Number string `log:"last4"`
    CVV    string `log:"-"`
}

type Customer struct {
    Name  string
    Email string `log:"redact"`
    Cards []Card
}
```

Three behaviours: `redact` replaces the value, `last4` keeps the last four characters and `-` blanks the field.

## The walker

```go
w := structwalker.New(structwalker.WithSliceElements(true))

w.HandleAll(func(ctx structwalker.FieldContext) error {
    tag, _ := ctx.Lookup("log")
    switch {
    case tag.IsIgnored():
        return ctx.SetValue(nil)
    case tag.Name == "redact":
        return ctx.SetValue("[redacted]")
    case tag.Name == "last4":
        s := ctx.Value.String()
        keep := max(len(s)-4, 0)
        return ctx.SetValue(strings.Repeat("*", keep) + s[keep:])
    }
    return nil
})
```

It uses `HandleAll` and `Lookup` instead of `Handle("log", ...)` for one reason: tag handlers aren't called for a `-` tag. Here `-` is an instruction.

`WithSliceElements(true)` is what gets the walker into `Cards`.

## Redact a copy

Handlers change the struct you give them, so give them a copy.

```go
safe := customer
safe.Cards = slices.Clone(customer.Cards)

if err := w.Walk(&safe); err != nil {
    return err
}

logger.Info("charge failed", "customer", safe)
```

```text
{Name:Ada Email:[redacted] Cards:[{Number:************4242 CVV:}]}
```

> [!WARNING]
> Assigning a struct copies it but slices, maps and pointers inside it still point at the original's data. Clone anything the walker is going to reach, or the "copy" will redact the real thing.

## Fail closed

If a handler can fail, decide what a failure means before it happens. For redaction the safe answer is usually "don't log it":

```go
if err := w.Walk(&safe); err != nil {
    logger.Error("could not redact customer", "error", err)
    return
}
```

Avoid `SkipOnError` here. It would carry on and log whatever wasn't scrubbed.

## Variations

- **Redact by type.** Give secrets their own type, such as `type Secret string`. Switch on `ctx.Field.Type` in the handler. No tags needed and nobody can forget one.
- **Allow-list instead.** Blank every string field that isn't tagged `log:"ok"`. It's stricter and new fields are private by default.
- **As a slog value.** Implement `LogValue() slog.Value` on the type and run the walker inside it. Then every log call is covered.

This recipe runs as `Example_redaction` in the repository's tests.
