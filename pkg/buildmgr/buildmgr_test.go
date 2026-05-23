package buildmgr

import (
	"context"
	"os"
	"path/filepath"
	"testing"
)

func TestIsNativePackageWithGyp(t *testing.T) {
	dir := t.TempDir()
	gypPath := filepath.Join(dir, "binding.gyp")
	if err := os.WriteFile(gypPath, []byte("{}"), 0644); err != nil {
		t.Fatal(err)
	}
	if !IsNativePackage(dir) {
		t.Error("expected true when binding.gyp exists")
	}
}

func TestIsNativePackageWithoutGyp(t *testing.T) {
	dir := t.TempDir()
	if IsNativePackage(dir) {
		t.Error("expected false when no binding.gyp")
	}
}

func TestIsNativePackageWithGypfileFlag(t *testing.T) {
	dir := t.TempDir()
	pkgJSON := filepath.Join(dir, "package.json")
	content := `{"gypfile": true}`
	if err := os.WriteFile(pkgJSON, []byte(content), 0644); err != nil {
		t.Fatal(err)
	}
	if !IsNativePackage(dir) {
		t.Error("expected true when gypfile flag is set")
	}
}

func TestIsHeavyweightBinaryPkg(t *testing.T) {
	cases := []struct {
		name string
		want bool
	}{
		{"canvas", true},
		{"sharp", true},
		{"cypress", true},
		{"puppeteer", true},
		{"electron", true},
		{"playwright", true},
		{"esbuild", true},
		{"swc", true},
		{"lightningcss", true},
		{"lodash", false},
		{"express", false},
		{"react", false},
		{"Canvas", true},
		{"SHARP", true},
	}
	for _, c := range cases {
		got := IsHeavyweightBinaryPkg(c.name)
		if got != c.want {
			t.Errorf("IsHeavyweightBinaryPkg(%q) = %v, want %v", c.name, got, c.want)
		}
	}
}

func TestEnvForHeavyweightPkg(t *testing.T) {
	env := EnvForHeavyweightPkg("cypress")
	foundCypress := false
	for _, e := range env {
		if e == "CYPRESS_INSTALL_BINARY=0" {
			foundCypress = true
		}
	}
	if !foundCypress {
		t.Error("expected CYPRESS_INSTALL_BINARY=0 for cypress")
	}

	env = EnvForHeavyweightPkg("puppeteer")
	foundPuppeteer := false
	for _, e := range env {
		if e == "PUPPETEER_SKIP_DOWNLOAD=true" {
			foundPuppeteer = true
		}
	}
	if !foundPuppeteer {
		t.Error("expected PUPPETEER_SKIP_DOWNLOAD=true for puppeteer")
	}

	env = EnvForHeavyweightPkg("playwright")
	foundPlaywright := false
	for _, e := range env {
		if e == "PLAYWRIGHT_SKIP_BROWSER_DOWNLOAD=1" {
			foundPlaywright = true
		}
	}
	if !foundPlaywright {
		t.Error("expected PLAYWRIGHT_SKIP_BROWSER_DOWNLOAD=1 for playwright")
	}

	env = EnvForHeavyweightPkg("sharp")
	foundSharp := false
	for _, e := range env {
		if e == "SHARP_IGNORE_GLOBAL_LIBVIPS=1" {
			foundSharp = true
		}
	}
	if !foundSharp {
		t.Error("expected SHARP_IGNORE_GLOBAL_LIBVIPS=1 for sharp")
	}

	env = EnvForHeavyweightPkg("lodash")
	if len(env) != 0 {
		t.Errorf("expected no env vars for lodash, got %v", env)
	}
}

func TestNewBuildManager(t *testing.T) {
	bm := NewBuildManager("/test/dir")
	if bm.Dir != "/test/dir" {
		t.Errorf("expected Dir=/test/dir, got %q", bm.Dir)
	}
	if bm.NodesDir != "/test/dir/node_modules" {
		t.Errorf("expected NodesDir=/test/dir/node_modules, got %q", bm.NodesDir)
	}
}

func TestInspectPackageWithGyp(t *testing.T) {
	dir := t.TempDir()
	nodesDir := filepath.Join(dir, "node_modules", "test-pkg")
	os.MkdirAll(nodesDir, 0755)

	pkgJSON := `{"name":"test-pkg","scripts":{"install":"node-gyp rebuild"}}`
	os.WriteFile(filepath.Join(nodesDir, "package.json"), []byte(pkgJSON), 0644)

	gypPath := filepath.Join(nodesDir, "binding.gyp")
	os.WriteFile(gypPath, []byte("{}"), 0644)

	bm := NewBuildManager(dir)
	info, err := bm.inspectPackage("test-pkg")
	if err != nil {
		t.Fatalf("inspectPackage failed: %v", err)
	}
	if info == nil {
		t.Fatal("expected non-nil info")
	}
	if !info.HasGypFile {
		t.Error("expected HasGypFile=true")
	}
}

