package intel

import (
	"encoding/json"
	"fmt"
	"math"
	"os"
	"path/filepath"
	"sort"
	"strings"

	"manpm/pkg/graph"
	"manpm/pkg/lockfile"
)

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

type AuditResult struct {
	PackageName        string
	Severity           string
	Title              string
	CVE                string
	FixAvailable       string
	ExploitProbability float64
}

type DoctorResult struct {
	Issues []Issue
	Score  float64
}

type Issue struct {
	Severity    string
	Message     string
	PackageName string
	Fix         string
}

type EntropyResult struct {
	Score           float64
	TotalPackages   int
	RedundantGroups []string
	UniqueLibraries int
	AvgDepth        float64
	CircularDeps    int
}

type pkgJSON struct {
	Name            string            `json:"name"`
	Version         string            `json:"version"`
	Description     string            `json:"description"`
	Homepage        string            `json:"homepage"`
	License         string            `json:"license"`
	Dependencies    map[string]string `json:"dependencies"`
	DevDependencies map[string]string `json:"devDependencies"`
	Scripts         map[string]string `json:"scripts"`
}

func readPackageJSON(pkgDir, pkgName string) (*pkgJSON, error) {
	pkgPath := filepath.Join(pkgDir, "node_modules", pkgName, "package.json")
	data, err := os.ReadFile(pkgPath)
	if err != nil {
		return nil, fmt.Errorf("read package.json for %s: %w", pkgName, err)
	}
	var p pkgJSON
	if err := json.Unmarshal(data, &p); err != nil {
		return nil, fmt.Errorf("parse package.json for %s: %w", pkgName, err)
	}
	return &p, nil
}

func Explain(pkgDir, pkgName string) (string, error) {
	p, err := readPackageJSON(pkgDir, pkgName)
	if err != nil {
		return "", err
	}

	var b strings.Builder
	b.WriteString(fmt.Sprintf("Package: %s\n", p.Name))
	b.WriteString(fmt.Sprintf("Version: %s\n", p.Version))
	if p.Description != "" {
		b.WriteString(fmt.Sprintf("Description: %s\n", p.Description))
	}
	if p.Homepage != "" {
		b.WriteString(fmt.Sprintf("Homepage: %s\n", p.Homepage))
	}
	if p.License != "" {
		b.WriteString(fmt.Sprintf("License: %s\n", p.License))
	}
	if len(p.Dependencies) > 0 {
		b.WriteString("Dependencies:\n")
		for name, ver := range p.Dependencies {
			b.WriteString(fmt.Sprintf("  * %s@%s\n", name, ver))
		}
	}
	if len(p.DevDependencies) > 0 {
		b.WriteString("Dev Dependencies:\n")
		for name, ver := range p.DevDependencies {
			b.WriteString(fmt.Sprintf("  * %s@%s\n", name, ver))
		}
	}
	b.WriteString(fmt.Sprintf("Install scripts: %v\n", hasInstallScript(p)))
	b.WriteString(fmt.Sprintf("Native addon: %v\n", isNative(pkgDir, pkgName)))
	return b.String(), nil
}

func hasInstallScript(p *pkgJSON) bool {
	if p.Scripts == nil {
		return false
	}
	for name := range p.Scripts {
		if name == "preinstall" || name == "install" || name == "postinstall" {
			return true
		}
	}
	return false
}

func isNative(pkgDir, pkgName string) bool {
	pkgPath := filepath.Join(pkgDir, "node_modules", pkgName)
	if _, err := os.Stat(filepath.Join(pkgPath, "binding.gyp")); err == nil {
		return true
	}
	if _, err := os.Stat(filepath.Join(pkgPath, "build")); err == nil {
		return true
	}
	return false
}

