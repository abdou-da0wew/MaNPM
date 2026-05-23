# API Reference

This documents every exported type and function across all `pkg/` packages.

---

## pkg/binlink

Binary symlink management for `node_modules/.bin`.

### type Linker

```go
type Linker struct {
    NodeModulesDir string
    BinDir         string
}
```

### func NewLinker(nodeModulesDir string) *Linker

Creates a Linker for the given `node_modules` directory. `BinDir` is set to `<nodeModulesDir>/.bin`.

### func (l *Linker) LinkPackage(pkgPath string) error

Reads the `bin` field from `<nodeModulesDir>/<pkgPath>/package.json` and creates symlinks (POSIX) or `.cmd` wrappers (Windows) in the `.bin` directory. Skips existing links with identical content. Scoped packages (`@scope/name`) have the scope stripped from the link name.

### func (l *Linker) LinkAllPackages(ctx context.Context, pkgPaths []string) error

Links multiple packages concurrently using a semaphore-limited goroutine pool (max 10). Returns a single error aggregating all failures.

### func (l *Linker) ReadPackageBin(pkgPath string) (map[string]string, error)

Reads the `bin` field from a package's `package.json`. Supports both string form (`"bin": "./cli.js"`) and map form (`"bin": {"cmd": "./cli.js"}`). Returns a map of binary name to relative target path.

---

## pkg/buildmgr

Native package detection and rebuild orchestration.

### type BuildManager

```go
type BuildManager struct {
    Dir      string
    NodesDir string
    Verbose  bool
}
```

### func NewBuildManager(dir string) *BuildManager

Creates a BuildManager at the given project directory. `NodesDir` is set to `<dir>/node_modules`.

### func (bm *BuildManager) DetectNativePackages(ctx context.Context) ([]NativePkgInfo, error)

Scans `node_modules/` and returns packages that have `binding.gyp`, `binding.gypi`, prebuild directories, node-gyp in their dependencies, install/postinstall scripts, a build directory, or the `gypfile` flag in `package.json`.

### func (bm *BuildManager) RebuildAll(ctx context.Context) error

Detects all native packages and rebuilds them in parallel. Returns an error listing any packages that failed.

### func (bm *BuildManager) RebuildSequential(ctx context.Context) error

Same as `RebuildAll` but processes packages one at a time. Useful for packages with shared build state.

### func (bm *BuildManager) RunInstallScripts(ctx context.Context) error

Finds native packages with a local node-gyp installation and runs `node-gyp rebuild` in each.

### type NativePkgInfo

```go
type NativePkgInfo struct {
    Name             string
    Path             string
    HasGypFile       bool
    HasPrebuild      bool
    HasNodeGyp       bool
    HasInstallScript bool
}
```

### func IsNativePackage(pkgDir string) bool

Returns true if the directory contains `binding.gyp` or its `package.json` has `gypfile: true`.

### func IsHeavyweightBinaryPkg(name string) bool

Returns true for packages known to have large binary downloads: canvas, sharp, cypress, puppeteer, electron, playwright, lmdb, re2, msgpackr-extract, esbuild, swc, lightningcss. Case-insensitive substring match.

### func EnvForHeavyweightPkg(name string) []string

Returns environment variables that suppress binary downloads for heavyweight packages: `CYPRESS_INSTALL_BINARY=0`, `PUPPETEER_SKIP_DOWNLOAD=true`, `PLAYWRIGHT_SKIP_BROWSER_DOWNLOAD=1`, `SHARP_IGNORE_GLOBAL_LIBVIPS=1`.

---

## pkg/cache

Persistent JSON metadata cache for package resolutions.

### type MetadataCache

```go
type MetadataCache struct {
    Entries map[string]*CacheEntry
}
```

### func NewMetadataCache(cacheDir string) (*MetadataCache, error)

Creates or loads a cache from `<cacheDir>/metadata.json`. Creates the directory if it does not exist.

### func (c *MetadataCache) Get(name string) *CacheEntry

Thread-safe read of a cache entry.

### func (c *MetadataCache) Set(name string, entry *CacheEntry)

Thread-safe write of a cache entry.

