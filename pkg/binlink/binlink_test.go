package binlink

import (
	"context"
	"os"
	"path/filepath"
	"runtime"
	"testing"
)

func TestReadPackageBin(t *testing.T) {
	dir := t.TempDir()
	nmDir := filepath.Join(dir, "node_modules")
	pkgDir := filepath.Join(nmDir, "mypkg")
	if err := os.MkdirAll(pkgDir, 0755); err != nil {
		t.Fatal(err)
	}

	l := NewLinker(nmDir)

	t.Run("bin as string", func(t *testing.T) {
		pkg := filepath.Join(pkgDir, "package.json")
		if err := os.WriteFile(pkg, []byte(`{"name":"mypkg","bin":"./cli.js"}`), 0644); err != nil {
			t.Fatal(err)
		}
		bins, err := l.ReadPackageBin("mypkg")
		if err != nil {
			t.Fatal(err)
		}
		if len(bins) != 1 || bins["mypkg"] != "./cli.js" {
			t.Errorf("unexpected bins: %v", bins)
		}
	})

	t.Run("bin as map", func(t *testing.T) {
		pkg := filepath.Join(pkgDir, "package.json")
		if err := os.WriteFile(pkg, []byte(`{"name":"mypkg","bin":{"mycli":"./cli.js","mycmd":"./cmd.js"}}`), 0644); err != nil {
			t.Fatal(err)
		}
		bins, err := l.ReadPackageBin("mypkg")
		if err != nil {
			t.Fatal(err)
		}
		if len(bins) != 2 || bins["mycli"] != "./cli.js" || bins["mycmd"] != "./cmd.js" {
			t.Errorf("unexpected bins: %v", bins)
		}
	})

	t.Run("no bin field", func(t *testing.T) {
		pkg := filepath.Join(pkgDir, "package.json")
		if err := os.WriteFile(pkg, []byte(`{"name":"mypkg"}`), 0644); err != nil {
			t.Fatal(err)
		}
		bins, err := l.ReadPackageBin("mypkg")
		if err != nil {
			t.Fatal(err)
		}
		if len(bins) != 0 {
			t.Errorf("expected empty bins, got %v", bins)
		}
	})

	t.Run("missing package.json", func(t *testing.T) {
		_, err := l.ReadPackageBin("nonexistent")
		if err == nil {
			t.Error("expected error for missing package.json")
		}
	})
}

func TestLinkPackage(t *testing.T) {
	dir := t.TempDir()
	nmDir := filepath.Join(dir, "node_modules")
	pkgDir := filepath.Join(nmDir, "mypkg")
	if err := os.MkdirAll(pkgDir, 0755); err != nil {
		t.Fatal(err)
	}

	cliPath := filepath.Join(pkgDir, "cli.js")
	if err := os.WriteFile(cliPath, []byte("#!/usr/bin/env node\nconsole.log('hello')"), 0644); err != nil {
		t.Fatal(err)
	}

	pkg := filepath.Join(pkgDir, "package.json")
	if err := os.WriteFile(pkg, []byte(`{"name":"mypkg","bin":"./cli.js"}`), 0644); err != nil {
		t.Fatal(err)
	}

	l := NewLinker(nmDir)
	if err := l.LinkPackage("mypkg"); err != nil {
		t.Fatal(err)
	}

	binDir := filepath.Join(nmDir, ".bin")
	if runtime.GOOS == "windows" {
		linkPath := filepath.Join(binDir, "mypkg.cmd")
		if _, err := os.Stat(linkPath); err != nil {
			t.Errorf("expected .cmd file: %v", err)
		}
	} else {
		linkPath := filepath.Join(binDir, "mypkg")
		target, err := os.Readlink(linkPath)
		if err != nil {
			t.Fatal(err)
		}
		expected := filepath.Clean(filepath.Join("..", "mypkg", "./cli.js"))
		if target != expected {
			t.Errorf("expected symlink target %q, got %q", expected, target)
		}
	}
}

func TestLinkAllPackages(t *testing.T) {
	dir := t.TempDir()
	nmDir := filepath.Join(dir, "node_modules")
	l := NewLinker(nmDir)

	pkgs := []string{"pkg-a", "pkg-b", "pkg-c"}
	for _, p := range pkgs {
		pkgDir := filepath.Join(nmDir, p)
		if err := os.MkdirAll(pkgDir, 0755); err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(filepath.Join(pkgDir, "cli.js"), []byte("#!/usr/bin/env node\n"), 0644); err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(filepath.Join(pkgDir, "package.json"), []byte(`{"name":"`+p+`","bin":"./cli.js"}`), 0644); err != nil {
			t.Fatal(err)
		}
	}

	if err := l.LinkAllPackages(context.Background(), pkgs); err != nil {
		t.Fatal(err)
	}

	for _, p := range pkgs {
		linkPath := filepath.Join(nmDir, ".bin", p)
		if runtime.GOOS == "windows" {
			linkPath += ".cmd"
		}
		if _, err := os.Stat(linkPath); err != nil {
			t.Errorf("expected link for %s: %v", p, err)
		}
	}
}