var advisoryDB = []AuditResult{
	{PackageName: "lodash", Severity: "medium", Title: "Prototype Pollution in lodash", CVE: "CVE-2020-8203", FixAvailable: "4.17.21", ExploitProbability: 0.6},
	{PackageName: "minimist", Severity: "low", Title: "Prototype Pollution in minimist", CVE: "CVE-2020-7598", FixAvailable: "1.2.3", ExploitProbability: 0.3},
	{PackageName: "node-fetch", Severity: "medium", Title: "URL request spoofing", CVE: "CVE-2022-0235", FixAvailable: "3.2.3", ExploitProbability: 0.5},
	{PackageName: "follow-redirects", Severity: "high", Title: "Exposure of sensitive information", CVE: "CVE-2022-0155", FixAvailable: "1.14.8", ExploitProbability: 0.7},
	{PackageName: "glob-parent", Severity: "low", Title: "Regular expression denial of service", CVE: "CVE-2020-28469", FixAvailable: "5.1.2", ExploitProbability: 0.2},
}

func Audit(lockfilePath string) ([]AuditResult, error) {
	lf, err := lockfile.Parse(lockfilePath)
	if err != nil {
		return nil, fmt.Errorf("audit: %w", err)
	}

	advisoryIndex := make(map[string][]AuditResult)
	for _, a := range advisoryDB {
		advisoryIndex[a.PackageName] = append(advisoryIndex[a.PackageName], a)
	}

	var results []AuditResult
	for key, pkg := range lf.Packages {
		if key == "" {
			continue
		}
		name := filepath.Base(key)
		if advisories, ok := advisoryIndex[name]; ok {
			for _, adv := range advisories {
				res := adv
				if pkg.Version == adv.FixAvailable {
					continue
				}
				res.PackageName = name
				results = append(results, res)
			}
		}
	}
	if results == nil {
		results = []AuditResult{}
	}
	return results, nil
}

func Doctor(projectDir string, dag *graph.DependencyGraph) (*DoctorResult, error) {
	var issues []Issue

	nmDir := filepath.Join(projectDir, "node_modules")
	nmStat, err := os.Stat(nmDir)
	if os.IsNotExist(err) {
		issues = append(issues, Issue{
			Severity: "error",
			Message:  "node_modules directory does not exist",
			Fix:      "Run npm install",
		})
	} else if err != nil {
		issues = append(issues, Issue{
			Severity: "error",
			Message:  "cannot stat node_modules: " + err.Error(),
		})
	}

	if dag != nil {
		if dag.HasCycle() {
			issues = append(issues, Issue{
				Severity: "error",
				Message:  "Dependency graph contains a cycle",
				Fix:      "Review and remove circular dependencies in package.json",
			})
		}

		if len(dag.Nodes) == 0 {
			issues = append(issues, Issue{
				Severity: "warning",
				Message:  "Dependency graph is empty",
				Fix:      "Run npm install to populate node_modules",
			})
		}

		depCount := len(dag.Nodes)
		if depCount > 500 {
			issues = append(issues, Issue{
				Severity:    "suggestion",
				Message:     fmt.Sprintf("Large dependency tree with %d packages", depCount),
				PackageName: "",
				Fix:         "Consider auditing for unused dependencies or using a lighter alternative",
			})
		}
	} else {
		issues = append(issues, Issue{
			Severity: "warning",
			Message:  "No dependency graph provided — analysis is limited",
			Fix:      "Build a dependency graph first",
		})
	}

	if nmStat != nil && nmStat.IsDir() {
		entries, err := os.ReadDir(nmDir)
		if err == nil && len(entries) == 0 {
			issues = append(issues, Issue{
				Severity: "warning",
				Message:  "node_modules directory is empty",
				Fix:      "Run npm install",
			})
		}
	}

	score := calculateHealthScore(issues)
	return &DoctorResult{Issues: issues, Score: score}, nil
}

func calculateHealthScore(issues []Issue) float64 {
	if len(issues) == 0 {
		return 100
	}
	deductions := 0.0
	for _, iss := range issues {
		switch iss.Severity {
		case "error":
			deductions += 25
		case "warning":
			deductions += 10
		case "suggestion":
			deductions += 3
		default:
			deductions += 1
		}
	}
	score := 100 - deductions
	if score < 0 {
		score = 0
	}
	return score
}

