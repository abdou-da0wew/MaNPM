# MaNPM — Agent instructions

## Build & test

```bash
# BUILD / TEST MUST be done from /tmp (fuseblk RLock incompatibility)
cp -a /storagesdcard/ManPM/* /tmp/manpm-src/
cd /tmp/manpm-src && go test ./...        # run all tests
cd /tmp/manpm-src && go vet ./...         # lint
cd /tmp/manpm-src && go build -ldflags="-s -w" -o /tmp/manpm ./cmd/manpm/  # release binary

# After any change, sync back:
cp -a /tmp/manpm-src/* /storagesdcard/ManPM/
```

## Remotes & push

- Origin: `git@github.com:abdou-da0wew/MaNPM.git` (SSH only, key at `~/.ssh/id_rsa`)
- Push: `git push origin main`

## Architecture

- Module: `manpm` (Go 1.26, stdlib only — NO external dependencies)
- Entrypoint: `cmd/manpm/main.go` — custom `buildRouter()`/`dispatch()` (no cobra/urfave)
- Packages: `pkg/{binlink,buildmgr,cache,config,extractor,graph,intel,lockfile,platform,preflight,ui}`
- Config: `manpm.config.toml` — hand-rolled TOML parser (no BurntSushi/toml)
- Palette: `pallet.json` — 9-color dark theme (Orange `#E35A00`, Golden `#FCA710`, Cyan `#6CD0E5`, Coral `#E24F44`)
- Binary: ~2.4MB stripped ARM64 at `/tmp/manpm`

## Commands

12 subcommands: `install`, `add`, `explain`, `audit`, `doctor`, `map`, `entropy`, `prune`, `sandbox`, `compare`, `sensei`, `profile`

## Testing quirks

- `pkg/intel` tests that call `Explain`/`Audit`/`Compare`/`Sensei` work against temp dirs (no live npm needed)
- `pkg/buildmgr` tests create mock `node_modules/` dirs with `binding.gyp` files
- `cmd/manpm` tests call `dispatch()` directly — `--help` handling is in `main()`, not `dispatch()`
- Tests produce ANSI output to stdout (expected, not a failure)

## Skills

28 skills registered at `~/.config/opencode/skills/`. Load via the `skill` tool when task matches. Key ones: `plan-master`, `caveman` (token-efficient mode), `cross-platform-scripts`, `prompt-engineer`, `skill-creator`.
