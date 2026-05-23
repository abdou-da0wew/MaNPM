# SYSTEM DIRECTIVE: MANPM (Golang Parallel NPM Orchestrator)
**Role:** You are an elite, autonomous Systems Architect and Principal Golang Engineer.
**Objective:** Build `manpm`, a high-performance CLI tool written in Go that radically speeds up Node.js package installations. It acts as an advanced orchestrator that parses `package-lock.json`, topological sorts dependencies, streams tarballs directly to disk using goroutines, and defers compilation to standard `npm`.
**Language:** Go (1.21+). Use modern Go idioms, standard libraries where possible (`net/http`, `archive/tar`, `compress/gzip`, `crypto/sha512`), and lightweight CLI frameworks like `cobra` or `urfave/cli`.

## ARCHITECTURAL TRAPS TO AVOID (CRITICAL READ)
1. **The Arborist Race Condition:** Do NOT run multiple `npm install` processes in the same directory. You will implement a **Memory-Piped Extraction Engine** in Go. Read the tarball URL from the lockfile, download it via HTTP, and extract it directly into `node_modules/<pkg>` using Go's `archive/tar`, bypassing npm's tree builder entirely.
2. **Native Compilation Clashes:** Native C++ modules (`node-gyp`) will corrupt if built concurrently. You must skip all scripts during the Go-based extraction phase, and run a single, sequential `npm rebuild --foreground-scripts` at the very end.
3. **The `.bin` Symlink Trap:** Because you are extracting tarballs manually, CLI tool executables (like `tsc` or `vite`) won't be symlinked. You must manually parse the downloaded `package.json`'s `bin` field and generate correct cross-platform symlinks/cmd wrappers in `node_modules/.bin/`.
4. **Git/Local Dependencies:** If a dependency in the lockfile is a Git repository or local file path (not a standard registry tarball), your Go extractor must fall back to spawning an isolated `npm install <pkg> --cache=/tmp/manpm/<id> --no-audit --ignore-scripts`.

---

## EXECUTION PHASES & TASKS

### Phase 1: CLI Scaffolding & "Reality Checks"
1. **Initialize:** `go mod init manpm`. Set up the CLI entry point.
2. **Pre-Flight Reality Checks:** Write a `PreFlight()` function that:
   - Validates the current directory contains a `package.json`.
   - **Critical:** Validates the existence of `package-lock.json` v2 or v3. If missing, ABORT and instruct the user to run `npm install` to generate the lockfile deterministically.
   - Verifies `node` and `npm` executables are in the system `$PATH`.
   - Checks file system permissions for creating `./node_modules`.

### Phase 2: Lockfile Parsing & Corgi API Weight Estimation
1. **Parse Lockfile:** Unmarshal `package-lock.json` into a Go struct. Extract the `packages` map, capturing `version`, `resolved` (URL), `integrity`, and `dependencies`.
2. **Weight Estimation (Corgi API):**
   - Query the npm registry for package metadata using the `application/vnd.npm.install-v1+json` header to get the ultra-lightweight Corgi payload.
   - Calculate cumulative weight: $W(P_i) = S_{\text{unpacked}} + \text{UniqueTransitive}(P_i)$.
   - Cache these weights in a local SQLite or JSON cache (`~/.manpm/metadata.json`) to bypass network on future runs.

### Phase 3: DAG Generation & Topological Batching
1. **Graph Construction:** Build a Directed Acyclic Graph (DAG) of all packages.
2. **Level Formulation:** Perform a topological sort to group packages into discrete execution levels ($L_0, L_1, ... L_k$). Packages in $L_i$ only depend on packages in $L_0$ to $L_{i-1}$.
3. **Smallest-Last Heuristic:** Within each level, sort packages by cumulative weight ascending. 
   - Flag heavyweight packages ($W > \theta_{\text{size}}$) for isolated or dedicated worker queues.
   - Group lightweight packages into parallel worker channels.

### Phase 4: Memory-Piped Concurrency Engine (The Core)
1. **Worker Pool:** Create a dynamic worker pool of Goroutines based on `runtime.NumCPU()`.
2. **Direct Extraction Pipeline:** For each package in the current execution level:
   - Perform an HTTP GET on the `resolved` tarball URL.
   - Stream the response body (`io.Reader`) directly into Go's `sha512` hasher (to verify `integrity` matching the lockfile) AND simultaneously into `gzip.NewReader` $\rightarrow$ `tar.NewReader`.
   - Write the files directly to `./node_modules/<package-path>`.
   - Strip the leading `package/` folder from the tarball paths during extraction.
3. **Fallback:** If extraction fails, gracefully catch the error and spawn an isolated npm fallback: `exec.Command("npm", "install", pkgName, "--cache="+tmpCache, "--ignore-scripts", "--no-package-lock")`.

### Phase 5: Bin Linking & Lifecycle Deferral
1. **Symlink Generation:** After all levels are extracted, scan the extracted directories for `package.json` `bin` fields. Create relative symlinks in `node_modules/.bin/` (and `.cmd` wrappers for Windows).
2. **Rebuild Execution:** Execute `npm rebuild --foreground-scripts` via `os/exec` to compile native bindings (`node-gyp`) and trigger post-install hooks.
3. **Binary Hooks:** Execute `CYPRESS_INSTALL_BINARY=0` logic if specific heavyweight packages are detected during extraction, deferring their binary downloads to this final phase.

### Phase 6: Comprehensive Testing Suite
1. **Unit Tests:**
   - Test the DAG and Topological Sort logic using a mock dependency tree. Ensure cyclic dependencies throw clear errors.
   - Test the Corgi API weight calculation.
2. **Integration Tests (`httptest`):**
   - Spin up a local `httptest.Server` acting as a mock npm registry serving dummy `.tgz` files.
   - Test the tarball streaming, gzip extraction, and integrity verification.
3. **E2E Reality Test:** 
   - Create a temporary directory with a dummy `package.json` and `package-lock.json` containing 5 real packages (e.g., `is-even`, `chalk`, `lodash`).
   - Run the Manpm binary and verify `node_modules` is populated correctly, `.bin` contains executables, and a test script can `require()` them.

---

## STRICT AGENT RULES
* **Goroutine Safety:** Use `sync.RWMutex` or channels when writing to shared structures. Avoid race conditions.
* **OS Pathing:** Use `path/filepath` universally. Windows uses `\` and POSIX uses `/`. Tarballs *always* use `/`. You must translate tarball paths to OS paths safely to prevent Zip Slip vulnerabilities.
* **Error Handling:** Never swallow errors. If a Goroutine panics or an HTTP request fails, the worker pool must cleanly cancel (use `context.Context` with cancellation) and report the exact package failure.
* **Sub-Agents:** Spawn separate agent threads for the **DAG/Graph logic**, the **Tarball Extraction Engine**, and the **Symlink/Bin Engine**. 

Begin immediately by scaffolding the `go.mod`, directory structure (`cmd`, `pkg/graph`, `pkg/extractor`), and writing the Phase 1 Pre-Flight Reality Checks.

