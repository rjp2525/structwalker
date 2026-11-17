# Changelog

Every notable change to structwalker is listed here. Versions follow [semantic versioning](https://semver.org).

## Unreleased

### Changed

- The code now uses `reflect.Pointer` and `reflect.TypeFor` in place of their older spellings. Nothing changes for callers. ([#1](https://github.com/rjp2525/structwalker/pull/1))

### Project

- CI now tests on Go 1.26 and 1.27 and fails if `go fix` has anything left to do. ([#2](https://github.com/rjp2525/structwalker/pull/2))
- Dependabot keeps the GitHub Actions up to date. ([#2](https://github.com/rjp2525/structwalker/pull/2))
- Release notes are taken from this file, so a tag without a changelog section fails the release. ([#2](https://github.com/rjp2525/structwalker/pull/2))

## v1.0.0

First release.
