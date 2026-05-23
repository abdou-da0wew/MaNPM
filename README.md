# MaNPM

[![Go](https://img.shields.io/badge/Go-1.26+-00ADD8?style=flat-square)](https://go.dev)
[![Node](https://img.shields.io/badge/Node.js-18%2B-brightgreen?style=flat-square)](https://nodejs.org)
[![MIT](https://img.shields.io/badge/License-MIT-blue.svg?style=flat-square)](LICENSE)

<img src="assets/banner.png" alt="MaNPM banner" width="100%">

MaNPM is a Go CLI tool that replaces `npm install` with parallel tarball download, SHA512 integrity verification, topological dependency resolution, and native module rebuild orchestration. It has zero external dependencies and runs on Linux, macOS, Windows, and Android ARM64.

## Requirements

- Go 1.26+ (to build from source)
- Node.js 18+ and npm 9+ (runtime dependency for native rebuild fallback)

## Install

Build from source. The Go toolchain requires RLock filesystem support; use `/tmp` on systems with fuseblk storage:

```bash
cp -a /storagesdcard/ManPM/* /tmp/manpm-src/
cd /tmp/manpm-src && go build -ldflags="-s -w" -o /tmp/manpm ./cmd/manpm/
```

## Usage

```
manpm install [options]
```

This reads `package-lock.json`, builds a dependency DAG, downloads and extracts tarballs in parallel with SHA512 verification, runs native module rebuilds, and links `.bin` executables.

```bash
manpm install --threads 8
manpm doctor
manpm sensei
manpm add express --dev --exact
```

## Configuration

Create `manpm.config.toml` in the project root:

```toml
[core]
parallel_limit = 8
auto_fix_peers = true

[install_policy]
prefer_lockfile = true
version_strategy = "stable"

[profile.strict]
ui = "minimal"
safe_mode = true
```

See [Configuration](docs/configuration.md) for all fields and profiles.

## Architecture

```
cmd/manpm/          CLI entry point and subcommand router
pkg/binlink/        .bin symlink management
pkg/buildmgr/       Native package rebuild chain
pkg/cache/          JSON metadata cache
pkg/config/         TOML config loader (zero dependencies)
pkg/extractor/      Parallel download, SHA512 verify, tar extraction
pkg/graph/          DAG builder, topological sort, cycle detection
pkg/intel/          Project intelligence (explain, audit, doctor, map, entropy, sensei, compare)
pkg/lockfile/       package-lock.json v2/v3 parser
pkg/platform/       OS/arch detection, memory-aware worker tuning
pkg/preflight/      Pre-install validation
pkg/ui/             ANSI terminal output
```

See [Architecture](docs/architecture.md) for data flow, design decisions, and the full package reference.

## Documentation

- [Getting Started](docs/getting-started.md)
- [Commands](docs/commands.md)
- [Configuration](docs/configuration.md)
- [Architecture](docs/architecture.md)
- [Development](docs/development.md)
- [API Reference](docs/api.md)

## License

MIT
