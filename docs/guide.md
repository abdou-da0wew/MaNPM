# Guide

## How MaNPM installs packages

The install process has six stages:

### 1. Preflight

MaNPM validates the project directory. It checks for `package.json`, reads and parses `package-lock.json`, verifies `node` and `npm` are available in `$PATH`, and confirms write access to the `node_modules` directory.

Use `manpm doctor` to run preflight checks independently.

### 2. Graph construction

Every entry in `package-lock.json` is added to a dependency graph. Each package records its version, resolved URL, integrity hash, and declared dependencies. A Kahn topological sort assigns every package to a level where all packages at the same level have no dependencies on each other and can be processed concurrently.

If a cycle is detected, a warning is shown and the install continues by processing all packages in a single pass. Cycles do not block the install.

### 3. Download and extraction

Packages are processed level by level. Within each level, a configurable number of goroutines download tarballs over HTTP, verify SHA512 integrity during streaming (using `io.TeeReader`), decompress gzip, and extract tar entries to `node_modules/<path>`. Path traversal attacks are prevented by sanitizing every tar path and checking it against the target directory boundary.

Failed downloads are retried up to `retry_count` times with exponential backoff (attempt^2 seconds). Non-retryable failures (integrity mismatch, HTTP 4xx) abort immediately. If all retries are exhausted, the extractor falls back to `npm install` for that specific package.

### 4. Native module rebuild

After extraction, MaNPM scans `node_modules/` for native packages (those with `binding.gyp`, the `gypfile` flag, prebuild directories, or install/postinstall scripts). For each native package, it tries the rebuild chain in order:

1. **prebuild-install** -- download prebuilt binaries if available
2. **node-gyp rebuild** -- compile using node-gyp
3. **npm rebuild --foreground-scripts** -- let npm handle it
4. **npm rebuild --build-from-source** -- compile from source as last resort

Heavyweight packages (canvas, sharp, cypress, puppeteer, electron, playwright, esbuild, swc, etc.) are detected and their environment is configured to skip unnecessary binary downloads during rebuild.

### 5. Binary linking

Each package's `bin` field (from `package.json`) is read and used to create symlinks in `node_modules/.bin/`. On POSIX systems, symlinks are created. On Windows, `.cmd` wrapper scripts are generated. Scoped packages (`@scope/name`) have the scope stripped from the link name.

### 6. Lifecycle scripts

After binary linking, MaNPM runs `postinstall` scripts for each installed package. Scripts are executed sequentially. If a script fails, a warning is printed but the install continues. Using `--skip-scripts` or setting `ignore_scripts = true` in the config skips this stage entirely.

### Post-install analysis

After install completes, you can run `manpm sensei` for a full project review: listing project files, checking config presence, scanning for vulnerabilities, and reporting `node_modules` status. `manpm entropy` calculates chaos metrics: duplicate package ratio, average dependency depth, circular dependency count.

## Using profiles

Profiles let you switch between sets of configuration presets. Define them in `manpm.config.toml`:

```toml
[profile.strict]
ui = "minimal"
safe_mode = true
version_strategy = "stable"

[profile.fast]
ui = "minimal"
version_strategy = "testing"
safe_mode = false
```

Activate a profile:

```
manpm profile use strict
```

Profiles are merged on top of the base configuration. Setting `safe_mode = true` in a profile overrides the base `behavior.safe_mode` value.

## Comparing packages

```
manpm compare express koa
```

Reads both packages' `package.json` files and displays a side-by-side comparison of version, license, description, homepage, and dependency counts.

## Running project scripts

```
manpm run <script>
```

MaNPM can execute any script defined in `package.json` under the `scripts` field. The `run` command looks up the script name and executes it via the shell. It supports the alias `manpm r <script>`.

Examples:

```
manpm run build
manpm run test
manpm r start
```

## Analyzing project health

```
manpm doctor
```

Scans the project and returns a score out of 100, with issues grouped by severity:

- **error** (-25 points): missing `node_modules`, cycle detected
- **warning** (-10 points): empty graph, empty `node_modules`
- **suggestion** (-3 points): large dependency tree (>500 packages)

## Running vulnerability scans

```
manpm audit
```

Reads `package-lock.json` and cross-references against a built-in advisory database. Currently covers known vulnerabilities in lodash, minimist, node-fetch, follow-redirects, and glob-parent. Packages already at the fixed version are skipped.

## Working with monorepos

Use per-project overrides in `manpm.config.toml` to customize behavior for specific projects within a monorepo:

```toml
[override.critical-app]
prefer_lockfile = true
version_strategy = "locked"
ignore_scripts = true

[override.legacy-app]
version_strategy = "stable"
```

## Platform-specific behavior

MaNPM detects the operating system (Linux, macOS, Windows, Android) and architecture (x86_64, ARM64, ARM, x86) at runtime.

- **Memory awareness**: On ARM devices with limited RAM, the worker count is capped to prevent out-of-memory conditions. RAM is detected by reading `/proc/meminfo` on Linux/Android.
- **Symlinks**: Enabled on POSIX systems. On Windows, symlinks are disabled unless `MANPM_FORCE_SYMLINKS` is set.
- **Temp directory**: On Android, the temp directory defaults to `/data/local/tmp/manpm` and can be overridden with `MANPM_TMPDIR`.
