package lockfile

import (
	"os"
	"path/filepath"
	"testing"
)

func writeTestLockfile(t *testing.T, dir string, content string) string {
	t.Helper()
	path := filepath.Join(dir, "package-lock.json")
	if err := os.WriteFile(path, []byte(content), 0644); err != nil {
		t.Fatal(err)
	}
	return path
}

func TestParseV2(t *testing.T) {
	dir := t.TempDir()
	path := writeTestLockfile(t, dir, `{
		"lockfileVersion": 2,
		"packages": {
			"": {"version": "1.0.0", "resolved": "https://example.com/pkg.tgz", "integrity": "sha512-abc"},
			"node_modules/foo": {"version": "2.0.0", "resolved": "https://example.com/foo.tgz", "integrity": "sha512-def"}
		}
	}`)

	lf, err := Parse(path)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if lf.LockfileVersion != 2 {
		t.Errorf("expected version 2, got %d", lf.LockfileVersion)
	}
	if len(lf.Packages) != 2 {
		t.Errorf("expected 2 packages, got %d", len(lf.Packages))
	}
	if lf.Packages[""].Version != "1.0.0" {
		t.Errorf("expected root version 1.0.0, got %s", lf.Packages[""].Version)
	}
}

func TestParseV3(t *testing.T) {
	dir := t.TempDir()
	path := writeTestLockfile(t, dir, `{
		"lockfileVersion": 3,
		"packages": {
			"": {"version": "0.0.0", "resolved": "https://example.com/pkg.tgz", "integrity": "sha512-xyz"}
		}
	}`)

	lf, err := Parse(path)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if lf.LockfileVersion != 3 {
		t.Errorf("expected version 3, got %d", lf.LockfileVersion)
	}
}

func TestParseInvalidVersion(t *testing.T) {
	dir := t.TempDir()
	path := writeTestLockfile(t, dir, `{
		"lockfileVersion": 1,
		"packages": {
			"": {"version": "1.0.0", "resolved": "", "integrity": ""}
		}
	}`)

	_, err := Parse(path)
	if err == nil {
		t.Fatal("expected error for version 1, got nil")
	}
}

func TestParseEmptyPackages(t *testing.T) {
	dir := t.TempDir()
	path := writeTestLockfile(t, dir, `{
		"lockfileVersion": 2,
		"packages": {}
	}`)

	_, err := Parse(path)
	if err == nil {
		t.Fatal("expected error for empty packages, got nil")
	}
}

func TestFindLockfile(t *testing.T) {
	dir := t.TempDir()
	content := `{"lockfileVersion": 2, "packages": {"": {"version": "1.0.0", "resolved": "", "integrity": ""}}}`
	if err := os.WriteFile(filepath.Join(dir, "package-lock.json"), []byte(content), 0644); err != nil {
		t.Fatal(err)
	}

	found, err := FindLockfile(dir)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if found != filepath.Join(dir, "package-lock.json") {
		t.Errorf("expected %s, got %s", filepath.Join(dir, "package-lock.json"), found)
	}
}

func TestFindLockfileNotFound(t *testing.T) {
	dir := t.TempDir()
	_, err := FindLockfile(dir)
	if err == nil {
		t.Fatal("expected error when no lockfile exists")
	}
}

func TestValidate(t *testing.T) {
	dir := t.TempDir()
	path := writeTestLockfile(t, dir, `{
		"lockfileVersion": 2,
		"packages": {
			"": {"version": "1.0.0", "resolved": "", "integrity": ""}
		}
	}`)

	if err := Validate(path); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestValidateInvalid(t *testing.T) {
	dir := t.TempDir()
	path := writeTestLockfile(t, dir, `{
		"lockfileVersion": 1,
		"packages": {
			"": {"version": "1.0.0", "resolved": "", "integrity": ""}
		}
	}`)

	if err := Validate(path); err == nil {
		t.Fatal("expected error for invalid lockfile")
	}
}
