---
title: Performance and concurrency
description: What a walk costs, what's cached, where the allocations come from and how to share a walker between goroutines.
---

A walk is fast enough that you can usually stop thinking about it. Here's what it actually costs so you can decide for yourself.

## Numbers

Measured on an Apple M5 Pro with Go 1.27.

| | Time | Allocations |
|---|---|---|
| Five-field struct, one handler | 203 ns | 0 |
| Five-field struct, two handlers | 221 ns | 0 |
| Five-field struct, handler plus middleware | 268 ns | 0 |
| Two nested structs, eleven fields | 768 ns | 10 |
| `Fields` loop over five fields | 147 ns | 3 |
| Hand-written reflection loop, for scale | 109 ns | 0 |

So a walk costs about twice what the bare reflection loop does. For that you get parsed tags, paths and handler dispatch. For validating a request body or loading config that's noise. For a hot loop over millions of structs it might not be.

Run them on your own hardware:

```bash
go test -bench=. -benchmem
```

## What's cached

The first time a walker meets a struct type it builds a plan for it:

- which fields to visit
- every tag on them, parsed
- which handlers run on each field, already sorted, split by phase and wrapped in middleware

Later walks of that type look the plan up and go. A lookup takes about 8ns. Nothing is parsed or sorted during a walk.

The cache is per walker. That's the reason to build a walker once and keep it, not make a new one per call.

## Where allocations come from

A flat struct allocates nothing. Three things do allocate:

- **Paths of nested fields.** `Address.Street` is a new string. Root-level paths are the field name and cost nothing.
- **Cycle tracking.** The first pointer, slice or map the walk enters allocates a small map.
- **Your handlers.** `ctx.Value.Interface()` on a non-pointer value allocates, as it does anywhere in Go.

## Keeping it quick

- **Reuse walkers.** Put them in package variables or build them at startup.
- **Register only what you use.** A field with no handlers is close to free.
- **Use `WithLeafTypes`** for struct types you never need to look inside.
- **Leave containers off** unless you need them. `WithSliceElements` on a struct with a ten-thousand-element slice walks ten thousand structs.
- **Prefer typed getters.** `ctx.Value.Int()` doesn't allocate. `ctx.Value.Interface()` can.

## Concurrency

A `Walker` is safe to use from any number of goroutines.

```go
var validator = buildValidator()

func handler(w http.ResponseWriter, r *http.Request) {
    // called concurrently, no locking needed
    err := validator.WalkContext(r.Context(), &req)
}
```

Walks don't take locks. Handlers and middleware live in an immutable snapshot and a walk reads the one that was current when it started.

You can call `Handle`, `HandleAll` and `Use` while walks are running. Each call swaps in a new snapshot. Walks already in flight finish with the old one and the per-type plans are rebuilt on next use.

### What's still yours to make safe

- **Your handlers.** If a handler appends to a shared slice, two concurrent walks will race on it. Keep per-walk state out of closures or guard it.
- **The struct being walked.** Two goroutines walking the same struct with handlers that set fields are two goroutines writing the same memory.
- **A custom tag parser.** It can be called from several goroutines at once.

### Per-walk state

Handlers are shared, so they can't hold one walk's results. Put that state in the context:

```go
type collected struct{ fields []string }

w.HandleAll(func(fc structwalker.FieldContext) error {
    c := fc.Context().Value(key{}).(*collected)
    c.fields = append(c.fields, fc.Path)
    return nil
})

c := &collected{}
err := w.WalkContext(context.WithValue(ctx, key{}, c), &v)
```

Or skip handlers and use a [`Fields`](/docs/fields/) loop, where the state is just local variables.
