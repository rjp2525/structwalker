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

### Changed

- The code now uses `reflect.Pointer` and `reflect.TypeFor` in place of their older spellings. Nothing changes for callers. ([#1](https://github.com/rjp2525/structwalker/pull/1))
- Handler priority now orders every handler on a field. A `json` handler at priority 10 runs before a `validate` handler at the default 100, and `HandleAll` handlers take part too. Handlers that tie run in the order their tags are written on the field, then `HandleAll` handlers, then registration order. If you only ever used priorities within one tag key, nothing changes. ([#3](https://github.com/rjp2525/structwalker/pull/3))
- Walks are much cheaper. Handlers are sorted and wrapped in middleware once per struct type instead of on every field. A flat six-field struct went from 844ns and 10 allocations to 227ns and 1 on an M-series laptop. ([#3](https://github.com/rjp2525/structwalker/pull/3))
- `MultiError.Error()` no longer ends with a newline and no longer repeats the `structwalker:` prefix on every line. ([#4](https://github.com/rjp2525/structwalker/pull/4))
- A recovered panic reads `structwalker: Name (tag "x"): panic: boom`. The path and tag used to appear twice. ([#4](https://github.com/rjp2525/structwalker/pull/4))
- `WithSliceElements` and `WithMapEntries` reach structs at any nesting: `[][]Item`, `map[string][]*Item`, `*[]Item`, `**Item` and arrays all work. Only one level of slice or map was walked before. ([#5](https://github.com/rjp2525/structwalker/pull/5))
- Map entries are visited in key order, so paths, handler calls and collected errors come out the same on every run. ([#5](https://github.com/rjp2525/structwalker/pull/5))
- A flat struct now walks with zero allocations. ([#5](https://github.com/rjp2525/structwalker/pull/5))

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

### Deprecated

- `WithContext`. It still works, but a context stored on the walker is shared by every walk and a single expired deadline ends them all. Use `WalkContext`. ([#6](https://github.com/rjp2525/structwalker/pull/6))

### Project

- CI now tests on Go 1.26 and 1.27 and fails if `go fix` has anything left to do. ([#2](https://github.com/rjp2525/structwalker/pull/2))
- Dependabot keeps the GitHub Actions up to date. ([#2](https://github.com/rjp2525/structwalker/pull/2))
- Release notes are taken from this file, so a tag without a changelog section fails the release. ([#2](https://github.com/rjp2525/structwalker/pull/2))

## v1.0.0

First release.
