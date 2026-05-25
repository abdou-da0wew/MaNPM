package pkgjson

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestReadPackageJSON(t *testing.T) {
	dir := t.TempDir()
	content := `{
  "name": "test-pkg",
  "version": "1.0.0",
  "dependencies": {
    "express": "^4.0.0"
  }
}`
	if err := os.WriteFile(filepath.Join(dir, "package.json"), []byte(content), 0644); err != nil {
		t.Fatal(err)
	}

	pkg, err := ReadPackageJSON(dir)
	if err != nil {
		t.Fatal(err)
	}
	if pkg.Name != "test-pkg" {
		t.Errorf("Name = %q, want %q", pkg.Name, "test-pkg")
	}
	if pkg.Version != "1.0.0" {
		t.Errorf("Version = %q, want %q", pkg.Version, "1.0.0")
	}
	if v := pkg.Dependencies["express"]; v != "^4.0.0" {
		t.Errorf("Dependencies[express] = %q, want %q", v, "^4.0.0")
	}
}

func TestReadPackageJSONNotFound(t *testing.T) {
	dir := t.TempDir()
	_, err := ReadPackageJSON(dir)
	if err == nil {
		t.Fatal("expected error for missing package.json")
	}
}

func TestWritePackageJSON(t *testing.T) {
	dir := t.TempDir()
	pkg := &PackageJSON{
		Name:    "test-pkg",
		Version: "1.0.0",
		Dependencies: map[string]string{
			"express": "^4.0.0",
		},
	}
	if err := WritePackageJSON(dir, pkg); err != nil {
		t.Fatal(err)
	}

	data, err := os.ReadFile(filepath.Join(dir, "package.json"))
	if err != nil {
		t.Fatal(err)
	}
	if !strings.HasSuffix(string(data), "\n") {
		t.Error("expected trailing newline")
	}

	got, err := ReadPackageJSON(dir)
	if err != nil {
		t.Fatal(err)
	}
	if got.Name != pkg.Name {
		t.Errorf("Name = %q, want %q", got.Name, pkg.Name)
	}
	if got.Version != pkg.Version {
		t.Errorf("Version = %q, want %q", got.Version, pkg.Version)
	}
	if v := got.Dependencies["express"]; v != "^4.0.0" {
		t.Errorf("Dependencies[express] = %q, want %q", v, "^4.0.0")
	}
}

func TestLockVersionsAll(t *testing.T) {
	pkg := &PackageJSON{
		Dependencies:    map[string]string{"express": "^4.0.0", "lodash": "~1.2.3"},
		DevDependencies: map[string]string{"mocha": "^5.0.0"},
		PeerDependencies: map[string]string{"react": "~16.0.0"},
	}
	count := LockVersions(pkg, nil, false)
	if count != 4 {
		t.Errorf("count = %d, want 4", count)
	}
	tests := []struct {
		got, want string
		label     string
	}{
		{pkg.Dependencies["express"], "4.0.0", "Dependencies[express]"},
		{pkg.Dependencies["lodash"], "1.2.3", "Dependencies[lodash]"},
		{pkg.DevDependencies["mocha"], "5.0.0", "DevDependencies[mocha]"},
		{pkg.PeerDependencies["react"], "16.0.0", "PeerDependencies[react]"},
	}
	for _, tt := range tests {
		if tt.got != tt.want {
			t.Errorf("%s = %q, want %q", tt.label, tt.got, tt.want)
		}
	}
}

func TestLockVersionsSpecific(t *testing.T) {
	pkg := &PackageJSON{
		Dependencies: map[string]string{"express": "^4.0.0", "lodash": "~1.2.3"},
	}
	count := LockVersions(pkg, []string{"express"}, false)
	if count != 1 {
		t.Errorf("count = %d, want 1", count)
	}
	if v := pkg.Dependencies["express"]; v != "4.0.0" {
		t.Errorf("Dependencies[express] = %q, want %q", v, "4.0.0")
	}
	if v := pkg.Dependencies["lodash"]; v != "~1.2.3" {
		t.Errorf("Dependencies[lodash] = %q, want %q", v, "~1.2.3")
	}
}

func TestLockVersionsMajorOnly(t *testing.T) {
	pkg := &PackageJSON{
		Dependencies: map[string]string{"express": "^4.0.0", "lodash": "~1.2.3"},
	}
	count := LockVersions(pkg, nil, true)
	if count != 1 {
		t.Errorf("count = %d, want 1", count)
	}
	if v := pkg.Dependencies["express"]; v != "^4.0.0" {
		t.Errorf("Dependencies[express] = %q, want %q", v, "^4.0.0")
	}
	if v := pkg.Dependencies["lodash"]; v != "1.2.3" {
		t.Errorf("Dependencies[lodash] = %q, want %q", v, "1.2.3")
	}
}

func TestLockVersionsNoCaretTilde(t *testing.T) {
	pkg := &PackageJSON{
		Dependencies: map[string]string{"express": "4.0.0"},
	}
	count := LockVersions(pkg, nil, false)
	if count != 0 {
		t.Errorf("count = %d, want 0", count)
	}
	if v := pkg.Dependencies["express"]; v != "4.0.0" {
		t.Errorf("Dependencies[express] = %q, want %q", v, "4.0.0")
	}
}

func TestDetectMismatches(t *testing.T) {
	pkg := &PackageJSON{
		Dependencies:    map[string]string{"express": "^4.0.0", "lodash": "^1.0.0"},
		DevDependencies: map[string]string{"express": "^5.0.0", "mocha": "^6.0.0"},
	}
	mismatches := DetectMismatches(pkg)
	if len(mismatches) != 1 {
		t.Fatalf("got %d mismatches, want 1", len(mismatches))
	}
	m := mismatches[0]
	if m.Name != "express" {
		t.Errorf("Name = %q, want %q", m.Name, "express")
	}
	if len(m.Versions) != 2 {
		t.Errorf("got %d versions, want 2: %v", len(m.Versions), m.Versions)
	}
}

func TestDetectMismatchesNone(t *testing.T) {
	pkg := &PackageJSON{
		Dependencies: map[string]string{"express": "^4.0.0"},
	}
	mismatches := DetectMismatches(pkg)
	if len(mismatches) != 0 {
		t.Errorf("got %d mismatches, want 0", len(mismatches))
	}
}

func TestFixMismatches(t *testing.T) {
	pkg := &PackageJSON{
		Dependencies:    map[string]string{"express": "^4.0.0"},
		DevDependencies: map[string]string{"express": "^5.0.0"},
	}
	count := FixMismatches(pkg, "highest")
	if count != 1 {
		t.Errorf("count = %d, want 1", count)
	}
	if v := pkg.Dependencies["express"]; v != "^5.0.0" {
		t.Errorf("Dependencies[express] = %q, want %q", v, "^5.0.0")
	}
	if v := pkg.DevDependencies["express"]; v != "^5.0.0" {
		t.Errorf("DevDependencies[express] = %q, want %q", v, "^5.0.0")
	}
}
