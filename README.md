# structwalker

[![CI](https://github.com/rjp2525/structwalker/actions/workflows/ci.yml/badge.svg)](https://github.com/rjp2525/structwalker/actions/workflows/ci.yml)
[![Go Reference](https://pkg.go.dev/badge/github.com/rjp2525/structwalker.svg)](https://pkg.go.dev/github.com/rjp2525/structwalker)
[![Go Report Card](https://goreportcard.com/badge/github.com/rjp2525/structwalker)](https://goreportcard.com/report/github.com/rjp2525/structwalker)

A Go library for recursively walking structs, parsing tags, and dispatching tag-based callbacks with built-in reflection caching.

Register tag handlers once. Walk any struct. Each field's tags are parsed, matched to registered handlers, and dispatched with full context: parsed options, field value, path, and parent. Recursive, cached, zero-alloc on cache hits.

Zero dependencies. stdlib only.

## Install

```bash
go get github.com/rjp2525/structwalker
```

Requires Go 1.26+.

## Quick Start

```go
package main

import (
    "fmt"

    "github.com/rjp2525/structwalker"
)

type User struct {
    Name  string `json:"name" validate:"required"`
    Email string `json:"email" validate:"email"`
    Age   int    `json:"age" validate:"min=0"`
}

func main() {
    w := structwalker.New()

    w.Handle("json", func(ctx structwalker.FieldContext) error {
        fmt.Printf("json: %s -> %s\n", ctx.Path, ctx.Tag.Name)
        return nil
    })

    w.Handle("validate", func(ctx structwalker.FieldContext) error {
        fmt.Printf("validate: %s rule=%s\n", ctx.Path, ctx.Tag.Name)
        return nil
    })

    user := &User{Name: "Alice", Email: "alice@example.com", Age: 30}
    if err := w.Walk(user); err != nil {
        panic(err)
    }
}
```

Output:

```
json: Name -> name
validate: Name rule=required
json: Email -> email
validate: Email rule=email
json: Age -> age
validate: Age rule=min=0
```

---

## Tag Format

Tags follow the standard Go convention: `key:"name,flag1,flag2,opt=value"`.

- First element is the name (field alias)
- Bare identifiers are flags (`omitempty`, `required`)
- `key=value` pairs are options (`max=255`, `format=date`)
- `-` as the name ignores a field
- `!flag` negates a flag (`!omitempty`)

```go
tag := structwalker.ParseTag("validate", "email,required,max=255")
tag.Key                     // "validate"
tag.Name                    // "email"
tag.Raw                     // "email,required,max=255"
tag.HasFlag("required")     // true
tag.HasFlag("optional")     // false
tag.Option("max")           // "255", true
tag.Option("min")           // "", false
tag.OptionOr("min", "0")    // "0"
tag.IsIgnored()             // false
```

### Parsing a Full Struct Tag

`ParseAllTags` extracts every tag from a `reflect.StructTag`:

```go
type User struct {
    Name string `json:"name,omitempty" validate:"required,max=100" db:"user_name"`
}

sf, _ := reflect.TypeOf(User{}).FieldByName("Name")
tags := structwalker.ParseAllTags(sf.Tag)
// tags[0] = Tag{Key: "json", Name: "name", Flags: ["omitempty"]}
// tags[1] = Tag{Key: "validate", Name: "required", Options: {"max": "100"}}
// tags[2] = Tag{Key: "db", Name: "user_name"}
```

### Negation Flags

Prefix a flag with `!` to negate it. Negated flags are stored with the `!` prefix:

```go
tag := structwalker.ParseTag("validate", "field,!omitempty")
tag.HasFlag("!omitempty")   // true
tag.HasFlag("omitempty")    // false
```

---

## Handlers

Handlers are callbacks registered for specific tag keys. When walking a struct, each field's tags are matched against registered handlers and dispatched in priority order.

### Tag-Specific Handlers

```go
w := structwalker.New()

w.Handle("mask", func(ctx structwalker.FieldContext) error {
    if ctx.Tag.HasFlag("redact") {
        return ctx.SetValue("***REDACTED***")
    }
    return nil
})

type Response struct {
    UserID int    `json:"user_id"`
    Email  string `json:"email" mask:",redact"`
    SSN    string `json:"ssn" mask:",redact"`
}

resp := &Response{UserID: 1, Email: "alice@example.com", SSN: "123-45-6789"}
w.Walk(resp)
// resp.Email == "***REDACTED***"
// resp.SSN == "***REDACTED***"
```

### Global Handlers

`HandleAll` registers a handler invoked for every field, regardless of tags:

```go
w.HandleAll(func(ctx structwalker.FieldContext) error {
    fmt.Printf("[audit] %s = %v\n", ctx.Path, ctx.Value.Interface())
    return nil
})
```

### Handler Options

#### Priority

Multiple handlers for the same tag key execute in priority order (lower runs first). Default priority is 100.

```go
w.Handle("validate", validateRequired, structwalker.WithPriority(10))  // runs first
w.Handle("validate", validateFormat, structwalker.WithPriority(50))    // runs second
w.Handle("validate", logValidation, structwalker.WithPriority(200))    // runs last
```

#### Filter

Only invoke the handler when a predicate returns true:

```go
// Only validate root-level fields
w.Handle("validate", handler, structwalker.WithFilter(func(ctx structwalker.FieldContext) bool {
    return ctx.Depth == 0
}))

// Only handle string fields
w.Handle("mask", handler, structwalker.WithFilter(func(ctx structwalker.FieldContext) bool {
    return ctx.Value.Kind() == reflect.String
}))
```

#### Phase

Handlers run before recursing into child fields by default. Use `AfterChildren` to run after:

```go
// Runs before walking nested struct fields
w.Handle("tag", beforeHandler)

// Runs after walking nested struct fields
w.Handle("tag", afterHandler, structwalker.WithPhase(structwalker.AfterChildren))
```

Execution order for a nested struct:

```
before handler: Parent
  before handler: Parent.Child
  after handler:  Parent.Child
after handler:  Parent
```

---

## Walker Options

All options are set at walker creation time:

```go
w := structwalker.New(
    structwalker.WithMaxDepth(10),
    structwalker.WithUnexported(true),
    structwalker.WithFollowPointers(false),
    structwalker.WithSliceElements(true),
    structwalker.WithMapEntries(true),
    structwalker.WithErrorMode(structwalker.CollectErrors),
    structwalker.WithContext(ctx),
    structwalker.WithTagParser(myCustomParser),
)
```

### Option Reference

| Option | Default | Description |
|--------|---------|-------------|
| `WithMaxDepth(n)` | `32` | Maximum recursion depth before returning an error |
| `WithUnexported(bool)` | `false` | Include unexported fields (read-only, not settable) |
| `WithFollowPointers(bool)` | `true` | Dereference pointers during walk |
| `WithSliceElements(bool)` | `false` | Walk into individual slice/array elements |
| `WithMapEntries(bool)` | `false` | Walk into map values |
| `WithErrorMode(mode)` | `StopOnError` | Error handling strategy (see below) |
| `WithContext(ctx)` | `context.Background()` | Context for cancellation support |
| `WithTagParser(fn)` | built-in CSV | Override the tag parsing function |

### Error Modes

| Mode | Behavior |
|------|----------|
| `StopOnError` | First handler error halts the walk (default) |
| `CollectErrors` | Collect all errors, return as `*MultiError` |
| `SkipOnError` | Ignore handler errors, continue walking |

```go
// Collect all validation errors at once
w := structwalker.New(structwalker.WithErrorMode(structwalker.CollectErrors))
w.Handle("validate", func(ctx structwalker.FieldContext) error {
    if ctx.Tag.Name == "required" && ctx.IsZero() {
        return fmt.Errorf("field is required")
    }
    return nil
})

err := w.Walk(&myStruct)
if err != nil {
    var multi *structwalker.MultiError
    if errors.As(err, &multi) {
        for _, e := range multi.Errors {
            fmt.Printf("%s: %s\n", e.Path, e.Err)
        }
    }
}
```

### Custom Tag Parser

Override the default CSV-based tag parser for alternative formats:

```go
w := structwalker.New(structwalker.WithTagParser(func(key, raw string) structwalker.Tag {
    // Custom parsing logic
    return structwalker.Tag{
        Key:  key,
        Name: raw,
        Raw:  raw,
    }
}))
```

---

## Flow Control

Handlers can return sentinel errors to control walk behavior.

### SkipField

Stop processing remaining handlers on the current field and move to the next:

```go
w.Handle("json", func(ctx structwalker.FieldContext) error {
    if ctx.Tag.IsIgnored() {
        return structwalker.SkipField
    }
    // process field...
    return nil
})
```

### SkipChildren

Process the current field's handlers but skip recursion into its nested struct fields:

```go
w.Handle("audit", func(ctx structwalker.FieldContext) error {
    if ctx.Tag.Name == "skip" {
        return structwalker.SkipChildren
    }
    return nil
})

type Config struct {
    Name    string  `json:"name"`
    Secrets Secrets `json:"secrets" audit:"skip"` // won't recurse into Secrets
}
```

---

## FieldContext

Every handler receives a `FieldContext` with full field information.

### Fields

| Field | Type | Description |
|-------|------|-------------|
| `Tag` | `Tag` | Parsed tag for this handler's registered key |
| `AllTags` | `[]Tag` | Every parsed tag on the field |
| `Field` | `reflect.StructField` | Reflection field metadata |
| `Value` | `reflect.Value` | Field value (settable if walker received a pointer) |
| `Path` | `string` | Dot-separated path from root: `"Address.Street"`, `"Items[2].Name"` |
| `Depth` | `int` | Nesting level (0 = root struct fields) |
| `Parent` | `reflect.Value` | The containing struct's `reflect.Value` |
| `Index` | `int` | Field index within the parent struct |
| `Walker` | `*Walker` | Reference to the walker instance |

### SetValue

Sets a field's value with type coercion. The walker must receive a pointer for fields to be settable.

Supported coercions from `string`:
- `string` to `int`, `int8`, `int16`, `int32`, `int64`
- `string` to `uint`, `uint8`, `uint16`, `uint32`, `uint64`
- `string` to `float32`, `float64`
- `string` to `bool`
- `string` to `time.Time` (RFC 3339 format)

```go
w.Handle("default", func(ctx structwalker.FieldContext) error {
    if ctx.IsZero() {
        return ctx.SetValue(ctx.Tag.Name) // coerces "42" to int, "true" to bool, etc.
    }
    return nil
})

type Config struct {
    Port    int    `default:"8080"`
    Debug   bool   `default:"false"`
    Timeout string `default:"30s"`
}
```

Setting to `nil` resets the field to its zero value:

```go
ctx.SetValue(nil) // resets to zero value
```

### Helper Methods

```go
ctx.IsZero() bool      // true if the field is the zero value for its type
ctx.IsNil() bool       // true if the field is a nil pointer, slice, map, or interface
ctx.IsExported() bool  // true if the field is exported
```

---

## Walking Nested Structures

### Nested Structs

Struct fields are recursed into automatically. The `Path` reflects the nesting:

```go
type Address struct {
    Street string `json:"street"`
    City   string `json:"city"`
}
type User struct {
    Name    string  `json:"name"`
    Address Address `json:"address"`
}

// Handler receives paths: "Name", "Address", "Address.Street", "Address.City"
```

### Pointers

Pointers to structs are dereferenced automatically (configurable with `WithFollowPointers`). Nil pointers are skipped. Circular references are detected and broken.

```go
type Node struct {
    Name string `json:"name"`
    Next *Node  `json:"next"`
}

a := &Node{Name: "a", Next: &Node{Name: "b"}}
a.Next.Next = a // circular reference, handled safely
```

### Slices

Enable with `WithSliceElements(true)`. Each element gets a path like `Items[0].Name`:

```go
type Item struct {
    Name string `tag:"name"`
}
type Cart struct {
    Items []Item `tag:"items"`
}

w := structwalker.New(structwalker.WithSliceElements(true))
// Paths: "Items", "Items[0].Name", "Items[1].Name"
```

Nil elements in pointer slices are skipped.

### Maps

Enable with `WithMapEntries(true)`. Map keys appear in the path:

```go
type Config struct {
    Settings map[string]Setting `tag:"settings"`
}

w := structwalker.New(structwalker.WithMapEntries(true))
// Paths: "Settings", "Settings[database].Host", "Settings[database].Port"
```

---

## Middleware

Middleware wraps every handler invocation for logging, timing, error recovery, or custom logic.

```go
w := structwalker.New()
w.Use(myMiddleware1, myMiddleware2)
```

Middleware executes in order, wrapping the handler from outside in:

```
middleware1 > middleware2 > handler > middleware2 > middleware1
```

### Built-in Middleware

`WithRecovery` catches panics in handlers and converts them to errors:

```go
w.Use(structwalker.WithRecovery())
```

`WithLogging` logs every field visit at debug level using `log/slog`:

```go
w.Use(structwalker.WithLogging(slog.Default()))
```

`WithTiming` records handler execution time via a callback:

```go
w.Use(structwalker.WithTiming(func(path, tagKey string, d time.Duration) {
    metrics.RecordHistogram("structwalker.handler.duration", d,
        "path", path,
        "tag", tagKey,
    )
}))
```

### Custom Middleware

```go
func RequireExported() structwalker.Middleware {
    return func(ctx structwalker.FieldContext, next structwalker.HandlerFunc) error {
        if !ctx.IsExported() {
            return nil // skip unexported fields
        }
        return next(ctx)
    }
}

w.Use(RequireExported())
```

---

## Context Cancellation

Pass a context to cancel long-running walks:

```go
ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
defer cancel()

w := structwalker.New(structwalker.WithContext(ctx))
err := w.Walk(&largeStruct)
if errors.Is(err, context.DeadlineExceeded) {
    // walk timed out
}
```

The context is checked between each field, so cancellation responds quickly even on large structs.

---

## Error Handling

Handler errors are wrapped in `*WalkError` with field path context:

```go
type WalkError struct {
    Path   string // "Config.Database.Host"
    TagKey string // "validate"
    Field  string // "Host"
    Err    error  // underlying error
}
```

`WalkError` supports `errors.Is` and `errors.As` through `Unwrap`:

```go
err := w.Walk(&myStruct)

var walkErr *structwalker.WalkError
if errors.As(err, &walkErr) {
    fmt.Printf("field %s failed: %v\n", walkErr.Path, walkErr.Err)
}
```

With `CollectErrors` mode, `*MultiError` also supports `errors.Is` across all collected errors:

```go
var multi *structwalker.MultiError
if errors.As(err, &multi) {
    for _, e := range multi.Errors {
        fmt.Printf("  %s: %v\n", e.Path, e.Err)
    }
}
```

---

## Reflection Caching

Type metadata (field info, parsed tags, handler key matching) is parsed once per type on first walk and cached with `sync.Map`. Later walks of the same struct type skip parsing entirely.

- Cache hits: ~5ns, zero allocations
- Safe for concurrent use
- Invalidated automatically when new handlers are registered with `Handle`

---

## Concurrency

A `Walker` is safe for concurrent use after handler registration is complete. The reflection cache uses `sync.Map`. Registering new handlers (`Handle`, `HandleAll`) during concurrent walks invalidates the cache and may cause brief re-parsing overhead.

```go
// Set up once
w := structwalker.New()
w.Handle("json", jsonHandler)
w.Handle("validate", validateHandler)
w.Use(structwalker.WithRecovery())

// Use concurrently
go func() { w.Walk(&struct1) }()
go func() { w.Walk(&struct2) }()
```

---

## Benchmarks

```
cpu: Apple M4 Pro
BenchmarkWalk_Flat-14              1000000     1185 ns/op     360 B/op    13 allocs/op
BenchmarkWalk_Nested-14             380620     3226 ns/op     960 B/op    39 allocs/op
BenchmarkWalk_Deep-14              1000000     1072 ns/op     376 B/op    14 allocs/op
BenchmarkWalk_MultipleHandlers-14   819598     1508 ns/op     528 B/op    19 allocs/op
BenchmarkWalk_WithMiddleware-14     895304     1331 ns/op     480 B/op    18 allocs/op
BenchmarkWalk_HandleAll-14         1427517      837 ns/op     360 B/op    13 allocs/op
BenchmarkParseTag-14               8229410      147 ns/op     416 B/op     4 allocs/op
BenchmarkParseAllTags-14           3281874      378 ns/op    1088 B/op    11 allocs/op
BenchmarkRawReflection-14        21606843       56 ns/op       0 B/op     0 allocs/op
BenchmarkTypeCache_Hit-14       216546840        5 ns/op       0 B/op     0 allocs/op
```

Run benchmarks yourself:

```bash
go test -bench=. -benchmem ./...
```

---

## License

[MIT](LICENSE)
