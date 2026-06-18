# Query To Header

[![Maintenance](https://img.shields.io/maintenance/yes/2026.svg)](https://github.com/muench-dev/query-to-header)
[![Go CI](https://github.com/muench-dev/query-to-header/actions/workflows/go.yml/badge.svg)](https://github.com/muench-dev/query-to-header/actions/workflows/go.yml)
[![CodeQL](https://github.com/muench-dev/query-to-header/actions/workflows/codeql-analysis.yml/badge.svg)](https://github.com/muench-dev/query-to-header/actions/workflows/codeql-analysis.yml)
[![Go Report Card](https://goreportcard.com/badge/github.com/muench-dev/query-to-header)](https://goreportcard.com/report/github.com/muench-dev/query-to-header)

A [Traefik](https://traefik.io) middleware plugin that copies values from incoming HTTP
request query parameters into HTTP request headers before the request reaches the next
handler in the chain.

It is useful when an upstream service expects authentication tokens, tenant IDs, or other
metadata in headers, but clients (or legacy integrations) only have the ability to send them
as query parameters.

## How it works

For every configured mapping, the plugin:

1. Reads the named query parameter from the incoming request.
2. If the parameter is absent, does nothing for that mapping.
3. If a header with the target name already exists, the value is **left untouched** unless
   `overwrite: true` is set.
4. Sets the header to the **first** value of the query parameter (query parameters can be
   repeated; only `values[0]` is used).
5. If `bearer: true`, the value is prefixed with `Bearer ` before being set — useful for turning
   a query parameter directly into an `Authorization` header.
6. If `remove: true`, deletes the query parameter from the request URL so it is not forwarded
   upstream.
7. Mappings are applied in the order they are declared; later mappings can see headers/query
   changes made by earlier ones.

The plugin never touches the response — it only rewrites the request before calling the next
handler.

## Installation

Traefik plugins are not compiled into a binary you install separately — Traefik downloads the
plugin source and runs it through its embedded [Yaegi](https://github.com/traefik/yaegi)
interpreter. You only need to reference it in Traefik's static configuration.

### Static configuration

Pin a specific tagged version (recommended for production):

```yaml
experimental:
  plugins:
    query-to-header:
      moduleName: github.com/muench-dev/query-to-header
      version: v0.1.0
```

```toml
# traefik.toml
[experimental.plugins.query-to-header]
  moduleName = "github.com/muench-dev/query-to-header"
  version = "v0.1.0"
```

```bash
# CLI
--experimental.plugins.query-to-header.modulename=github.com/muench-dev/query-to-header
--experimental.plugins.query-to-header.version=v0.1.0
```

### Local plugin (for development against an unpublished version)

Point Traefik at a local checkout instead of a tagged release by using
`localPlugins` and mounting/copying this repository into Traefik's plugins
directory (`./plugins-local/src/github.com/muench-dev/query-to-header` by default):

```yaml
experimental:
  localPlugins:
    query-to-header:
      moduleName: github.com/muench-dev/query-to-header
```

## Dynamic configuration

Once declared in the static configuration, configure it as a middleware and attach it to a
router:

```yaml
http:
  middlewares:
    my-query-to-header:
      plugin:
        query-to-header:
          mappings:
            - query: token
              header: X-Token
              remove: true
              overwrite: false

  routers:
    my-router:
      rule: Host(`example.com`)
      service: my-service
      middlewares:
        - my-query-to-header
```

Equivalent labels (Docker provider):

```yaml
labels:
  - "traefik.http.middlewares.my-query-to-header.plugin.query-to-header.mappings[0].query=token"
  - "traefik.http.middlewares.my-query-to-header.plugin.query-to-header.mappings[0].header=X-Token"
  - "traefik.http.middlewares.my-query-to-header.plugin.query-to-header.mappings[0].remove=true"
```

### Mapping options

| Option      | Type   | Description                                                                    | Default |
|-------------|--------|----------------------------------------------------------------------------------|---------|
| `query`     | string | Name of the query parameter to read. Required.                                   | -       |
| `header`    | string | Name of the HTTP header to set with the query parameter's value. Required.       | -       |
| `remove`    | bool   | Remove the query parameter from the request URL after copying it.                | `false` |
| `overwrite` | bool   | Replace an existing header with the same name instead of leaving it untouched.   | `false` |
| `bearer`    | bool   | Prefix the header value with `Bearer ` (e.g. for `Authorization` headers).       | `false` |

Multiple mappings can be declared; an empty `query` or `header` on any mapping causes the
middleware to fail to load (validated in `New`, at construction time, not per-request).

## Project layout

| File                          | Purpose                                                              |
|--------------------------------|----------------------------------------------------------------------|
| `query_to_header.go`          | Plugin implementation: `Config`, `CreateConfig`, `New`, `ServeHTTP`. |
| `query_to_header_test.go`     | Unit tests using `httptest`.                                        |
| `.traefik.yml`                | Plugin manifest required by the Traefik Plugin Catalog.             |
| `go.mod`                       | No third-party dependencies — required for Yaegi compatibility.     |
| `.golangci.yml`                | Lint configuration.                                                  |
| `justfile`                     | `lint`, `test`, `vendor`, `clean` recipes (run with [`just`](https://just.systems)). |
| `.release-it.json`             | Config for [`release-it`](https://github.com/release-it/release-it) (must be installed globally). |

## Local development

```bash
go build ./...              # compile-correctness check (plugin is never shipped as a binary)
go vet ./...
go test -v -cover ./...     # run all tests
go test -run TestName -v ./... # run a single test
golangci-lint run           # lint, config in .golangci.yml
just                         # default recipe: lists all available recipes
```

### Testing against a real Traefik instance

1. Clone this repository into `./plugins-local/src/github.com/muench-dev/query-to-header`
   relative to your Traefik working directory.
2. Add the `localPlugins` static configuration shown above.
3. Start Traefik and confirm the plugin loads without errors in the logs.
4. Attach the middleware to a router and send a request with the configured query parameter
   to confirm the header is set on the upstream request.

## Releasing

Releases are tag-driven. [`release-it`](https://github.com/release-it/release-it) (install it
globally, e.g. `brew install release-it` or `npm install -g release-it`) is used locally to
bump the version, commit, create the `vX.Y.Z` tag, and push it:

```bash
release-it           # interactively pick the version bump
release-it patch      # or specify the bump directly
```

`release-it` only creates and pushes the git tag — it does not publish to npm or create a
GitHub Release itself (see `.release-it.json`). Pushing the tag triggers the `release` GitHub
Actions workflow, which runs `goreleaser` to publish the actual GitHub Release.

## Constraints

Traefik plugins run inside Yaegi, which only reliably supports the Go standard library.
Do not add third-party dependencies to `go.mod` — the plugin must remain dependency-free to
load correctly in the Traefik Plugin Catalog and in Yaegi generally.
