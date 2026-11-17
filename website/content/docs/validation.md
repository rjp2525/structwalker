---
title: Validation
description: A validator that reports every problem at once, with paths, in about thirty lines.
---

Validation is the classic use: check each field against rules in its tag and report everything that's wrong, not just the first thing.

```go
type Address struct {
    City string `json:"city" validate:"required"`
    Zip  string `json:"zip" validate:"len=5"`
}

type Signup struct {
    Email   string  `json:"email" validate:"required"`
    Age     int     `json:"age" validate:"min=18,max=120"`
    Role    string  `json:"role" validate:"oneof='admin,editor,viewer'"`
    Address Address `json:"address"`
}
```

## The walker

Two options do the heavy lifting.

```go
w := structwalker.New(
    structwalker.WithErrorMode(structwalker.CollectErrors),
    structwalker.WithTagParser(func(key, raw string) structwalker.Tag {
        if key == "validate" {
            raw = "," + raw // every item is a rule, none of them is a name
        }
        return structwalker.ParseTag(key, raw)
    }),
)
```

- `CollectErrors` keeps walking after a failure and returns all of them.
- The [tag parser](/docs/tag-parsers/) tweak makes every item in a `validate` tag a flag or an option. Without it the first item would be the tag's name.

## The rules

```go
w.Handle("validate", func(ctx structwalker.FieldContext) error {
    rules, v := ctx.Tag, ctx.Value

    if rules.HasFlag("required") && ctx.IsZero() {
        return errors.New("is required")
    }
    if n, ok := intRule(rules, "min"); ok && v.Int() < n {
        return fmt.Errorf("must be at least %d", n)
    }
    if n, ok := intRule(rules, "max"); ok && v.Int() > n {
        return fmt.Errorf("must be at most %d", n)
    }
    if n, ok := intRule(rules, "len"); ok && int64(v.Len()) != n {
        return fmt.Errorf("must be %d characters", n)
    }
    if allowed, ok := rules.Option("oneof"); ok && !slices.Contains(strings.Split(allowed, ","), v.String()) {
        return fmt.Errorf("must be one of %s", allowed)
    }
    return nil
})

func intRule(rules structwalker.Tag, name string) (int64, bool) {
    n, err := strconv.ParseInt(rules.OptionOr(name, ""), 10, 64)
    return n, err == nil
}
```

`oneof='admin,editor,viewer'` uses [single quotes](/docs/tags/#commas-inside-a-value) to keep its commas.

## The result

```go
err := w.Walk(&Signup{Age: 16, Role: "owner", Address: Address{Zip: "123"}})

if multi, ok := errors.AsType[*structwalker.MultiError](err); ok {
    for _, e := range multi.Errors {
        fmt.Printf("%s: %v\n", e.Path, e.Err)
    }
}
```

```text
Email: is required
Age: must be at least 18
Role: must be one of admin,editor,viewer
Address.City: is required
Address.Zip: must be 5 characters
```

Five problems, one pass, nested struct included.

## Report JSON names

API clients know `email_address`, not `Email`. Read the `json` tag in the handler with `Lookup`:

```go
name := ctx.Field.Name
if tag, ok := ctx.Lookup("json"); ok && tag.Name != "" {
    name = tag.Name
}
return fmt.Errorf("%s is required", name)
```

To build a full JSON path, keep a `map[string]string` from Go path to JSON path and fill it in a `HandleAll` handler that runs first.

## One rule per handler

As the rule set grows, split it up. Each rule becomes a handler with a [filter](/docs/handlers/#filters):

```go
func rule(name string, check structwalker.HandlerFunc) {
    w.Handle("validate", check, structwalker.WithFilter(func(ctx structwalker.FieldContext) bool {
        _, isOption := ctx.Tag.Option(name)
        return isOption || ctx.Tag.HasFlag(name)
    }))
}

rule("required", func(ctx structwalker.FieldContext) error { ... })
rule("email", func(ctx structwalker.FieldContext) error { ... })
```

Adding a rule is then one call and nothing else changes.

## Validating lists

Turn on [`WithSliceElements`](/docs/traversal/#slices-and-arrays) and the rules apply inside `[]Item` too, with paths like `Items[2].Name`.

## Share the walker

Build the validator once at startup and reuse it. It's safe to call from every request handler at once. The per-type work is cached after the first walk.

This recipe runs as `Example_validation` in the repository's tests.
