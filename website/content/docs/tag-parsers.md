---
title: Custom tag parsers
description: Replace the built-in tag parser when your tags use a different syntax, for one key or for all of them.
---

The built-in parser reads `name,flag,key=value`. If some of your tags are shaped differently, give the walker your own parser.

```go
w := structwalker.New(structwalker.WithTagParser(parse))
```

```go
func parse(key, raw string) structwalker.Tag
```

It receives one tag key and its value, already unquoted. It returns the `Tag` handlers will see. It's called once per field per struct type and the result is cached.

## Adjust one key

Most of the time you want different behaviour for one key and the built-in parser for the rest. Switch on the key and fall through to `ParseTag`.

A validation tag where every item is a rule and none of them is a name:

```go
structwalker.WithTagParser(func(key, raw string) structwalker.Tag {
    if key == "validate" {
        raw = "," + raw
    }
    return structwalker.ParseTag(key, raw)
})
```

The leading comma gives the tag an empty name, so `validate:"required,min=18"` parses to the flag `required` and the option `min`. The [validation recipe](/docs/validation/) uses exactly this.

## A different separator

Tags that use spaces and colons, such as `rule:"min:3 max:10 required"`:

```go
func parse(key, raw string) structwalker.Tag {
    if key != "rule" {
        return structwalker.ParseTag(key, raw)
    }

    tag := structwalker.Tag{Key: key, Raw: raw, Options: map[string]string{}}
    for _, item := range strings.Fields(raw) {
        if k, v, ok := strings.Cut(item, ":"); ok {
            tag.Options[k] = v
        } else {
            tag.Flags = append(tag.Flags, item)
        }
    }
    return tag
}
```

## What to fill in

| Field | Used for |
|---|---|
| `Key` | Matching the tag to handlers. Keep the key you were given unless you mean to redirect it |
| `Name` | `IsIgnored` returns true when it's `-`, which stops tag handlers running |
| `Raw` | Yours to define. Handlers often read it for single-value tags |
| `Flags`, `Options` | `HasFlag`, `Option` and `OptionOr` |

## Things to know

- The parser applies to **every** key on every field, including `json` and `db`. Fall through to `ParseTag` for the keys you don't care about.
- It has to be safe to call from several goroutines. Walks can run concurrently and two of them may meet a new type at the same moment.
- It's set when the walker is created. There's no changing it afterwards.
