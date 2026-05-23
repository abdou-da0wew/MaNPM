# Architecture

## Package Map

```
cmd/manpm/
  main.go         Entry point. Parses os.Args, dispatches to subcommands
  commands.go     Command router (buildRouter/dispatch), 12 subcommand definitions

pkg/
  binlink/        Create and manage .bin symlinks (POSIX) and .cmd wrappers (Windows)
  buildmgr/       Detect native packages, orchestrate rebuild chain
  cache/          JSON metadata cache for package resolutions
  config/         TOML config loader and profile system (hand-rolled parser)
  extractor/      Parallel tarball download, SHA512 verification, gzip/tar extraction
  graph/          Dependency graph builder, topological sort (Kahn), cycle detection
  intel/          Project intelligence: explain, audit, doctor, map, entropy, sensei, compare, sandbox
  lockfile/       package-lock.json v2/v3 parser
  platform/       OS/arch detection, memory-aware worker tuning
  preflight/      Pre-install validation: package.json, lockfile, PATH, permissions
  ui/             ANSI terminal output with color palette
```

## Execution Flow

### Install Path

1. **Preflight** (`pkg/preflight`) validates the project directory: checks for `package.json`, parses `package-lock.json`, verifies `node` and `npm` are in PATH, and confirms write access to `node_modules/`.
2. **Lockfile** (`pkg/lockfile`) parses the lockfile and extracts every package entry with its resolved URL, integrity hash, and dependencies.
3. **Graph** (`pkg/graph`) builds a DAG from the lockfile entries, performs a Kahn topological sort to assign each package to an execution level, then runs a smallest-last heuristic to split packages into lightweight and heavyweight groups.
4. **Extractor** (`pkg/extractor`) processes packages level by level. For each level, it spawns up to `parallel_limit` goroutines that download tarballs, verify SHA512 integrity during streaming, extract gzip/tar content to `node_modules/`, and sanitize paths to prevent zip slip attacks. If extraction fails, it falls back to `npm install` for that specific package.
5. **Build** (`pkg/buildmgr`) detects native packages (those with `binding.gyp`, `gypfile`, install scripts, or prebuilds) and runs the rebuild chain: prebuild-install, then node-gyp rebuild, then npm rebuild, then build from source.
6. **Binlink** (`pkg/binlink`) reads the `bin` field from each package's `package.json` and creates symlinks (POSIX) or `.cmd` wrappers (Windows) in `node_modules/.bin/`.

## Data Flow

```
package-lock.json
       |
       v
  lockfile.Parse()  -->  map of PackageDef (name, version, resolved, integrity, deps)
       |
       v
  graph.DependencyGraph.AddNode()  for each package
  graph.TopologicalSort()          Kahn's algorithm
  graph.SmallestLastHeuristic()    split into light/heavy levels
       |
       v
  extractor.ExtractLevel()         parallel goroutines per level
  each goroutine:
    download tarball (HTTP)
    verify SHA512 integrity (streaming)
    gzip decompress
    tar extract to node_modules/<path>
  fallback: npm install <pkg> if extraction fails
       |
       v
  buildmgr.DetectNativePackages()  scan for gyp/prebuild/scripts
  buildmgr.RebuildAll()            prebuild -> node-gyp -> npm rebuild -> source
       |
       v
  binlink.LinkAllPackages()        symlink .bin entries
```

## Design Decisions

### Zero External Dependencies

All packages use only Go standard library types. There is no dependency on `github.com/BurntSushi/toml` (TOML parser is hand-written in `pkg/config`), `github.com/spf13/cobra` (CLI routing is hand-rolled in `cmd/manpm/commands.go`), or any other third-party module. This keeps the binary small (2.4 MB) and avoids supply chain risk and Android/Termux cross-compilation issues.

### Custom Subcommand Router

Instead of using cobra or urfave/cli, commands are defined as a `Command` struct tree with `Name`, `Description`, `Usage`, `Flags`, `Run` function, and optional `Subcommands`. Each command creates its own `flag.FlagSet` for parsing. The `dispatch()` function walks the tree, matches by name, and calls the appropriate `Run`. Unknown commands trigger a Levenshtein-based suggestion system.

### Level-Based Parallel Extraction

Packages are grouped into topological levels where all packages in a level have no interdependencies. All packages in the same level can be extracted concurrently. The extractor creates a goroutine pool per level, with semaphore-based concurrency limiting. If any individual extraction fails after retries, it falls back to `npm install` for that specific package, preserving the install for everything else.

### Build from /tmp

The Go toolchain requires RLock support in the filesystem. On Android, the primary storage (`/storagesdcard/`) uses fuseblk which does not support RLock. All builds and tests must be performed from `/tmp` (typically f2fs) and synced back. This is documented in [Development](development.md).

### Platform Detection

The `platform` package detects the OS (Linux, macOS, Windows, Android) and architecture (ARM, ARM64, AMD64, 386). On Android, it reads `/system/build.prop` for OS detection and `/proc/meminfo` for memory-aware worker tuning. Symlinks are disabled on Windows unless `MANPM_FORCE_SYMLINKS` is set.
