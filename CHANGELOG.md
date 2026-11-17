# Changelog

Every notable change to structwalker is listed here. Versions follow [semantic versioning](https://semver.org).

## Unreleased

### Changed

- The code now uses `reflect.Pointer` and `reflect.TypeFor` in place of their older spellings. Nothing changes for callers. ([#1](https://github.com/rjp2525/structwalker/pull/1))
- Handler priority now orders every handler on a field. A `json` handler at priority 10 runs before a `validate` handler at the default 100, and `HandleAll` handlers take part too. Handlers that tie run in the order their tags are written on the field, then `HandleAll` handlers, then registration order. If you only ever used priorities within one tag key, nothing changes. ([#3](https://github.com/rjp2525/structwalker/pull/3))
- Walks are much cheaper. Handlers are sorted and wrapped in middleware once per struct type instead of on every field. A flat six-field struct went from 844ns and 10 allocations to 227ns and 1 on an M-series laptop. ([#3](https://github.com/rjp2525/structwalker/pull/3))

### Fixed

- A `Walker` is now safe to use from several goroutines from the first walk. Two walks starting at the same time used to race on an internal cache. You can also call `Handle`, `HandleAll` and `Use` while walks are running. ([#3](https://github.com/rjp2525/structwalker/pull/3))
- `ErrSkipChildren` only skips the field's children. It used to stop the other tag handlers on the same field as well. ([#3](https://github.com/rjp2525/structwalker/pull/3))
- Handlers with the same priority always run in a predictable order. ([#3](https://github.com/rjp2525/structwalker/pull/3))
- A field that repeats a tag key no longer runs its handlers twice. The first value wins, the same as `reflect.StructTag.Get`. ([#3](https://github.com/rjp2525/structwalker/pull/3))
- `Use` covers handlers registered before it was called, whatever order you set things up in. ([#3](https://github.com/rjp2525/structwalker/pull/3))

### Project

- CI now tests on Go 1.26 and 1.27 and fails if `go fix` has anything left to do. ([#2](https://github.com/rjp2525/structwalker/pull/2))
- Dependabot keeps the GitHub Actions up to date. ([#2](https://github.com/rjp2525/structwalker/pull/2))
- Release notes are taken from this file, so a tag without a changelog section fails the release. ([#2](https://github.com/rjp2525/structwalker/pull/2))

## v1.0.0

First release.
