# structwalker

[![CI](https://github.com/rjp2525/structwalker/actions/workflows/ci.yml/badge.svg)](https://github.com/rjp2525/structwalker/actions/workflows/ci.yml)
[![Go Reference](https://pkg.go.dev/badge/github.com/rjp2525/structwalker.svg)](https://pkg.go.dev/github.com/rjp2525/structwalker)
[![Go Report Card](https://goreportcard.com/badge/github.com/rjp2525/structwalker)](https://goreportcard.com/report/github.com/rjp2525/structwalker)

Give your struct tags a job.

Every Go project ends up with the same reflection loop. Range over the fields, read a tag, recurse into nested structs, remember the path for the error message. You write it for config defaults. Then again for validation. Then again to redact secrets from logs.

structwalker is that loop, written once. You say what a tag means and it handles the walking.

```go
type Config struct {
    Port    int           `default:"8080"`
    Timeout time.Duration `default:"30s"`
    DB      struct {
        Host string `default:"localhost"`
    }
}

w := structwalker.New()
w.Handle("default", func(ctx structwalker.FieldContext) error {
    if ctx.IsZero() {
        return ctx.SetValue(ctx.Tag.Raw)
    }
    return nil
})

var cfg Config
err := w.Walk(&cfg)
// cfg.Port == 8080, cfg.Timeout == 30*time.Second, cfg.DB.Host == "localhost"
```

That is a working `default` tag in six lines. No dependencies, standard library only.

**[Read the docs at structwalker.reno.sh](https://structwalker.reno.sh)**

## Install

```bash
go get github.com/rjp2525/structwalker
```

Requires Go 1.26 or newer.

## Two ways to use it

### Handlers

Register a handler per tag key. One walk runs all of them, so validation, defaults and redaction can share a pass over the struct.

```go
w := structwalker.New(structwalker.WithErrorMode(structwalker.CollectErrors))

w.Handle("validate", func(ctx structwalker.FieldContext) error {
    if ctx.Tag.Name == "required" && ctx.IsZero() {
        return errors.New("is required")
    }
    return nil
})

err := w.Walk(&signup)
// structwalker: 2 errors:
//   - Email (tag "validate"): is required
//   - Address.City (tag "validate"): is required
```

### A plain loop

For a one-off there is nothing to register. `Fields` is an iterator.

```go
for f := range structwalker.Fields(&cfg) {
    if tag, ok := f.Lookup("env"); ok {
        if err := f.SetValue(os.Getenv(tag.Name)); err != nil {
            return err
        }
    }
}
```

## What you get

- **Tags, parsed.** `validate:"email,required,max=255"` arrives as a name, flags and options. Single quotes keep a comma inside a value: `oneof='admin,editor'`.
- **The whole struct.** Nested structs and pointers by default. Slices, maps and interface values when you ask for them. Cycles are detected.
- **Paths.** Every field knows where it is: `Address.Street`, `Items[2].Name`, `Limits[eu].Max`.
- **A setter that converts.** `SetValue("30s")` fills a `time.Duration`. It also handles numbers, bools, pointer fields and anything that implements `encoding.TextUnmarshaler`. It never stores a value that didn't fit.
- **Control over the walk.** Return `ErrSkipField` or `ErrSkipChildren` from a handler. Order handlers with priorities. Run them before or after a field's children.
- **Errors with context.** Stop at the first error, collect them all or ignore them. Each one carries the field's path.
- **Middleware.** Recovery, logging and timing are included. Writing your own takes a few lines.
- **Speed.** Reflection work is cached per type. A flat struct walks in about 200ns with zero allocations.
- **Safe to share.** Build a walker once and use it from as many goroutines as you like.

## Things people build with it

- Config loading from environment variables, files and defaults
- Validation with every error reported at once
- Redacting or masking sensitive fields before logging
- Sanitising and normalising input
- Mapping structs to database columns, CLI flags or API schemas
- Audit logs and diffs

The [docs](https://structwalker.reno.sh) have a worked recipe for each of these.

## Benchmarks

Measured on an Apple M5 Pro with Go 1.27. The structs are in [`benchmarks_test.go`](benchmarks_test.go).

| Benchmark | Time | Allocations |
|---|---|---|
| Five-field struct, one handler | 203 ns | 0 |
| Five-field struct, two handlers | 221 ns | 0 |
| Five-field struct, handler plus middleware | 268 ns | 0 |
| Two nested structs, eleven fields | 768 ns | 10 |
| `Fields` loop over five fields | 147 ns | 3 |
| Hand-written reflection loop, for scale | 109 ns | 0 |

Run them yourself:

```bash
go test -bench=. -benchmem
```

## Upgrading

[CHANGELOG.md](CHANGELOG.md) lists what changed in each release and anything you need to do about it.

## License

[MIT](LICENSE)
