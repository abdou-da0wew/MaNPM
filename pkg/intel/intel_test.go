package intel

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"manpm/pkg/graph"
)

func TestExplain(t *testing.T) {
	dir := t.TempDir()

	pkgDir := filepath.Join(dir, "node_modules", "test-pkg")
	if err := os.MkdirAll(pkgDir, 0755); err != nil {
		t.Fatal(err)
	}

	pkgJSON := `{
		"name": "test-pkg",
		"version": "1.2.3",
		"description": "A test package",
		"license": "MIT",
		"dependencies": {
			"left-pad": "^1.0.0"
		}
	}`
	if err := os.WriteFile(filepath.Join(pkgDir, "package.json"), []byte(pkgJSON), 0644); err != nil {
		t.Fatal(err)
	}

	result, err := Explain(dir, "test-pkg")
	if err != nil {
		t.Fatalf("Explain failed: %v", err)
	}

	if !strings.Contains(result, "test-pkg") {
		t.Errorf("expected package name in output, got: %s", result)
	}
	if !strings.Contains(result, "1.2.3") {
		t.Errorf("expected version in output, got: %s", result)
	}
	if !strings.Contains(result, "MIT") {
		t.Errorf("expected license in output, got: %s", result)
	}
	if !strings.Contains(result, "left-pad") {
		t.Errorf("expected dependency in output, got: %s", result)
	}
}

func TestExplainMissingPackage(t *testing.T) {
	dir := t.TempDir()
	_, err := Explain(dir, "nonexistent")
	if err == nil {
		t.Fatal("expected error for missing package")
	}
}

func TestMap(t *testing.T) {
	dag := graph.NewDependencyGraph()
	dag.AddNode("root", "1.0.0", "", "", map[string]string{"dep-a": "^1.0.0"})
	dag.AddNode("dep-a", "1.0.0", "", "", nil)

	if err := dag.TopologicalSort(); err != nil {
		t.Fatal(err)
	}

	out := Map(dag)
	if !strings.Contains(out, "root") {
		t.Errorf("expected root in map output, got: %s", out)
	}
	if !strings.Contains(out, "dep-a") {
		t.Errorf("expected dep-a in map output, got: %s", out)
	}
}

func TestMapEmpty(t *testing.T) {
	out := Map(nil)
	if !strings.Contains(out, "empty") {
		t.Errorf("expected empty message, got: %s", out)
	}

	dag := graph.NewDependencyGraph()
	out = Map(dag)
	if !strings.Contains(out, "empty") {
		t.Errorf("expected empty message, got: %s", out)
	}
}

func TestEntropy(t *testing.T) {
	dag := graph.NewDependencyGraph()
	result := Entropy(dag)
	if result.Score != 0 {
		t.Errorf("expected score 0 for empty graph, got %f", result.Score)
	}
	if result.TotalPackages != 0 {
		t.Errorf("expected 0 packages, got %d", result.TotalPackages)
	}
}

func TestEntropyWithPackages(t *testing.T) {
	dag := graph.NewDependencyGraph()
	dag.AddNode("a", "1.0.0", "", "", map[string]string{"b": "^1.0.0", "c": "^1.0.0"})
	dag.AddNode("b", "1.0.0", "", "", nil)
	dag.AddNode("c", "2.0.0", "", "", nil)

	if err := dag.TopologicalSort(); err != nil {
		t.Fatal(err)
	}

	result := Entropy(dag)
	if result.TotalPackages != 3 {
		t.Errorf("expected 3 packages, got %d", result.TotalPackages)
	}
	if result.UniqueLibraries != 3 {
		t.Errorf("expected 3 unique libraries, got %d", result.UniqueLibraries)
	}
}

func TestDoctor(t *testing.T) {
	dir := t.TempDir()
	dag := graph.NewDependencyGraph()

	result, err := Doctor(dir, dag)
	if err != nil {
		t.Fatalf("Doctor failed: %v", err)
	}

	if result.Score < 0 || result.Score > 100 {
		t.Errorf("score out of range: %f", result.Score)
	}

	foundWarning := false
	for _, iss := range result.Issues {
		if iss.Severity == "warning" {
			foundWarning = true
		}
	}
	if !foundWarning {
		t.Errorf("expected at least one warning (empty graph), issues: %+v", result.Issues)
	}
}

