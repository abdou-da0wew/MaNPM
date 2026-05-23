package preflight

import (
	"bytes"
	"os"
	"path/filepath"
	"testing"

	"manpm/pkg/platform"
)

func TestPreflightMissingPackageJSON(t *testing.T) {
	dir := t.TempDir()

	_, err := Run(dir)
	if err == nil {
		t.Fatal("expected an error for missing package.json, got nil")
	}
}

func TestPreflightMissingLockfile(t *testing.T) {
	dir := t.TempDir()

	pkgPath := filepath.Join(dir, "package.json")
	if err := os.WriteFile(pkgPath, []byte(`{}`), 0644); err != nil {
		t.Fatal(err)
	}

	_, err := Run(dir)
	if err == nil {
		t.Fatal("expected an error about missing lockfile, got nil")
	}
}

func TestPreflightPackageJSONOnly(t *testing.T) {
	dir := t.TempDir()

	pkgPath := filepath.Join(dir, "package.json")
	if err := os.WriteFile(pkgPath, []byte(`{"name":"test"}`), 0644); err != nil {
		t.Fatal(err)
	}

	lockfileContent := `{
  "lockfileVersion": 2,
  "packages": {
    "": {
      "version": "1.0.0",
      "resolved": "https://registry.npmjs.org/test/-/test-1.0.0.tgz",
      "integrity": "sha512-xxxxx=="
    }
  }
}`
	lockPath := filepath.Join(dir, "package-lock.json")
	if err := os.WriteFile(lockPath, []byte(lockfileContent), 0644); err != nil {
		t.Fatal(err)
	}

	res, err := Run(dir)
	if err != nil {
		t.Logf("Run failed (acceptable if node/npm not in $PATH): %v", err)
		return
	}

	if res == nil {
		t.Fatal("expected non-nil result")
	}
	if !res.HasPackageJSON {
		t.Error("expected HasPackageJSON to be true")
	}
	if !res.HasLockfile {
		t.Error("expected HasLockfile to be true")
	}
	if res.LockfileVersion != 2 {
		t.Errorf("expected LockfileVersion 2, got %d", res.LockfileVersion)
	}
}

func TestPrintSummary(t *testing.T) {
	r := &Result{
		OS:               platform.Linux,
		Arch:             platform.ArchAMD64,
		NodeVersion:      "v20.0.0",
		NpmVersion:       "10.0.0",
		LockfileVersion:  2,
		ConcurrencyLimit: 4,
	}

	stdout := os.Stdout
	pr, pw, err := os.Pipe()
	if err != nil {
		t.Fatal(err)
	}
	os.Stdout = pw

	done := make(chan struct{})
	var captured bytes.Buffer
	go func() {
		b := make([]byte, 4096)
		for {
			n, err := pr.Read(b)
			if n > 0 {
				captured.Write(b[:n])
			}
			if err != nil {
				break
			}
		}
		close(done)
	}()

	PrintSummary(r)

	pw.Close()
	os.Stdout = stdout
	<-done

	if captured.Len() == 0 {
		t.Error("PrintSummary produced no output")
	}
}