### func (c *MetadataCache) Save() error

Persists the cache to disk as JSON.

### type CacheEntry

```go
type CacheEntry struct {
    Weight    int64
    Resolved  string
    Integrity string
    CachedAt  time.Time
}
```

---

## pkg/config

TOML-based configuration loader and profile system.

### type Config

```go
type Config struct {
    Core     CoreConfig
    Behavior BehaviorConfig
    Install  InstallPolicy
    Profile  string
    UI       UIConfig
}
```

### func DefaultConfig() *Config

Returns a Config with all fields set to their default values.

### func LoadConfig(dir string) (*Config, error)

Reads `<dir>/manpm.config.toml` and parses it into a Config. If the file does not exist, returns DefaultConfig. Automatically applies the active profile if one is set.

### func (c *Config) Save(dir string) error

Serializes the Config to TOML format and writes it to `<dir>/manpm.config.toml`.

### func (c *Config) GetProfile(name string) *Profile

Returns the named profile, or nil if it does not exist.

### func (c *Config) ApplyProfile(name string) error

Applies the named profile's overrides to the Config. Returns an error if the profile does not exist.

### func (c *Config) GetEffectiveInstallPolicy(projectName string) InstallPolicy

Returns the install policy for the given project, checking project-specific overrides first, then falling back to the base policy.

### type CoreConfig

```go
type CoreConfig struct {
    ParallelLimit int
    AutoFixPeers  bool
    RetryCount    int
}
```

### type BehaviorConfig

```go
type BehaviorConfig struct {
    SafeMode           bool
    AutoPrune          bool
    ConfirmDestructive bool
}
```

### type InstallPolicy

```go
type InstallPolicy struct {
    PreferLockfile  bool
    VersionStrategy string
    IgnoreScripts   bool
    SkipBinLink     bool
}
```

### type UIConfig

```go
type UIConfig struct {
    Mode string
}
```

### type Profile

```go
type Profile struct {
    Name            string
    UI              string
    Strictness      string
    VersionStrategy string
    SafeMode        *bool
}
```

---

## pkg/extractor

Parallel tarball download, integrity verification, and extraction engine.

### type Extractor

```go
type Extractor struct {
    Client      *http.Client
    BaseDir     string
    NumWorkers  int
    Concurrency int
    FallbackDir string
    MaxRetries  int
}
```

### func NewExtractor(baseDir string, numWorkers int) *Extractor

Creates an Extractor with an HTTP transport tuned for high concurrency (`MaxIdleConnsPerHost`, `IdleConnTimeout`, `TLSHandshakeTimeout`, `ResponseHeaderTimeout`). If `numWorkers < 1`, uses `platform.DefaultConcurrencyLimit()`.

### func (e *Extractor) ExtractPackage(ctx context.Context, job PackageJob) error

Downloads a single tarball, verifies SHA512 integrity, extracts gzip/tar content to `node_modules/<job.Path>`, and sanitizes paths to prevent zip slip attacks. Cleans up the target directory on failure.

### func (e *Extractor) ExtractLevel(ctx context.Context, jobs []PackageJob) []ExtractResult

Extracts a batch of packages concurrently using a worker pool. Context cancellation aborts all in-flight extractions. Each failed extraction falls back to `npm install` for that specific package.

### type PackageJob

```go
type PackageJob struct {
    Name       string
    Path       string
    TarballURL string
    Integrity  string
}
```

### type ExtractResult

```go
type ExtractResult struct {
    PackageName string
    Error       error
}
```

### type ExtractError

```go
type ExtractError struct {
    Op  string
    Pkg string
    Err error
}
```

Implements `Error()` and `Unwrap()` for error wrapping.

---

## pkg/graph

Dependency graph construction, topological sort, and cycle detection.

### type DependencyGraph

```go
type DependencyGraph struct {
    Nodes  map[string]*PackageNode
    Levels [][]*PackageNode
}
```

### func NewDependencyGraph() *DependencyGraph

Returns an empty dependency graph.

### func (g *DependencyGraph) AddNode(name, version, resolved, integrity string, deps map[string]string)