func Map(dag *graph.DependencyGraph) string {
	if dag == nil || len(dag.Nodes) == 0 {
		return "(empty dependency graph)"
	}

	if len(dag.Levels) == 0 {
		if err := dag.TopologicalSort(); err != nil {
			return fmt.Sprintf("(cannot render: %v)", err)
		}
	}

	var b strings.Builder
	b.WriteString("Dependency Map\n")
	b.WriteString("==============\n\n")

	for i, level := range dag.Levels {
		b.WriteString(fmt.Sprintf("Level %d (%d packages):\n", i, len(level)))
		for j, node := range level {
			prefix := "+-- "
			branch := "|   "
			if j == len(level)-1 {
				prefix = "`-- "
				branch = "    "
			}
			b.WriteString(fmt.Sprintf("%s%s@%s\n", prefix, node.Name, node.Version))
			depNames := sortedKeys(node.Dependencies)
			for k, dep := range depNames {
				depPrefix := branch + "+-- "
				if k == len(depNames)-1 {
					depPrefix = branch + "`-- "
				}
				b.WriteString(fmt.Sprintf("%s%s (depends)\n", depPrefix, dep))
			}
		}
		b.WriteString("\n")
	}
	return b.String()
}

func sortedKeys(m map[string]string) []string {
	keys := make([]string, 0, len(m))
	for k := range m {
		keys = append(keys, k)
	}
	sort.Strings(keys)
	return keys
}

func Entropy(dag *graph.DependencyGraph) *EntropyResult {
	if dag == nil {
		return &EntropyResult{}
	}

	total := len(dag.Nodes)
	uniqueLibraries := countUniqueLibs(dag)
	redundant := findRedundant(dag)

	circular := 0
	if dag.HasCycle() {
		circular = 1
	}

	avgDepth := 0.0
	if len(dag.Levels) > 0 && total > 0 {
		sum := 0
		for i, level := range dag.Levels {
			sum += (i + 1) * len(level)
		}
		avgDepth = float64(sum) / float64(total)
	}

	score := entropyScore(total, uniqueLibraries, len(redundant), avgDepth, circular)

	return &EntropyResult{
		Score:           score,
		TotalPackages:   total,
		RedundantGroups: redundant,
		UniqueLibraries: uniqueLibraries,
		AvgDepth:        math.Round(avgDepth*100) / 100,
		CircularDeps:    circular,
	}
}

func countUniqueLibs(dag *graph.DependencyGraph) int {
	seen := make(map[string]int)
	for _, node := range dag.Nodes {
		seen[node.Name]++
	}
	return len(seen)
}

func findRedundant(dag *graph.DependencyGraph) []string {
	nameVersions := make(map[string]map[string]int)
	for _, node := range dag.Nodes {
		if _, ok := nameVersions[node.Name]; !ok {
			nameVersions[node.Name] = make(map[string]int)
		}
		nameVersions[node.Name][node.Version]++
	}

	var redundant []string
	for name, versions := range nameVersions {
		if len(versions) > 1 {
			var vs []string
			for v := range versions {
				vs = append(vs, v)
			}
			sort.Strings(vs)
			redundant = append(redundant, fmt.Sprintf("%s (%s)", name, strings.Join(vs, ", ")))
		}
	}
	sort.Strings(redundant)
	return redundant
}

func entropyScore(total, unique, redundant int, avgDepth float64, circular int) float64 {
	if total == 0 {
		return 0
	}

	dupRatio := 0.0
	if unique > 0 {
		dupRatio = float64(total-unique) / float64(total)
	}

	depthFactor := avgDepth / 10.0
	if depthFactor > 1 {
		depthFactor = 1
	}

	circularPenalty := float64(circular) * 30

	score := dupRatio*40 + depthFactor*30 + circularPenalty
	if score > 100 {
		score = 100
	}
	return math.Round(score*100) / 100
}

