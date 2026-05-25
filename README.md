# MaNPM

[![CI](https://github.com/abdou-da0wew/MaNPM/actions/workflows/ci.yml/badge.svg)](https://github.com/abdou-da0wew/MaNPM/actions/workflows/ci.yml)
[![Go](https://img.shields.io/badge/Go-1.26+-00ADD8?style=flat-square)](https://go.dev)
[![Node](https://img.shields.io/badge/Node.js-18%2B-brightgreen?style=flat-square)](https://nodejs.org)
[![License: MIT](https://img.shields.io/badge/License-MIT-blue.svg?style=flat-square)](LICENSE)

<img src="assets/banner.png" alt="MaNPM banner" width="100%">

MaNPM is a CLI tool written in Go that parallelizes `npm install`. It reads `package-lock.json`, builds a dependency graph, downloads and extracts tarballs concurrently, verifies SHA512 integrity, orchestrates native module rebuilds, and links `.bin` executables. Zero external dependencies.

## Quick start

```
go install github.com/abdou-da0wew/MaNPM/cmd/manpm@latest
```

Or build from source:

```
git clone https://github.com/abdou-da0wew/MaNPM.git
cd MaNPM
go build -ldflags="-s -w" -o manpm ./cmd/manpm/
```

## Usage

```
manpm install [options]
manpm add <package> [options]
manpm doctor
manpm sensei
manpm audit
manpm pkgjson lock
```

See [Commands](docs/commands.md) for the full reference.

## Documentation

- [Getting Started](docs/getting-started.md)
- [Guide](docs/guide.md)
- [Commands](docs/commands.md)
- [Configuration](docs/configuration.md)
- [Architecture](docs/architecture.md)
- [Development](docs/development.md)
- [Contributing](CONTRIBUTING.md)

## License

MIT
