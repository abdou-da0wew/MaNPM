# Development

## Prerequisites

- Go 1.26+
- Node.js 18+ and npm 9+ (for native rebuild fallback and E2E tests)
- Git

## Building

```
git clone https://github.com/abdou-da0wew/MaNPM.git
cd MaNPM
go build -ldflags="-s -w" -o manpm ./cmd/manpm/
```

## Testing

Run all tests:

```
go test ./...
```

Run a specific package:

```
go test ./pkg/graph/
go test ./pkg/config/
go test ./pkg/platform/
```

Run a specific test:

```
go test -run TestParseTOML ./pkg/config/
go test -run TestMap ./pkg/intel/
```

Run vet:

```
go vet ./...
```

## Project layout

```
cmd/manpm/        CLI entry point and command router
pkg/binlink/      .bin symlink management
pkg/buildmgr/     Native package rebuild orchestration
pkg/cache/        Metadata cache
pkg/config/       TOML config loader
pkg/extractor/    Parallel tarball download and extraction
pkg/graph/        Dependency graph and topological sort
pkg/intel/        Intelligence and analysis commands
pkg/lockfile/     package-lock.json parser
pkg/platform/     OS/arch detection, worker tuning
pkg/preflight/    Pre-install validation
pkg/ui/           Terminal output and color helpers
```

## Coding standards

- **Zero external dependencies**. All imports must come from the Go standard library.
- Follow existing package patterns: one file per package, test file co-located.
- Use `context.Context` as the first parameter for any function that may block.
- Error types that wrap other errors should implement `Unwrap()`.
- Run `go vet ./...` before committing.

## Testing conventions

- Use `t.TempDir()` for isolated test directories.
- `pkg/intel` tests work against scratch directories, not a live project.
- `pkg/buildmgr` tests create mock `node_modules/` trees with `binding.gyp` and `package.json` fixtures.
- `cmd/manpm` tests call `dispatch()` directly. The `--help` flag is handled in `main()`, not `dispatch()`.

## Adding a new command

1. Define a new `Command` struct in `cmd/manpm/commands.go` inside `buildRouter()`.
2. Set `Flags` and implement the `Run` function.
3. Append the command to `root.Subcommands`.
4. Add dispatch tests in `cmd/manpm/manpm_test.go`.

## Adding a new package

1. Create `pkg/<name>/` with the implementation.
2. Write `_test.go` with table-driven tests.
3. Import as `manpm/pkg/<name>`.

## Color palette

UI colors are defined as ANSI escape constants in `pkg/ui/ui.go`:

| Constant | Code | Usage |
|----------|------|-------|
| Orange | `\033[38;5;208m` | Headers, logo accents |
| Cyan | `\033[36m` | Info, subheaders |
| Green | `\033[32m` | Success |
| Yellow | `\033[33m` | Warnings |
| Red | `\033[31m` | Errors |
| Gray | `\033[90m` | Labels, dim text |
| Magenta | `\033[35m` | Accents |
| White | `\033[97m` | Header titles |

## Release

```
VERSION=v0.1.0
git tag $VERSION
git push origin $VERSION
```
