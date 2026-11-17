# Changelog

Every notable change to structwalker is listed here. Versions follow [semantic versioning](https://semver.org).

## Unreleased

### Added

- `ErrMaxDepth`. Hitting the depth limit now returns a `*WalkError` you can test with `errors.Is(err, structwalker.ErrMaxDepth)`, with the path where the walk stopped descending. ([#4](https://github.com/rjp2525/structwalker/pull/4))
- `PanicError`. `WithRecovery` returns one with the panic value and the stack, so you can log where a handler blew up. If the handler panicked with an error, `errors.Is` and `errors.As` see through to it. ([#4](https://github.com/rjp2525/structwalker/pull/4))
- `WithInterfaces(true)` lets the walker look inside interface fields and visit the struct or pointer an `any` is holding. ([#5](https://github.com/rjp2525/structwalker/pull/5))
- `WithLeafTypes` names struct types the walker should treat as plain values, such as a decimal or a UUID type. Handlers still run on the field. `time.Time` is a leaf out of the box. ([#5](https://github.com/rjp2525/structwalker/pull/5))
- `WalkContext` and `WalkValueContext` take a context per walk. One walker can serve many requests, each with its own deadline. ([#6](https://github.com/rjp2525/structwalker/pull/6))
- `FieldContext.Context()` gives handlers the walk's context, so a handler that calls a database or an API can pass it along. ([#6](https://github.com/rjp2525/structwalker/pull/6))
- `SetValue` parses a string into a `time.Duration` (`default:"30s"` now works on a duration field), a `[]byte` and any type that implements `encoding.TextUnmarshaler`. That covers `time.Time`, `netip.Addr`, `big.Int` and most UUID and decimal types without structwalker knowing about them. ([#7](https://github.com/rjp2525/structwalker/pull/7))
- `SetValue` fills pointer fields. Setting `"8080"` on a `*int` allocates the int and points the field at it. ([#7](https://github.com/rjp2525/structwalker/pull/7))
- `ErrNotSettable`, returned by `SetValue` when the field cannot be changed, so a handler can tell that apart from a bad value. ([#7](https://github.com/rjp2525/structwalker/pull/7))
- Single quotes keep a comma inside a tag value: `validate:"role,oneof='admin,editor'"` gives the option `oneof` the value `admin,editor`. The quotes are stripped. An apostrophe in the middle of a value is left alone. ([#8](https://github.com/rjp2525/structwalker/pull/8))
- `FieldContext.Lookup(key)` returns another tag on the same field. A `validate` handler can use it to report the field's `json` name. ([#8](https://github.com/rjp2525/structwalker/pull/8))
- `Fields`, an iterator over a struct's fields for when you just want a loop. `for f := range structwalker.Fields(&cfg)` visits everything `Walk` would with the same `FieldContext`. `break` stops it. No walker, handlers or error plumbing needed. `Walker.Fields` does the same with a walker's options. ([#9](https://github.com/rjp2525/structwalker/pull/9))
- A documentation site at [structwalker.reno.sh](https://structwalker.reno.sh) with guides for every feature and six worked recipes: defaults, config from the environment, validation, redaction, struct mapping and diffs. ([#12](https://github.com/rjp2525/structwalker/pull/12))

### Changed

- The code now uses `reflect.Pointer` and `reflect.TypeFor` in place of their older spellings. Nothing changes for callers. ([#1](https://github.com/rjp2525/structwalker/pull/1))
- Handler priority now orders every handler on a field. A `json` handler at priority 10 runs before a `validate` handler at the default 100. `HandleAll` handlers take part too. Handlers that tie run in the order their tags are written on the field, then `HandleAll` handlers, then registration order. If you only ever used priorities within one tag key, nothing changes. ([#3](https://github.com/rjp2525/structwalker/pull/3))
- Walks are much cheaper. Handlers are sorted and wrapped in middleware once per struct type instead of on every field. A flat five-field struct went from 844ns and 10 allocations to 227ns and 1 on an M-series laptop. ([#3](https://github.com/rjp2525/structwalker/pull/3))
- `MultiError.Error()` no longer ends with a newline and no longer repeats the `structwalker:` prefix on every line. ([#4](https://github.com/rjp2525/structwalker/pull/4))
- A recovered panic reads `structwalker: Name (tag "x"): panic: boom`. The path and tag used to appear twice. ([#4](https://github.com/rjp2525/structwalker/pull/4))
- `WithSliceElements` and `WithMapEntries` reach structs at any nesting: `[][]Item`, `map[string][]*Item`, `*[]Item`, `**Item` and arrays all work. Only one level of slice or map was walked before. ([#5](https://github.com/rjp2525/structwalker/pull/5))
- Map entries are visited in key order, so paths, handler calls and collected errors come out the same on every run. ([#5](https://github.com/rjp2525/structwalker/pull/5))
- A flat struct now walks with zero allocations. ([#5](https://github.com/rjp2525/structwalker/pull/5))
- `SetValue` errors read `field Port: cannot parse "abc" as int: invalid syntax`. They name the field first and no longer include `strconv` internals. ([#7](https://github.com/rjp2525/structwalker/pull/7))
- Spaces around an option's key are trimmed, so `max = 5` sets `max`. ([#8](https://github.com/rjp2525/structwalker/pull/8))
- The package documentation is rewritten around what you would build with it. Every `FieldContext` field is documented. There are new runnable examples for defaults, `Lookup`, `WalkContext` and tag parsing. ([#10](https://github.com/rjp2525/structwalker/pull/10))
- The README is rewritten. It is shorter, leads with what the package is for and no longer documents `structwalker.SkipChildren`, a name that never existed. The reference material moves to the docs site. ([#11](https://github.com/rjp2525/structwalker/pull/11))

### Fixed

- A `Walker` is now safe to use from several goroutines from the first walk. Two walks starting at the same time used to race on an internal cache. You can also call `Handle`, `HandleAll` and `Use` while walks are running. ([#3](https://github.com/rjp2525/structwalker/pull/3))
- `ErrSkipChildren` only skips the field's children. It used to stop the other tag handlers on the same field as well. ([#3](https://github.com/rjp2525/structwalker/pull/3))
- Handlers with the same priority always run in a predictable order. ([#3](https://github.com/rjp2525/structwalker/pull/3))
- A field that repeats a tag key no longer runs its handlers twice. The first value wins, the same as `reflect.StructTag.Get`. ([#3](https://github.com/rjp2525/structwalker/pull/3))
- `Use` covers handlers registered before it was called, whatever order you set things up in. ([#3](https://github.com/rjp2525/structwalker/pull/3))
- `CollectErrors` returns one flat `MultiError` for the whole walk. Errors from nested structs used to arrive as a `MultiError` wrapped inside a `WalkError`, so ranging over `Errors` missed them. ([#4](https://github.com/rjp2525/structwalker/pull/4))
- `CollectErrors` keeps going after a failing slice element or map entry. It used to stop at the first one. ([#4](https://github.com/rjp2525/structwalker/pull/4))
- In `CollectErrors` mode a struct that is nested too deep is reported and the rest of the walk carries on. ([#4](https://github.com/rjp2525/structwalker/pull/4))
- `WithLogging` no longer logs `ErrSkipField` and `ErrSkipChildren` as handler errors. ([#4](https://github.com/rjp2525/structwalker/pull/4))
- A struct that points back to the value passed to `Walk` is no longer walked a second time before the cycle is noticed. ([#5](https://github.com/rjp2525/structwalker/pull/5))
- `WithFollowPointers(false)` is respected inside slices and maps. Pointer elements used to be followed regardless. ([#5](https://github.com/rjp2525/structwalker/pull/5))
- Exported fields of an unexported embedded struct are visited and can be set, the same way `encoding/json` treats them. They used to be skipped. ([#5](https://github.com/rjp2525/structwalker/pull/5))
- A pointer to a struct's first field is no longer mistaken for a cycle back to the struct. ([#5](https://github.com/rjp2525/structwalker/pull/5))
- Walking an unexported map field with `WithUnexported` and `WithMapEntries` no longer panics. ([#5](https://github.com/rjp2525/structwalker/pull/5))
- `SetValue(65)` on a string field returned no error and stored `"A"`. Numbers are no longer converted to strings. ([#7](https://github.com/rjp2525/structwalker/pull/7))
- `SetValue` refuses numbers that do not fit instead of wrapping them. `"300"` into an `int8`, `-1` into a `uint` and `1.5` into an `int` are all errors now and leave the field untouched. ([#7](https://github.com/rjp2525/structwalker/pull/7))
- Escaped characters in a tag value are unescaped. `desc:"say \"hi\""` used to come through with the backslashes still in it. Tag parsing now agrees with `reflect.StructTag.Lookup` on every input, which a fuzz test checks. ([#8](https://github.com/rjp2525/structwalker/pull/8))

### Deprecated

- `WithContext`. It still works, but a context stored on the walker is shared by every walk and a single expired deadline ends them all. Use `WalkContext`. ([#6](https://github.com/rjp2525/structwalker/pull/6))

### Project

- CI now tests on Go 1.26 and 1.27 and fails if `go fix` has anything left to do. ([#2](https://github.com/rjp2525/structwalker/pull/2))
- Dependabot keeps the GitHub Actions up to date. ([#2](https://github.com/rjp2525/structwalker/pull/2))
- Release notes are taken from this file, so a tag without a changelog section fails the release. ([#2](https://github.com/rjp2525/structwalker/pull/2))
- A fuzz test compares `ParseAllTags` with the standard library. ([#8](https://github.com/rjp2525/structwalker/pull/8))
- The recipes in the docs are runnable examples in `example_recipes_test.go`, so CI fails if one stops working. ([#12](https://github.com/rjp2525/structwalker/pull/12))
- The site is built by a small Go program in `website/`. It is its own module, so the library still has no dependencies. ([#12](https://github.com/rjp2525/structwalker/pull/12))

## v1.0.0

First release.
