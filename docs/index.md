# MaNPM Documentation

[Getting Started](getting-started.md) | [Commands](commands.md) | [Configuration](configuration.md) | [Architecture](architecture.md) | [Development](development.md) | [API Reference](api.md)

MaNPM is a Go-based CLI tool that orchestrates npm package installations in parallel. It reads `package-lock.json`, builds a dependency graph, streams tarballs concurrently, verifies SHA512 integrity, handles native module rebuilds, and links executables. The goal is to replace `npm install` with a faster, more transparent alternative that works across Linux, macOS, Windows, and Android ARM64.

## Quick Links

- [Getting Started](getting-started.md) -- prerequisites, build, first run
- [Commands](commands.md) -- every subcommand with flags and examples
- [Configuration](configuration.md) -- `manpm.config.toml` field reference
- [Architecture](architecture.md) -- package map, data flow, design decisions
- [Development](development.md) -- build from source, run tests, contribute
- [API Reference](api.md) -- exported Go types and functions

## Repository

- GitHub: [github.com/abdou-da0wew/MaNPM](https://github.com/abdou-da0wew/MaNPM)
- License: MIT
- Go version: 1.26
- Dependencies: zero (stdlib only)