Adds a package node. The internal key is `node_modules/<name>`.

### func (g *DependencyGraph) TopologicalSort() error

Performs Kahn's algorithm to topologically sort the graph into levels. Each level contains packages that can be processed in parallel. Returns `ErrCycleDetected` if a cycle exists.

### func (g *DependencyGraph) SmallestLastHeuristic(threshold int64) (lightweight [][]*PackageNode, heavyweight []*PackageNode)

Splits sorted levels into lightweight (weight <= threshold) and heavyweight groups. Within each level, packages are sorted by weight ascending.

### func (g *DependencyGraph) HasCycle() bool

DFS-based cycle detection using three-color marking (white/gray/black). Returns true if the graph contains a cycle.

### type PackageNode

```go
type PackageNode struct {
    Name         string
    Version      string
    Resolved     string
    Integrity    string
    Dependencies map[string]string
    Weight       int64
}
```

### var ErrCycleDetected = errors.New("dependency cycle detected")

---

## pkg/lockfile

package-lock.json v2/v3 parser.

### type LockfileV2

```go
type LockfileV2 struct {
    LockfileVersion int
    Packages        map[string]*PackageDef
}
```

### func Parse(path string) (*LockfileV2, error)

Reads and parses a package-lock.json file. Validates that the lockfile version is 2 or 3 and that packages is non-empty.

### func Validate(path string) error

Convenience wrapper around Parse that discards the result and returns only the error.

### func FindLockfile(dir string) (string, error)

Searches for `package-lock.json` in the given directory and returns its full path. Returns an error if not found.

### type PackageDef

```go
type PackageDef struct {
    Version      string
    Resolved     string
    Integrity    string
    Dependencies map[string]string
    Dev          bool
    Optional     bool
}
```

---

## pkg/platform

OS, architecture detection, and resource-aware concurrency limits.

### type OS

```go
type OS string

const (
    Windows OS = "windows"
    Linux   OS = "linux"
    Darwin  OS = "darwin"
    Android OS = "android"
)
```

### type Arch

```go
type Arch string

const (
    ArchARM   Arch = "arm"
    ArchARM64 Arch = "arm64"
    ArchAMD64 Arch = "amd64"
    Arch386   Arch = "386"
)
```

### func DetectOS() OS

Detects the operating system. On Linux, checks for `/system/build.prop` to distinguish Android.

### func DetectArch() Arch

Returns the CPU architecture from `runtime.GOARCH`.

### func IsARM() bool

Returns true if the architecture is ARM or ARM64.

### func IsAndroid() bool

Returns true if running on Android.

### func OptimalWorkerCount() int

Returns the optimal number of parallel workers based on CPU count and available RAM. On ARM devices with limited memory, caps the worker count to prevent OOM. Reads `/proc/meminfo` on Android for accurate RAM detection.

### func DefaultConcurrencyLimit() int

Returns `OptimalWorkerCount()` clamped to a minimum of 1.

### func SupportsSymlinks() bool

Returns true on POSIX systems. On Windows, returns true only if `MANPM_FORCE_SYMLINKS` is set.

### func TempDir() string

Returns a temporary directory. On Android, returns `/data/local/tmp/manpm` (or `MANPM_TMPDIR` if set).

---

## pkg/preflight

Pre-install validation.

### type Result

```go
type Result struct {
    HasPackageJSON   bool
    HasLockfile      bool
    LockfileVersion  int
    LockfilePath     string
    NodeInPATH       bool
    NpmInPATH        bool
    CanWriteNodeMods bool
    NodeVersion      string
    NpmVersion       string
    OS               platform.OS
    Arch             platform.Arch
    ConcurrencyLimit int
}
```

### func Run(dir string) (*Result, error)

Validates the project directory: checks for `package.json`, parses `package-lock.json`, verifies `node` and `npm` are in PATH with their versions, and confirms write access to `node_modules/`. Returns a Result struct with OS, arch, and concurrency limit.

### func PrintSummary(res *Result)

Prints a formatted summary of the preflight result to stdout.

---

## pkg/ui

Terminal output formatting with ANSI colors.

### Constants

