# MaNPM - The blazing-fast Go NPM Parallel Orchestrator

> A high-performance CLI tool written in Go that radically speeds up Node.js package installations by parsing \`package-lock.json\`, topological sorting dependencies, streaming tarballs directly to disk using goroutines, and deferring compilation to standard \`npm\`.

## Features

- ⚡ **Parallel Extraction** — Downloads and extracts packages concurrently using Go goroutines
- 📦 **Native Module Support** — Handles node-gyp rebuilds, prebuild-install, and fallback builds
- 🔍 **Project Intelligence** — \`doctor\`, \`entropy\`, \`explain\`, \`sensei\`, \`map\` for deep project insights
- 🔒 **Integrity Verification** — SHA512 checksums verified during streaming extraction
- 🧠 **Context-Aware** — Remembers your preferences, profiles, and project patterns
- 🎨 **Beautiful CLI** — Colorful output with live spinners, progress bars, and Bun-inspired UX
- 📝 **TOML Config** — \`manpm.config.toml\` with profiles and per-project overrides
- 📱 **Cross-Platform** — Linux, macOS, Windows, Android ARM64

## Quick Start

\`\`\`bash
# Install dependencies
manpm install

# Add a package with smart resolution
manpm add express --smart

# Full project health check
manpm doctor

# Get a senior dev review
manpm sensei
\`\`\`

## Commands

| Command | Description |
|---------|-------------|
| \`install\` | Install all dependencies (parallel) |
| \`add\` | Add a package with impact preview |
| \`explain\` | Show why a package is installed |
| \`audit\` | Run vulnerability analysis |
| \`doctor\` | Analyze project health |
| \`map\` | Show ASCII dependency graph |
| \`entropy\` | Measure project chaos level |
| \`prune\` | Find and remove unused deps |
| \`sandbox\` | Show isolated install info |
| \`compare\` | Compare two packages |
| \`sensei\` | Full project architecture review |
| \`profile\` | Manage installation profiles |

## Configuration

Create \`manpm.config.toml\` in your project root:

\`\`\`toml
[core]
parallel_limit = 8
auto_fix_peers = true

[ui]
mode = "developer"  # minimal | developer | psychotic

[profile.strict]
ui = "minimal"
safe_mode = true
\`\`\`

## How It Works

1. Parses \`package-lock.json\` (v2/v3)
2. Builds a DAG and topologically sorts dependencies
3. Groups packages into parallel execution levels
4. Streams tarballs directly to disk via Go (bypassing npm tree builder)
5. Verifies SHA512 integrity during download
6. Runs sequential \`npm rebuild\` for native modules
7. Links \`.bin\` executables
8. Reports project health and insights

## Platform Support

| Platform | Status |
|----------|--------|
| Linux x86_64 | ✅ |
| macOS ARM64 | ✅ |
| macOS x86_64 | ✅ |
| Windows x86_64 | ✅ |
| Android ARM64 | ✅ Tested |
| Linux ARM64 | ✅ |

## License

MIT

