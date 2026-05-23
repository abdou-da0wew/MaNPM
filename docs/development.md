# Development

## Requirements

- Go 1.26+
- Node.js 18+ and npm 9+ (for native rebuild fallback and E2E testing)
- Git

## Filesystem Note

The Go toolchain uses `RLock` for file access. This is not supported on **fuseblk** filesystems (Android primary storage). All build, test, and lint commands must be run from `/tmp` (f2fs) or another RLock-compatible filesystem.

## Build

```bash
cp -a /storagesdcard/ManPM/* /tmp/manpm-src/
cd /tmp/manpm-src
go build -ldflags="-s -w" -o /tmp/manpm ./cmd/manpm/
```

The output is a statically linked binary at `/tmp/manpm` (~2.4 MB stripped).

## Test

Run all tests:

```bash
cd /tmp/manpm-src && go test ./...
```

Run a single package's tests:

```bash
cd /tmp/manpm-src && go test ./pkg/graph/
cd /tmp/manpm-src && go test ./pkg/config/
cd /tmp/manpm-src && go test ./pkg/platform/
```

Run a single test function:

```bash
cd /tmp/manpm-src && go test -run TestParseTOML ./pkg/config/
cd /tmp/manpm-src && go test -run TestMap ./pkg/intel/
```

Vet all packages:

```bash
cd /tmp/manpm-src && go vet ./...
```

## Sync Back

After making changes, sync back to the source directory:

```bash
cp -a /tmp/manpm-src/* /storagesdcard/ManPM/
```

## Project Layout

```
cmd/manpm/        CLI entry point and command router
pkg/binlink/      .bin symlink management
pkg/buildmgr/     Native package rebuild orchestration
pkg/cache/        Metadata cache (JSON)
pkg/config/       TOML config loader (hand-written parser)
pkg/extractor/    Parallel tarball download and extraction
pkg/graph/        Dependency graph and topological sort
pkg/intel/        Intelligence and analysis commands
pkg/lockfile/     package-lock.json v2/v3 parser
pkg/platform/     OS/arch detection, worker tuning
pkg/preflight/    Pre-install validation
pkg/ui/           Terminal output and color helpers
```

## Testing Conventions

- Tests use `t.TempDir()` for isolated temp directories.
- Tests that exercise extraction or npm interaction create mock directories and files.
- `pkg/intel` tests that call `Explain`, `Audit`, `Compare`, or `Sensei` work against scratch directories, not a live project.
- `pkg/buildmgr` tests create mock `node_modules/` trees with `binding.gyp` and `package.json` fixtures.
- `cmd/manpm` tests call `dispatch()` directly. The `--help` flag is handled in `main()`, not in `dispatch()`, so it is tested via `printHelp()`.
- Test output includes ANSI terminal codes (expected for `pkg/ui` tests).

## Coding Standards

- Zero external dependencies. All imports must be from the Go standard library.
- Follow existing package patterns: one file per package, test file co-located.
- Error types that wrap other errors implement `Unwrap()`.
- Use `context.Context` as the first parameter for any function that may block.
- Run `go vet ./...` before committing.

## Adding a New Command

1. Add a new `Command` struct in `cmd/manpm/commands.go` inside `buildRouter()`.
2. Define `Flags` if needed, and implement the `Run` function.
3. Append the command to `root.Subcommands`.
4. Add tests in `cmd/manpm/manpm_test.go`.

## Adding a New Package

1. Create `pkg/<name>/` directory.
2. Write the implementation file.
3. Write `_test.go` with table-driven tests where applicable.
4. Import from `manpm/pkg/<name>` in the command router or other packages.

## Color Palette

UI colors are defined in `pallet.json` and used in `pkg/ui/ui.go` as ANSI escape constants:

| Constant | Hex | Usage |
|----------|-----|-------|
| Orange | `#E35A00` | Headers, logo accents |
| Cyan | `#6CD0E5` | Info, subheaders |
| Green | `#00FF00` (ANSI) | Success |
| Yellow | `#FFFF00` (ANSI) | Warnings |
| Red | `#FF0000` (ANSI) | Errors |
| Gray | `#555555` (ANSI) | Labels, dim text |

## Git Remote

```bash
git remote -v
origin  git@github.com:abdou-da0wew/MaNPM.git (fetch)
origin  git@github.com:abdou-da0wew/MaNPM.git (push)
```

Push via SSH:

```bash
git push origin main
```
