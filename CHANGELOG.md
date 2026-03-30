# Changelog

All notable changes to this project will be documented in this file.

The format follows [Keep a Changelog](https://keepachangelog.com/en/1.0.0/).
This project adheres to [Semantic Versioning](https://semver.org/spec/v2.0.0.html).

## [Unreleased] — v0.3.0

### Added
- Compile-time interface assertions (`var _ Routes = (*RouteGroup)(nil)`) in every generated file — a breaking change to `Routes` or `Router` now produces a compile error immediately rather than silently failing at call sites.
- `Routes` and `Router` interfaces are documented as stable; new methods will not be added without a major version bump.
- Graceful shutdown in example `main.go` files using `http.Server.Shutdown` with SIGINT/SIGTERM handling and a 30-second drain timeout.
- Build-tag gated `maxAllocsPerRequest` constant (`race_enabled_test.go` / `race_disabled_test.go`) so the allocation test passes under both `go test` and `go test -race`.
- Concurrent correctness test suite (`tests/runtime/concurrent_test.go`) covering: route registration races, `Use()` while serving, deep group nesting, thundering-herd after `Use()`, and version-bump atomicity.
- Heap stability test (T32), goroutine leak test (T33), per-request allocation test (T34), and generator state isolation test (T35) in `tests/performance/`.

### Fixed
- **Data race** in `RouteGroup.HandleFunc`: `g.routes = append(...)` is now protected by `g.mu`.
- **Data race** in `RouteGroup.Use`: `version.Add(1)` is now called inside the write lock, so the version increment is visible atomically with the middleware slice update.
- **Data race** in `Generator.Generate`: options are parsed into a local `*Options` per call instead of writing to `g.Options`; `applyOptions()` removed.
- `getOutputFilename` and related helpers now accept `*Options` as a parameter instead of reading from shared generator state.

### Changed
- Register functions are now package-level (`pb.RegisterTaskServiceRoutes(r, handler) error`) rather than methods on `*RouteGroup`. **Breaking for callers of the old method-style API.**
- `MustRegisterTaskServiceRoutes` / `MustRegisterProductServiceRoutes` etc. added as panic-on-error convenience wrappers.
- Example `main.go` files updated to use the new package-level API and proper graceful shutdown.

---

## [v0.2.0]

### Added
- Shared `http.ServeMux` support — multiple `RouteGroup` instances can register routes on the same mux.
- `RouteGroup.Group(prefix, middlewares...)` for sub-routing with path prefixes.
- `ResponseWriterWrapper` with `http.Flusher` forwarding for SSE-safe middleware.
- `ErrNilRouter` / `ErrNilHandler` sentinel errors.

### Changed
- Router interface redesigned: `Routes` (minimal, for custom routers) and `Router` (full, with `Group`/`Use`).

---

## [v0.1.0]

### Added
- Plugin options (`paths=source_relative`, `output_prefix`, `editions`).
- Proto editions support.
- `extractPathParams` / `convertPathPattern` for `{param}` → `:param` translation.

---

## [v0.0.4] and earlier

- Initial implementation: code generator, template engine, `http.ServeMux`-based router, middleware support.
