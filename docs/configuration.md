# Configuration

MaNPM reads settings from `manpm.config.toml` in the project root. If the file does not exist, defaults are used.

## File location

```
<project-root>/manpm.config.toml
```

## Sections

### [core]

Control parallelism and retry behavior.

```toml
[core]
parallel_limit = 8
auto_fix_peers = true
retry_count = 3
```

| Field | Type | Default | Description |
|-------|------|---------|-------------|
| `parallel_limit` | int | 8 | Maximum concurrent package extractions |
| `auto_fix_peers` | bool | true | Auto-resolve peer dependency mismatches |
| `retry_count` | int | 3 | Retry attempts for failed downloads |

### [behavior]

Runtime safety and maintenance.

```toml
[behavior]
safe_mode = false
auto_prune = false
confirm_destructive = true
```

| Field | Type | Default | Description |
|-------|------|---------|-------------|
| `safe_mode` | bool | false | Skip risky operations (exec scripts, rebuilds) |
| `auto_prune` | bool | false | Auto-remove unused packages after install |
| `confirm_destructive` | bool | true | Prompt before destructive operations |

### [install_policy]

Package resolution and installation.

```toml
[install_policy]
prefer_lockfile = true
version_strategy = "stable"
ignore_scripts = false
skip_binlink = false
```

| Field | Type | Default | Description |
|-------|------|---------|-------------|
| `prefer_lockfile` | bool | true | Use lockfile over package.json ranges |
| `version_strategy` | string | `stable` | `stable`, `testing`, `nightly`, `locked` |
| `ignore_scripts` | bool | false | Skip lifecycle scripts |
| `skip_binlink` | bool | false | Skip `.bin` symlink creation |

### [ui]

Terminal output verbosity.

```toml
[ui]
mode = "developer"
```

| Field | Type | Default | Description |
|-------|------|---------|-------------|
| `mode` | string | `developer` | `minimal`, `developer`, or `psychotic` |

### [profile]

Select an active profile.

```toml
[profile]
active = "strict"
```

| Field | Type | Default | Description |
|-------|------|---------|-------------|
| `active` | string | `""` | Name of the profile to activate |

### [profile.<name>]

Define a named profile that overrides base settings.

```toml
[profile.strict]
ui = "minimal"
safe_mode = true
version_strategy = "stable"

[profile.fast]
ui = "minimal"
safe_mode = false
```

| Field | Type | Description |
|-------|------|-------------|
| `ui` | string | Override UI mode |
| `safe_mode` | bool | Override safe mode |
| `strictness` | string | `safe` enables safe mode |
| `version_strategy` | string | Override version strategy |

### [override.<project>]

Per-project install policy overrides. The project name is matched against the `name` field in `package.json`.

```toml
[override.critical-app]
prefer_lockfile = true
version_strategy = "locked"
ignore_scripts = true
skip_binlink = true
```

## Complete example

```toml
[core]
parallel_limit = 12
auto_fix_peers = true
retry_count = 3

[behavior]
safe_mode = false
auto_prune = true
confirm_destructive = true

[install_policy]
prefer_lockfile = true
version_strategy = "stable"
ignore_scripts = false
skip_binlink = false

[ui]
mode = "developer"

[profile]
active = "strict"

[profile.strict]
ui = "minimal"
safe_mode = true
version_strategy = "stable"

[override.critical-app]
prefer_lockfile = true
version_strategy = "locked"
```

## Default values

If no config file is present, these values apply:

| Field | Default |
|-------|---------|
| core.parallel_limit | 8 |
| core.auto_fix_peers | true |
| core.retry_count | 3 |
| behavior.safe_mode | false |
| behavior.auto_prune | false |
| behavior.confirm_destructive | true |
| install_policy.prefer_lockfile | true |
| install_policy.version_strategy | stable |
| install_policy.ignore_scripts | false |
| install_policy.skip_binlink | false |
| ui.mode | developer |
