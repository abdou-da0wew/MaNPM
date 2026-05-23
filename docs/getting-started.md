# Getting Started

## Prerequisites

- Go 1.26+
- Node.js 18+ and npm 9+ (for native rebuild fallback)
- Git (for cloning)

## Build

The Go toolchain cannot RLock files on fuseblk filesystems. Build from `/tmp`:

```bash
cp -a /storagesdcard/ManPM/* /tmp/manpm-src/
cd /tmp/manpm-src
go build -ldflags="-s -w" -o /tmp/manpm ./cmd/manpm/
```

The binary is approximately 2.4 MB (stripped) and has zero external dependencies.

## First Run

```bash
cd /tmp/manpm-src
/tmp/manpm
```

This shows the help screen with all 12 available commands.

```bash
/tmp/manpm install
```

This runs a simulated install displaying the configuration that would be used. In production, this reads `package-lock.json`, resolves the dependency graph, downloads and extracts tarballs in parallel, verifies SHA512 checksums, runs native rebuilds, and links `.bin` executables.

```bash
/tmp/manpm doctor
```

Scans the project directory for `package.json`, `package-lock.json`, verifies `node` and `npm` are in PATH, checks filesystem permissions, and reports a health score out of 100.

## Sync Back

After building or modifying code, sync back to the source directory:

```bash
cp -a /tmp/manpm-src/* /storagesdcard/ManPM/
```

## Next Steps

- [Commands reference](commands.md) for all subcommands and flags
- [Configuration guide](configuration.md) for `manpm.config.toml`
- [Architecture overview](architecture.md) for package structure and data flow