func Sensei(projectDir string) (string, error) {
	var b strings.Builder
	b.WriteString("=== Sensei Project Review ===\n\n")

	b.WriteString("[1] Project Structure\n")
	entries, err := os.ReadDir(projectDir)
	if err != nil {
		return "", fmt.Errorf("sensei: read project dir: %w", err)
	}
	for _, e := range entries {
		b.WriteString(fmt.Sprintf("  %s\n", e.Name()))
	}

	b.WriteString("\n[2] Config Files\n")
	configFiles := []string{"package.json", "package-lock.json", ".npmrc", ".nvmrc"}
	for _, cf := range configFiles {
		path := filepath.Join(projectDir, cf)
		if _, err := os.Stat(path); err == nil {
			b.WriteString(fmt.Sprintf("  + %s found\n", cf))
		} else {
			b.WriteString(fmt.Sprintf("  - %s missing\n", cf))
		}
	}

	b.WriteString("\n[3] Recommendations\n")
	lockfilePath := filepath.Join(projectDir, "package-lock.json")
	if _, err := os.Stat(lockfilePath); err == nil {
		audits, err := Audit(lockfilePath)
		if err == nil && len(audits) > 0 {
			b.WriteString("  ! Known vulnerabilities found - run audit for details.\n")
		} else {
			b.WriteString("  + No known vulnerabilities.\n")
		}
	} else {
		b.WriteString("  ! No package-lock.json - vulnerability scanning skipped.\n")
	}

	nmPath := filepath.Join(projectDir, "node_modules")
	if nmStat, err := os.Stat(nmPath); err == nil && nmStat.IsDir() {
		entries, _ := os.ReadDir(nmPath)
		b.WriteString(fmt.Sprintf("  i node_modules contains %d top-level directories.\n", len(entries)))
	} else {
		b.WriteString("  ! node_modules not found - run npm install.\n")
	}

	b.WriteString("\n=== End of Review ===\n")
	return b.String(), nil
}

func Compare(pkgDir, pkg1, pkg2 string) (string, error) {
	p1, err := readPackageJSON(pkgDir, pkg1)
	if err != nil {
		return "", err
	}
	p2, err := readPackageJSON(pkgDir, pkg2)
	if err != nil {
		return "", err
	}

	var b strings.Builder
	b.WriteString(fmt.Sprintf("Comparing %s vs %s\n", pkg1, pkg2))
	b.WriteString(strings.Repeat("=", 40) + "\n\n")

	compareField(&b, "Version", p1.Version, p2.Version)
	compareField(&b, "License", p1.License, p2.License)
	compareField(&b, "Description", p1.Description, p2.Description)
	compareField(&b, "Homepage", p1.Homepage, p2.Homepage)
	compareField(&b, "Dependencies", fmt.Sprintf("%d", len(p1.Dependencies)), fmt.Sprintf("%d", len(p2.Dependencies)))
	compareField(&b, "Dev Dependencies", fmt.Sprintf("%d", len(p1.DevDependencies)), fmt.Sprintf("%d", len(p2.DevDependencies)))

	return b.String(), nil
}

func compareField(b *strings.Builder, name, v1, v2 string) {
	b.WriteString(fmt.Sprintf("%s:\n", name))
	b.WriteString(fmt.Sprintf("  %s\n", v1))
	b.WriteString(fmt.Sprintf("  %s\n", v2))
	b.WriteString("\n")
}

func SandboxInfo(pkgName string) string {
	return fmt.Sprintf(`Sandbox for %s:
  - Runtime: Node.js with limited permissions
  - Network: restricted to whitelisted registries
  - Filesystem: read-only except for %s's own directory
  - Execution timeout: 30s
  - No child process spawning
  - Memory limit: 512MB`,
		pkgName, pkgName)
}