func TestDoctorWithCycle(t *testing.T) {
	dir := t.TempDir()
	dag := graph.NewDependencyGraph()
	dag.AddNode("a", "1.0", "", "", map[string]string{"b": "^1.0"})
	dag.AddNode("b", "1.0", "", "", map[string]string{"a": "^1.0"})

	result, err := Doctor(dir, dag)
	if err != nil {
		t.Fatalf("Doctor failed: %v", err)
	}

	foundCycle := false
	for _, iss := range result.Issues {
		if strings.Contains(iss.Message, "cycle") {
			foundCycle = true
		}
	}
	if !foundCycle {
		t.Errorf("expected cycle detection, issues: %+v", result.Issues)
	}
}

func TestSandboxInfo(t *testing.T) {
	info := SandboxInfo("test-pkg")
	if !strings.Contains(info, "test-pkg") {
		t.Errorf("expected package name in output, got: %s", info)
	}
	if !strings.Contains(info, "Sandbox") {
		t.Errorf("expected Sandbox header, got: %s", info)
	}
}

func TestCompare(t *testing.T) {
	dir := t.TempDir()

	pkg1Dir := filepath.Join(dir, "node_modules", "pkg-a")
	os.MkdirAll(pkg1Dir, 0755)
	os.WriteFile(filepath.Join(pkg1Dir, "package.json"), []byte(`{"name":"pkg-a","version":"1.0.0","license":"MIT"}`), 0644)

	pkg2Dir := filepath.Join(dir, "node_modules", "pkg-b")
	os.MkdirAll(pkg2Dir, 0755)
	os.WriteFile(filepath.Join(pkg2Dir, "package.json"), []byte(`{"name":"pkg-b","version":"2.0.0","license":"GPL-3.0"}`), 0644)

	result, err := Compare(dir, "pkg-a", "pkg-b")
	if err != nil {
		t.Fatalf("Compare failed: %v", err)
	}

	if !strings.Contains(result, "pkg-a") || !strings.Contains(result, "pkg-b") {
		t.Errorf("expected both package names in output, got: %s", result)
	}
	if !strings.Contains(result, "MIT") || !strings.Contains(result, "GPL-3.0") {
		t.Errorf("expected both licenses in output, got: %s", result)
	}
}

func TestSensei(t *testing.T) {
	dir := t.TempDir()

	os.WriteFile(filepath.Join(dir, "package.json"), []byte(`{"name":"test"}`), 0644)
	os.WriteFile(filepath.Join(dir, "package-lock.json"), []byte(`{"lockfileVersion":2,"packages":{}}`), 0644)

	result, err := Sensei(dir)
	if err != nil {
		t.Fatalf("Sensei failed: %v", err)
	}

	if !strings.Contains(result, "Sensei") {
		t.Errorf("expected Sensei header, got: %s", result)
	}
	if !strings.Contains(result, "package.json") {
		t.Errorf("expected package.json in output, got: %s", result)
	}
}

func TestAudit(t *testing.T) {
	dir := t.TempDir()
	lfPath := filepath.Join(dir, "package-lock.json")

	lfContent := `{
		"lockfileVersion": 2,
		"packages": {
			"": {"name": "test"},
			"node_modules/lodash": {
				"version": "4.17.20",
				"dependencies": {}
			}
		}
	}`
	if err := os.WriteFile(lfPath, []byte(lfContent), 0644); err != nil {
		t.Fatal(err)
	}

	results, err := Audit(lfPath)
	if err != nil {
		t.Fatalf("Audit failed: %v", err)
	}

	found := false
	for _, r := range results {
		if r.PackageName == "lodash" {
			found = true
			break
		}
	}
	if !found {
		t.Errorf("expected lodash in audit results, got %+v", results)
	}
}
