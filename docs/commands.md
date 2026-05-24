# Commands

## Overview

```
manpm <command> [options]
```

If no command is given, `install` runs by default.

## Global flags

| Flag | Description |
|------|-------------|
| `--help`, `-h` | Show help |
| `--version`, `-V` | Show version |

## Command reference

### install

Install all dependencies from the lockfile.

```
manpm install [options]
```

| Flag | Short | Type | Default | Description |
|------|-------|------|---------|-------------|
| `--threads` | `-t` | int | 0 (auto) | Number of parallel workers. 0 uses platform-optimal count |
| `--lane-mode` | `-m` | string | `parallel` | Execution mode: `parallel` or `sequential` |
| `--priority-lock` | `-p` | bool | false | Prioritize lockfile over package.json |
| `--dry-run` | | bool | false | Simulate without downloading or extracting |
| `--skip-rebuild` | | bool | false | Skip native module rebuild step |
| `--skip-binlink` | | bool | false | Skip `.bin` executable linking |
| `--skip-scripts` | | bool | false | Skip lifecycle script execution |
| `--force-rebuild` | | bool | false | Force rebuild even if cached |
| `--retries` | `-r` | int | 3 | Maximum retries per package download |

Examples:

```
manpm install
manpm install --threads 8 --dry-run
manpm install --skip-rebuild --skip-scripts
manpm install --threads 4 --retries 5 --priority-lock
```

---

### add

Add a package and show impact preview.

```
manpm add <package> [options]
```

| Flag | Type | Default | Description |
|------|------|---------|-------------|
| `--smart` | bool | false | Enable smart dependency resolution |
| `--why` | bool | false | Show why the package would be needed |
| `--dry-run` | string | `""` | Simulate add. Use `=deep` for deep analysis |
| `--peer-fix` | bool | false | Auto-fix peer dependency conflicts |
| `--dev` | bool | false | Install as a dev dependency |
| `--exact` | bool | false | Save exact version (no range prefix) |

Examples:

```
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

Reads the package's `package.json` from `node_modules/<package>/` and displays name, version, description, homepage, license, dependencies, install script status, and native addon detection.

Examples:

```
manpm explain lodash
manpm explain express
```

Output:

```
Package: lodash
Version: 4.17.21
License: MIT
Dependencies: (none)
Install scripts: false
Native addon: false
```

---

### audit

Scan the lockfile for known vulnerabilities.

```
manpm audit
```

Reads `package-lock.json` and checks against a built-in advisory database. Reports severity, CVE, title, and available fix version for each match. Packages already at the fixed version are skipped.

Example output:

```
lodash [medium] Prototype Pollution in lodash (CVE: CVE-2020-8203)
  Fix available: 4.17.21
```

---

### doctor

Analyze project health and produce a score.

```
manpm doctor
```

Checks:

- `node_modules` existence and content
- Dependency graph for cycles
- Graph size (warns above 500 packages)
- Empty directories

Each issue type deducts from a base score of 100:

| Severity | Deduction | Examples |
|----------|-----------|---------|
| error | -25 | Missing node_modules, graph cycle |
| warning | -10 | Empty graph, empty node_modules |
| suggestion | -3 | Large dependency tree |

---

### map

Render the dependency graph as an ASCII tree.

```
manpm map
```

Groups packages by topological level. Each level is rendered with its packages and their immediate dependencies.

Example output:

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

| Field | Description |
|-------|-------------|
| Score | 0-100 chaos score combining duplicate ratio, depth, cycles |
| Total packages | Count of all packages in the graph |
| Unique libraries | Count of unique package names |
| Avg depth | Weighted average dependency depth |
| Circular deps | Number of detected circular dependencies |
| Redundant groups | Packages installed at multiple different versions |

---

### prune

Show and remove unused packages by scanning source files for import/require statements and tracing transitive dependencies through the dependency graph.

```
manpm prune [options]
```

| Flag | Type | Default | Description |
|------|------|---------|-------------|
| `--safe` | bool | false | Keep packages with recent usage |
| `--dry-run` | bool | false | Only show what would be removed |

---

### run

Run a project script from `package.json`.

```
manpm run <script>
```

Looks up the named script in `package.json` under `scripts` and executes it via the shell. Supports the alias `manpm r <script>`.

Examples:

```
manpm run build
manpm run test
manpm r start
```

---

### sandbox

Display sandbox information for a package.

```
manpm sandbox <package>
```

Output:

```
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

Compare two installed packages.

```
manpm compare <package1> <package2>
```

Reads both `package.json` files and compares version, license, description, homepage, and dependency counts.

---

### sensei

Run a full project architecture review.

```
manpm sensei
```

Lists all files in the project root, checks for config files (package.json, package-lock.json, .npmrc, .nvmrc), scans for known vulnerabilities if a lockfile exists, and reports `node_modules` status.

---

### profile

Manage installation profiles.

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

```
manpm profile list
manpm profile use strict
manpm profile create custom
```
