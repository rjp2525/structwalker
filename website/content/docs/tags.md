---
title: Tags
description: The tag format structwalker understands, what a parsed tag gives you and how to read a field's other tags.
---

A struct tag is a list of `key:"value"` pairs. structwalker parses each value into a `Tag`.

```go
Email string `json:"email,omitempty" validate:"email,required,max=255"`
```

## The format

A value is a comma-separated list.

- The **first** item is the name.
- An item with `=` is an **option**.
- Anything else is a **flag**.

```go
tag := structwalker.ParseTag("validate", "email,required,max=255")

tag.Key                  // "validate"
tag.Name                 // "email"
tag.HasFlag("required")  // true
tag.Option("max")        // "255", true
tag.OptionOr("min", "0") // "0"
tag.Raw                  // "email,required,max=255"
```

This is the convention `encoding/json` uses (`json:"name,omitempty"`), extended with `key=value`.

> [!TIP]
> When a tag is a single value, like `default:"8080"`, use `tag.Raw`. It's the value exactly as written, commas and all.

## An empty name

A leading comma leaves the name empty, the same as in `json:",omitempty"`.

```go
tag := structwalker.ParseTag("json", ",omitempty")
tag.Name                 // ""
tag.HasFlag("omitempty") // true
```

## Commas inside a value

Wrap a name, flag or option value in single quotes to keep a comma in it.

```go
Role string `validate:"role,oneof='admin,editor',required"`
```

```go
tag.Option("oneof")      // "admin,editor", true
tag.HasFlag("required")  // true
```

The quotes are removed. A quote only counts at the start of an item or straight after `=`, so an apostrophe in the middle of a value is left alone:

```go
structwalker.ParseTag("help", "name,msg=it's fine").Option("msg") // "it's fine", true
```

## Opting a field out

`-` as the name means "not this field".

```go
Password string `mask:"-"`
```

`tag.IsIgnored()` reports it. [Tag handlers](/docs/handlers/) are not called for an ignored tag.

## Reading other tags

A handler gets the tag it was registered for in `ctx.Tag`. To read a different one, use `Lookup`:

```go
w.Handle("validate", func(ctx structwalker.FieldContext) error {
    name := ctx.Field.Name
    if tag, ok := ctx.Lookup("json"); ok {
        name = tag.Name
    }
    return fmt.Errorf("%s is required", name)
})
```

`ctx.AllTags` has every tag on the field in the order they're written.

## Parsing without a walker

```go
sf, _ := reflect.TypeFor[User]().FieldByName("Email")

for _, tag := range structwalker.ParseAllTags(sf.Tag) {
    fmt.Println(tag.Key, tag.Name)
}
```

`ParseAllTags` reads a struct tag the same way the standard library does, including escaped characters. A fuzz test in the repository holds it to that.

## A different syntax

If your tags don't follow this format you can supply your own parser. See [custom tag parsers](/docs/tag-parsers/).
