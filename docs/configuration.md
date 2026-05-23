# Configuration

MaNPM reads a `manpm.config.toml` file from the project root. If the file does not exist, sensible defaults are used.

## File Location

```
<project-root>/manpm.config.toml
```

## Sections

### [core]

Controls parallelism and retry behavior.

| Field | Type | Default | Description |
|-------|------|---------|-------------|
| `parallel_limit` | int | 8 | Maximum number of concurrent package extractions |
| `auto_fix_peers` | bool | true | Automatically resolve peer dependency mismatches |
| `retry_count` | int | 3 | Number of retry attempts for failed downloads |

```toml
[core]
parallel_limit = 16
auto_fix_peers = false
retry_count = 5
```

### [behavior]

Runtime safety and maintenance behavior.

| Field | Type | Default | Description |
|-------|------|---------|-------------|
| `safe_mode` | bool | false | Skip risky operations (exec scripts, rebuilds) |
| `auto_prune` | bool | false | Automatically remove unused packages after install |
| `confirm_destructive` | bool | true | Prompt before destructive operations |

```toml
[behavior]
safe_mode = true
auto_prune = true
confirm_destructive = false
```

### [install_policy]

How packages are resolved and installed.

| Field | Type | Default | Description |
|-------|------|---------|-------------|
| `prefer_lockfile` | bool | true | Use lockfile resolutions over package.json ranges |
| `version_strategy` | string | `stable` | Version selection: `stable`, `testing`, `nightly`, `locked` |
| `ignore_scripts` | bool | false | Skip lifecycle scripts (install, postinstall) |
| `skip_binlink` | bool | false | Skip `.bin` symlink creation |

```toml
[install_policy]
prefer_lockfile = false
version_strategy = "locked"
ignore_scripts = true
skip_binlink = true
```

### [ui]

Display mode for terminal output.

| Field | Type | Default | Description |
|-------|------|---------|-------------|
| `mode` | string | `developer` | Output verbosity: `minimal`, `developer`, `psychotic` |

```toml
[ui]
mode = "minimal"
```

### [profile]

Select an active profile. Profiles can override any of the above settings.

| Field | Type | Default | Description |
|-------|------|---------|-------------|
| `active` | string | `""` | Name of the profile to apply |

```toml
[profile]
active = "strict"
```

### [profile.<name>]

Define a named profile that overrides base settings when activated. Profiles can set UI mode, strictness, version strategy, and safe mode.

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

### [override.<project>]

Per-project overrides for install policy. The `<project>` name is matched against the package name from `package.json`.

```toml
[override.critical-app]
prefer_lockfile = true
version_strategy = "locked"
ignore_scripts = true
skip_binlink = true
```

## Complete Example

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

## Defaults

If no config file exists, these values are used:

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
