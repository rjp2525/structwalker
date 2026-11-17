---
title: Limitations
description: What structwalker doesn't do, why not and what to do about it.
---

Better to read this page now than to discover it halfway through a feature.

## Not supported

### Handlers on slice elements and map entries

Handlers run on struct **fields**. A `[]string` or `map[string]int` field is one field. The walker goes into containers to find structs but it doesn't call handlers for each string or int inside.

Loop over `ctx.Value` in the handler if you need the elements.

### A handler for the struct itself

There's no call for the root struct, only for its fields. Cross-field checks on the root ("one of A or B must be set") go after the walk. For a nested struct, use an [after-children handler](/docs/handlers/#before-or-after-children) on the field that holds it.

### Map keys

Map values are walked. Keys aren't, even when they're structs.

### Channels and funcs

They're visited as fields and never entered.

### Typed handlers

Handlers work with `reflect.Value`. There's no generic `Handle[T]` that hands you a `*string`. `ctx.Value.Addr().Interface()` gets you a typed pointer when you want one.

## Partly supported

### Setting values inside maps

A struct stored directly in a map can be read but not set. Go's maps don't allow it. Use pointer values: `map[string]*Item`.

### Setting values behind interfaces

The same rule: an `any` holding a struct value is read-only and an `any` holding a pointer to a struct can be changed.

### Unexported fields

[`WithUnexported`](/docs/options/) lets handlers see them. Go's reflection doesn't let anyone set them. `Value.Interface()` panics on them.

### Nil embedded pointers

An embedded `*Base` that's nil isn't allocated for you. Its fields aren't visited until it points at something.

### Fields loops

A [`Fields`](/docs/fields/) loop can't skip a field's children and doesn't report errors. Use handlers when you need either.

### Splitting strings into slices

[`SetValue`](/docs/setting-values/) won't turn `"a,b,c"` into a `[]string`. The right separator is your call, so split it and pass the slice.

## By design

### Reaching the depth limit is an error

A walker that silently stopped at 32 levels would be a validator that silently stopped validating. Raise [`WithMaxDepth`](/docs/traversal/#depth) if your data really is that deep.

### Options are fixed

A walker's options can't change after `New`. Handlers and middleware can be added at any time.

### It's reflection

Cached and quick, but still reflection. If [the numbers](/docs/performance/) aren't good enough for a hot path, generate code for that path.

### One tag format

The built-in parser reads `name,flag,key=value`. Anything else needs a [custom parser](/docs/tag-parsers/).

## Missing something?

[Open an issue](https://github.com/rjp2525/structwalker/issues) and say what you're trying to build. Several things on this page are here because nobody has needed them yet, not because they're hard.
