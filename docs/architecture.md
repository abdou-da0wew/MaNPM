# Architecture

## Package overview

```
cmd/manpm/          CLI entry point and subcommand router
pkg/binlink/        .bin symlink management
pkg/buildmgr/       Native package detection and rebuild chain
pkg/cache/          JSON metadata cache for package resolutions
pkg/config/         TOML configuration loader
pkg/extractor/      Parallel tarball download, SHA512 verification, extraction
pkg/graph/          Dependency graph, topological sort, cycle detection
pkg/intel/          Project intelligence (explain, audit, doctor, map, entropy, sensei, compare, sandbox)
pkg/lockfile/       package-lock.json v2/v3 parser
pkg/platform/       OS/arch detection, memory-aware worker tuning
pkg/preflight/      Pre-install validation
pkg/ui/             ANSI terminal output
```

## Install data flow

```
package-lock.json
       |
       v
  lockfile.Parse()       --> []PackageDef
       |
       v
  graph.DependencyGraph  --> AddNode() for each package
  TopologicalSort()      --> Kahn's algorithm
  SmallestLastHeuristic()--> light/heavy levels
       |
       v
  extractor.ExtractLevel()  --> goroutine pool per level
    download + SHA512 verify (streaming via io.TeeReader)
    gzip decompress + tar extract
    integrity check on completed output
    fallback to npm install on failure
       |
       v
  buildmgr.DetectNativePackages()
  rebuild chain: prebuild-install -> node-gyp rebuild -> npm rebuild -> build from source
       |
       v
   binlink.LinkAllPackages()  --> symlink .bin entries
       |
       v
   buildmgr.RunPostinstallScripts()  --> lifecycle scripts
```

## Design decisions

### Zero external dependencies

All packages use only the Go standard library. The TOML parser in `pkg/config/` is hand-written. The CLI router in `cmd/manpm/commands.go` uses `flag.FlagSet` from stdlib instead of cobra or urfave/cli. This keeps the binary small (2.4 MB stripped) and avoids supply chain risk.

### Kahn topological sort

Packages are assigned to execution levels using Kahn's algorithm. Every package at the same level can be extracted concurrently because no dependencies exist between them. If a cycle exists (package A depends on B, B depends on A), a warning is emitted and all packages are processed in a single pass. Cycles do not block the install.

### Level-based parallel extraction

Within each topological level, the extractor creates a goroutine pool. Each goroutine handles one package: HTTP GET the tarball, stream through SHA512 hasher (using `io.TeeReader`), decompress gzip, extract tar entries to disk. Paths are sanitized and checked against the target directory boundary to prevent Zip Slip attacks.

### Retry with backoff

Failed downloads retry up to `MaxRetries` times with exponential backoff (attempt^2 seconds). Network-related errors (timeout, connection reset, TLS failure, HTTP 5xx) are retryable. Integrity validation failures are not retryable because they indicate a corrupted response.

### npm fallback

If all extraction attempts fail, the extractor falls back to `npm install <package>@<version>` for that specific package. This ensures the install can still complete even when the Go extraction engine encounters an unsupported edge case.

### Platform detection

The `pkg/platform/` package detects the OS and architecture. On Linux, it checks for `/system/build.prop` to distinguish Android. On Android, it reads `/proc/meminfo` for memory-aware worker count capping. Symlinks are disabled on Windows unless `MANPM_FORCE_SYMLINKS` is set.

### Rebuild chain

Native packages are detected by scanning for `binding.gyp`, `binding.gypi`, the `gypfile` flag in `package.json`, prebuild directories, install/postinstall scripts, or existing build directories. The rebuild chain tries each strategy in order and stops at the first success:
1. prebuild-install (prebuilt binary download)
2. node-gyp rebuild (system node-gyp)
3. npm rebuild --foreground-scripts
4. npm rebuild --build-from-source
