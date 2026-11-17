---
title: Introduction
description: What structwalker is, the problem it takes off your hands and when you should reach for something else.
---

Every Go project ends up with the same reflection loop. Range over the fields of a struct, read a tag, recurse into the nested structs, remember the path so the error message makes sense. You write it for config defaults. Then again for validation. Then a third time to keep secrets out of the logs.

`structwalker` is that loop, written once. You say what a tag means. It finds the fields, parses the tags, follows nested structs and pointers and hands you each field with everything you need to read or change it.

## The shape of it

```go
w := structwalker.New()

w.Handle("default", func(ctx structwalker.FieldContext) error {
    if ctx.IsZero() {
        return ctx.SetValue(ctx.Tag.Raw)
    }
    return nil
})

err := w.Walk(&cfg)
```

A **walker** holds your handlers. A **handler** is a function you register for a tag key. **Walking** a struct calls each handler on every field that carries its tag, however deep the field is nested.

If you don't need handlers there is a plain loop too:

```go
for f := range structwalker.Fields(&cfg) {
    fmt.Println(f.Path, f.Value)
}
```

## What you get

- **Tags, parsed.** `validate:"email,required,max=255"` arrives as a name, flags and options.
- **The whole struct.** Nested structs and pointers by default. Slices, maps and interfaces when you ask.
- **Paths.** Every field knows where it is: `Address.Street`, `Items[2].Name`.
- **A setter that converts.** Give it `"30s"` and it fills a `time.Duration`.
- **Control.** Skip a field, skip its children, order your handlers, run them before or after the children.
- **Errors with context.** Stop at the first one or collect them all.
- **No dependencies.** Standard library only.

## When it's the right tool

Reach for it when you're about to write `reflect.TypeOf(v).NumField()` and you can already tell the loop will grow. Some things people build on it:

- [Defaults](/docs/defaults/) and [config from the environment](/docs/env-config/)
- [Validation](/docs/validation/) that reports every problem at once
- [Redaction](/docs/redaction/) before logging
- [Mapping](/docs/mapping/) structs to columns, flags or schemas
- [Diffs and audit logs](/docs/audit-diff/)

## When it isn't

Being upfront saves you an afternoon.

- **You need the fastest possible path.** Reflection is cached and a walk is quick, but generated code is quicker. If you're serialising millions of structs a second, generate code.
- **You want a finished validator or config loader.** structwalker is what you'd build one with. If `go-playground/validator` or `envconfig` already does what you need, use it.
- **You don't control the struct's tags.** The package is built around tags you define. You can walk any struct but the value is in the tags.

The full list is on the [limitations](/docs/limitations/) page. Otherwise, [install it](/docs/installation/) and have something working in a few minutes.
