# Commands

## Overview

```
manpm <command> [options]
```

If no command is given, `install` runs by default.

## Commands

### install

Install all dependencies. This is the primary command.

```
manpm install [options]
```

| Flag | Short | Type | Default | Description |
|------|-------|------|---------|-------------|
| `--threads` | `-t` | int | 0 (auto) | Number of parallel workers |
| `--lane-mode` | `-m` | string | `parallel` | Execution mode: `parallel` or `sequential` |
| `--priority-lock` | `-p` | bool | false | Prioritize lockfile parsing over package.json |
| `--dry-run` | | bool | false | Simulate without downloading or extracting |
| `--skip-rebuild` | | bool | false | Skip native module rebuild step |
| `--skip-binlink` | | bool | false | Skip `.bin` executable linking |
| `--skip-scripts` | | bool | false | Skip lifecycle script execution |
| `--force-rebuild` | | bool | false | Force native rebuild even if cached |
| `--retries` | `-r` | int | 3 | Maximum retries per failed package download |

Examples:

```bash
manpm install
manpm install --threads 8 --dry-run
manpm install --skip-rebuild --skip-scripts
manpm install --threads 4 --retries 5 --priority-lock
```

---

### add

Add a package and preview its impact.

```
manpm add <package> [options]
```

| Flag | Type | Default | Description |
|------|------|---------|-------------|
| `--smart` | bool | false | Enable smart resolution |
| `--why` | bool | false | Show why this package would be needed |
| `--dry-run` | string | `""` | Simulate add. Use `=deep` for deep analysis |
| `--peer-fix` | bool | false | Auto-fix peer dependency conflicts |
| `--dev` | bool | false | Install as a dev dependency |
| `--exact` | bool | false | Save exact version (no ^ or ~) |

Examples:

```bash
manpm add express
manpm add lodash --dev --exact
manpm add react --why
manpm add typescript --dry-run=deep
```

---

### explain

Show detailed information about an installed package.

```
manpm explain <package>
```

Reads the package's `package.json` from `node_modules/<package>/` and displays name, version, description, homepage, license, dependencies, dev dependencies, install script status, and native addon detection.

Examples:

```bash
manpm explain lodash
manpm explain express
```

---

### audit

Run vulnerability analysis against the lockfile.

```
manpm audit
```

Reads `package-lock.json`, cross-references against a built-in advisory database for known vulnerabilities (lodash, minimist, node-fetch, follow-redirects, glob-parent), and reports severity, CVE, title, and fix version for each match.

Example output:

```
Package: lodash  Severity: medium  Title: Prototype Pollution in lodash
CVE: CVE-2020-8203  Fix: 4.17.21
```

---

### doctor

Analyze project health and get a numeric score.

```
manpm doctor
```

Checks: existence of `node_modules`, dependency graph cycles, graph size, empty directories. Each issue deducts from a base score of 100 (errors -25, warnings -10, suggestions -3).

Example output:

```
Score: 65.0%
node_modules directory does not exist
Fix: Run npm install
```

---

### map

Show the dependency graph as an ASCII tree.

```
manpm map
```

Uses the topological sort levels to render a tree showing each package at its dependency level with its version and immediate dependencies.

Example:

```
Dependency Map
==============

Level 0 (1 packages):
+-- root@1.0.0
`-- dep-a (depends)

Level 1 (1 packages):
`-- dep-a@1.0.0
```

---

### entropy

Measure project chaos metrics.

```
manpm entropy
```

| Metric | Description |
|--------|-------------|
| Score | 0-100 chaos score combining duplicate ratio, depth, and cycles |
| Total packages | Number of packages in the dependency graph |
| Unique libraries | Number of unique package names |
| Avg depth | Weighted average dependency depth |
| Circular deps | Number of detected circular dependencies |
| Redundant groups | Packages installed at multiple different versions |

---

### prune

Show and remove unused packages.

```
manpm prune [options]
```

| Flag | Type | Default | Description |
|------|------|---------|-------------|
| `--safe` | bool | false | Keep packages with recent usage |
| `--dry-run` | bool | false | Only show what would be removed |

---

### sandbox

Show isolation information for a package.

```
manpm sandbox <package>

Sandbox for lodash:
  - Runtime: Node.js with limited permissions
  - Network: restricted to whitelisted registries
  - Filesystem: read-only except for lodash's own directory
  - Execution timeout: 30s
  - No child process spawning
  - Memory limit: 512MB
```

---

### compare

Compare two installed packages side by side.

```
manpm compare <package1> <package2>
```

Reads both `package.json` files from `node_modules/` and compares: version, license, description, homepage, dependency count, dev dependency count.

Example:

```bash
manpm compare express koa
```

---

### sensei

Full project architecture review.

```
manpm sensei
```

Lists all files in the project root, checks for config files (package.json, package-lock.json, .npmrc, .nvmrc), scans for known vulnerabilities, and checks `node_modules` status.

---

### profile

Manage installation profiles defined in `manpm.config.toml`.

```
manpm profile <subcommand> [name]
```

| Subcommand | Description |
|------------|-------------|
| `list` | List available profiles |
| `use <name>` | Switch to a named profile |
| `create <name>` | Create a new profile |
| `delete <name>` | Delete an existing profile |

Examples:

```bash
manpm profile list
manpm profile use strict
manpm profile create custom
```