```go
const (
    Reset   = "\033[0m"
    Bold    = "\033[1m"
    Red     = "\033[31m"
    Green   = "\033[32m"
    Yellow  = "\033[33m"
    Cyan    = "\033[36m"
    Magenta = "\033[35m"
    White   = "\033[97m"
    Orange  = "\033[38;5;208m"
    Gray    = "\033[90m"
)
```

### func Error(msg string)

Prints a red error message with ✖ prefix to stderr.

### func Errorf(format string, args ...interface{})

Formatted variant of Error.

### func Warning(msg string)

Prints a yellow warning message with ⚠ prefix to stderr.

### func Success(msg string)

Prints a green success message with ✓ prefix to stdout.

### func Info(msg string)

Prints a cyan info message with i prefix to stdout.

### func Header(title string)

Prints an orange-bordered header box around the title to stdout.

### func Subheader(title string)

Prints a cyan-prefixed subheader to stdout.

### func Label(k, v string)

Prints a gray key-value pair with aligned columns to stdout.

### func BoldText(s string) string

Returns the string wrapped in ANSI bold with Reset suffix.

### func Colorize(color, s string) string

Returns the string wrapped in the given ANSI color with Reset suffix.

### func Dim(s string) string

Returns the string wrapped in ANSI gray with Reset suffix.

### func Separator()

Prints a horizontal line of 50 `─` characters to stdout.

---

## pkg/intel

Project intelligence and analysis commands.

### type PackageInfo

```go
type PackageInfo struct {
    Name             string
    Version          string
    Resolved         string
    Integrity        string
    Dependencies     map[string]string
    DevDeps          map[string]string
    Description      string
    Homepage         string
    License          string
    HasInstallScript bool
    IsNative         bool
    Size             int64
    TransitiveCount  int
    InstalledBy      string
}
```

### type AuditResult

```go
type AuditResult struct {
    PackageName        string
    Severity           string
    Title              string
    CVE                string
    FixAvailable       string
    ExploitProbability float64
}
```

### type DoctorResult

```go
type DoctorResult struct {
    Issues []Issue
    Score  float64
}
```

### type Issue

```go
type Issue struct {
    Severity    string
    Message     string
    PackageName string
    Fix         string
}
```

### type EntropyResult

```go
type EntropyResult struct {
    Score           float64
    TotalPackages   int
    RedundantGroups []string
    UniqueLibraries int
    AvgDepth        float64
    CircularDeps    int
}
```

### func Explain(pkgDir, pkgName string) (string, error)

Reads the package's `package.json` from `node_modules/` and returns a formatted report with name, version, description, homepage, license, dependencies, dev dependencies, install script status, and native addon detection.

### func Audit(lockfilePath string) ([]AuditResult, error)

Reads a package-lock.json and cross-references it against a built-in advisory database. Returns matching vulnerabilities with severity, CVE, and fix version. Skips packages already at the fixed version.

### func Doctor(projectDir string, dag *graph.DependencyGraph) (*DoctorResult, error)

Analyzes project health. Checks for `node_modules` existence, dependency graph cycles, graph size, and empty directories. Returns a score out of 100 with actionable issues.

### func Map(dag *graph.DependencyGraph) string

Renders the dependency graph as an ASCII tree grouped by topological level. Each level shows packages with their immediate dependencies.

### func Entropy(dag *graph.DependencyGraph) *EntropyResult

Calculates project chaos metrics: duplicate package ratio, average dependency depth, circular dependency count, and redundant version groups. The score combines these into a 0-100 scale.

### func Sensei(projectDir string) (string, error)

Full project architecture review. Lists project files, checks for config files (package.json, package-lock.json, .npmrc, .nvmrc), runs vulnerability scan if a lockfile exists, and reports `node_modules` status.

### func Compare(pkgDir, pkg1, pkg2 string) (string, error)

Reads two packages' `package.json` files and returns a side-by-side comparison of version, license, description, homepage, and dependency counts.

### func SandboxInfo(pkgName string) string

Returns a formatted string describing the sandbox environment for a package: runtime, network restrictions, filesystem access, execution timeout, process spawning policy, and memory limit.
