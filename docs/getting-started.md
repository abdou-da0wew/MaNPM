# Getting Started

## Prerequisites

- Go 1.26+ (to build from source)
- Node.js 18+ and npm 9+ (runtime requirement for native rebuild fallback)
- Git

## Installation

### Using go install

```
go install github.com/abdou-da0wew/MaNPM/cmd/manpm@latest
```

The binary is placed at `$GOPATH/bin/manpm` (or `$HOME/go/bin/manpm`). Ensure that directory is in your `$PATH`.

### Build from source

```
git clone https://github.com/abdou-da0wew/MaNPM.git
cd MaNPM
go build -ldflags="-s -w" -o manpm ./cmd/manpm/
```

This produces a statically linked binary at `./manpm` (~2.4 MB stripped).

### Verify

```
manpm --help
```

You should see the help screen listing all 12 commands.

## Your first run

### 1. Create or navigate to a Node.js project

```
cd my-node-project
```

### 2. Ensure a lockfile exists

MaNPM reads `package-lock.json` (v2 or v3). If you do not have one, run `npm install` first to generate it.

### 3. Scan your project

```
manpm doctor
```

This checks for `package.json`, reads `package-lock.json`, verifies `node` and `npm` are in PATH, checks filesystem permissions, and reports a health score out of 100.

### 4. Install dependencies

```
manpm install
```

This parses the lockfile, builds a dependency DAG, downloads and extracts tarballs in parallel with SHA512 verification, runs native module rebuilds if needed, and links `.bin` executables.

For a dry run that shows what would happen without making changes:

```
manpm install --dry-run
```

### 5. Explore your project

```
manpm audit          # Check for known vulnerabilities
manpm map            # Show the dependency graph as a tree
manpm entropy        # Measure project chaos metrics
manpm sensei         # Full project review
```

## Configuration

Create a `manpm.config.toml` file in your project root to customize behavior. See [Configuration](configuration.md) for all available options.

Example:

```toml
[core]
parallel_limit = 8
auto_fix_peers = true

[install_policy]
prefer_lockfile = true
version_strategy = "stable"

[ui]
mode = "developer"
```

## Next steps

- [Guide](guide.md) for detailed walkthroughs
- [Commands](commands.md) for every subcommand and flag
- [Configuration](configuration.md) for all config fields
