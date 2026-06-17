# CLAUDE.md

This file provides guidance to Claude Code (claude.ai/code) when working with code in this repository.

## What this is

A Traefik middleware plugin (`github.com/muench-dev/query-to-header`) that copies values from
incoming HTTP request query parameters into HTTP request headers before the request reaches
the next handler in the chain. Traefik plugins are not run as compiled binaries in production —
they are interpreted by [Yaegi](https://github.com/traefik/yaegi) inside Traefik itself, which
constrains the code to the Go standard library only (no third-party dependencies).

## Commands

```bash
go test -v -cover ./...   # run all tests with coverage
go test -run TestName ./... -v  # run a single test
go build ./...             # sanity-compile (plugin is never shipped as a binary)
go vet ./...
golangci-lint run          # lint (config in .golangci.yml)
just                        # default recipe: lists all available recipes
```

There is no `main` package and nothing is ever executed as a standalone program — `go build`
is only a compile-correctness check.

## Architecture

Everything lives in a single package, `traefik_query_to_header` (package name uses underscores
because the module path `query-to-header` is not a valid Go identifier):

- **`query_to_header.go`** — the entire plugin:
  - `Config` / `Mapping` — JSON-tagged structs populated by Traefik from the dynamic
    configuration (`mappings: [{query, header, remove, overwrite, bearer}]`).
  - `CreateConfig()` — required by the Traefik plugin contract; returns the default config.
  - `New(ctx, next, config, name)` — required factory signature Traefik calls to construct the
    middleware. Validates that every mapping has both `query` and `header` set, then returns a
    `*QueryToHeader` wrapping `next`.
  - `(*QueryToHeader).ServeHTTP` — the request path: for each configured mapping, reads the
    query parameter, optionally prefixes the value with `Bearer ` (`Bearer`), sets the header
    (skipping if already present unless `Overwrite` is true), optionally strips the query
    parameter (`Remove`), rewrites `req.URL.RawQuery`, then calls `next.ServeHTTP`.
- **`query_to_header_test.go`** — table-style tests built around `httptest.NewRequest` /
  `httptest.NewRecorder`, asserting on header/query state observed inside a stub `next` handler.
- **`.traefik.yml`** — the plugin manifest required by the Traefik Plugin Catalog
  (`displayName`, `type: middleware`, `import`, `summary`, `testData` used to validate the
  catalog example config matches `Config`'s JSON shape).

## Releasing

`.release-it.json` configures [`release-it`](https://github.com/release-it/release-it) (a
globally installed CLI, not a project dependency — there is no `package.json`) to only bump
the version, commit, tag, and push (`github.release: false`, `npm.publish: false`). Pushing the
resulting `vX.Y.Z` tag triggers `.github/workflows/release.yml`, which runs `goreleaser` to
actually publish the GitHub Release.

## Traefik Plugin Catalog requirements

The catalog (https://plugins.traefik.io) polls GitHub daily and imports repositories that meet
all of these criteria — breaking any of them silently stops catalog updates:

- Repository must not be a fork.
- The `traefik-plugin` topic must be set on the GitHub repository.
- `.traefik.yml` must exist at repo root with a valid `testData` property matching `Config`'s
  JSON shape.
- A valid `go.mod` must exist at repo root.
- The plugin must be versioned via git tag (source is pulled from the Go module proxy, not a
  branch).
- Any package dependencies must be vendored and committed — moot here since this plugin has none.

If the catalog's import fails, it opens an issue on this repo explaining the problem and pauses
re-import attempts until that issue is closed.

## Conventions to preserve

- Keep `New` and `CreateConfig` signatures exact — Traefik's plugin loader calls them by
  reflection-free convention; changing the signature breaks plugin loading.
- No external dependencies in `go.mod` — Yaegi's plugin sandbox only reliably supports the
  standard library for this kind of plugin.
- Mapping validation (non-empty `query`/`header`) happens in `New`, not in `ServeHTTP`, so
  invalid configs fail fast at middleware construction time.