func TestInspectPackageWithInstallScript(t *testing.T) {
	dir := t.TempDir()
	nodesDir := filepath.Join(dir, "node_modules", "script-pkg")
	os.MkdirAll(nodesDir, 0755)

	pkgJSON := `{"name":"script-pkg","scripts":{"postinstall":"node build.js"}}`
	os.WriteFile(filepath.Join(nodesDir, "package.json"), []byte(pkgJSON), 0644)

	bm := NewBuildManager(dir)
	info, err := bm.inspectPackage("script-pkg")
	if err != nil {
		t.Fatalf("inspectPackage failed: %v", err)
	}
	if info == nil {
		t.Fatal("expected non-nil info")
	}
	if !info.HasInstallScript {
		t.Error("expected HasInstallScript=true")
	}
}

func TestInspectPackageNonNative(t *testing.T) {
	dir := t.TempDir()
	nodesDir := filepath.Join(dir, "node_modules", "plain-pkg")
	os.MkdirAll(nodesDir, 0755)

	pkgJSON := `{"name":"plain-pkg"}`
	os.WriteFile(filepath.Join(nodesDir, "package.json"), []byte(pkgJSON), 0644)

	bm := NewBuildManager(dir)
	info, err := bm.inspectPackage("plain-pkg")
	if err != nil {
		t.Fatalf("inspectPackage failed: %v", err)
	}
	if info != nil {
		t.Errorf("expected nil for non-native pkg, got %+v", info)
	}
}

func TestDetectNativePackages(t *testing.T) {
	dir := t.TempDir()
	nodesDir := filepath.Join(dir, "node_modules")
	os.MkdirAll(nodesDir, 0755)

	nativePkg := filepath.Join(nodesDir, "native-pkg")
	os.MkdirAll(nativePkg, 0755)
	os.WriteFile(filepath.Join(nativePkg, "package.json"), []byte(`{"name":"native-pkg","scripts":{"install":"gyp rebuild"}}`), 0644)
	os.WriteFile(filepath.Join(nativePkg, "binding.gyp"), []byte("{}"), 0644)

	plainPkg := filepath.Join(nodesDir, "plain-pkg")
	os.MkdirAll(plainPkg, 0755)
	os.WriteFile(filepath.Join(plainPkg, "package.json"), []byte(`{"name":"plain-pkg"}`), 0644)

	bm := NewBuildManager(dir)
	native, err := bm.DetectNativePackages(context.Background())
	if err != nil {
		t.Fatalf("DetectNativePackages failed: %v", err)
	}
	if len(native) != 1 {
		t.Fatalf("expected 1 native package, got %d: %+v", len(native), native)
	}
	if native[0].Name != "native-pkg" {
		t.Errorf("expected native-pkg, got %q", native[0].Name)
	}
}

func TestRebuildAllNoNative(t *testing.T) {
	dir := t.TempDir()
	nodesDir := filepath.Join(dir, "node_modules")
	os.MkdirAll(nodesDir, 0755)
	plainPkg := filepath.Join(nodesDir, "plain-pkg")
	os.MkdirAll(plainPkg, 0755)
	os.WriteFile(filepath.Join(plainPkg, "package.json"), []byte(`{"name":"plain-pkg"}`), 0644)

	bm := NewBuildManager(dir)
	err := bm.RebuildAll(context.Background())
	if err != nil {
		t.Errorf("expected no error for no native packages, got %v", err)
	}
}

func TestRebuildSequentialNoNative(t *testing.T) {
	dir := t.TempDir()
	nodesDir := filepath.Join(dir, "node_modules")
	os.MkdirAll(nodesDir, 0755)

	bm := NewBuildManager(dir)
	err := bm.RebuildSequential(context.Background())
	if err != nil {
		t.Errorf("expected no error for no native packages, got %v", err)
	}
}

func TestGetNodeVersion(t *testing.T) {
	v := getNodeVersion()
	if v == "" {
		t.Error("expected non-empty node version")
	}
}

func TestFindNodeGyp(t *testing.T) {
	ng := findNodeGyp()
	_ = ng
}

func TestRunInstallScripts(t *testing.T) {
	dir := t.TempDir()
	nodesDir := filepath.Join(dir, "node_modules")
	os.MkdirAll(nodesDir, 0755)
	bm := NewBuildManager(dir)
	err := bm.RunInstallScripts(context.Background())
	if err != nil {
		t.Errorf("expected no error, got %v", err)
	}
}
