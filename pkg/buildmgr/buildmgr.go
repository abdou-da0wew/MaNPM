package buildmgr

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
)

type BuildManager struct {
	Dir        string
	NodesDir   string
	Verbose    bool
}

type NativePkgInfo struct {
	Name       string
	Path       string
	HasGypFile bool
	HasPrebuild bool
	HasNodeGyp bool
	HasInstallScript bool
}

func NewBuildManager(dir string) *BuildManager {
	return &BuildManager{
		Dir:      dir,
		NodesDir: filepath.Join(dir, "node_modules"),
	}
}

func (bm *BuildManager) DetectNativePackages(ctx context.Context) ([]NativePkgInfo, error) {
	entries, err := os.ReadDir(bm.NodesDir)
	if err != nil {
		return nil, fmt.Errorf("read node_modules: %w", err)
	}

	var native []NativePkgInfo

	for _, entry := range entries {
		if !entry.IsDir() {
			continue
		}
		if strings.HasPrefix(entry.Name(), ".") {
			continue
		}

		info, err := bm.inspectPackage(entry.Name())
		if err != nil {
			continue
		}
		if info != nil {
			native = append(native, *info)
		}
	}

	return native, nil
}

func (bm *BuildManager) inspectPackage(name string) (*NativePkgInfo, error) {
	pkgDir := filepath.Join(bm.NodesDir, name)
	pkgJSON := filepath.Join(pkgDir, "package.json")

	data, err := os.ReadFile(pkgJSON)
	if err != nil {
		return nil, err
	}

	var pkg struct {
		Name          string `json:"name"`
		Scripts       map[string]string `json:"scripts"`
		Gypfile       bool   `json:"gypfile"`
	}
	if err := json.Unmarshal(data, &pkg); err != nil {
		return nil, err
	}

	info := &NativePkgInfo{
		Name: name,
		Path: name,
	}

	gypPath := filepath.Join(pkgDir, "binding.gyp")
	if _, err := os.Stat(gypPath); err == nil {
		info.HasGypFile = true
	}

	gyp2Path := filepath.Join(pkgDir, "binding.gypi")
	if _, err := os.Stat(gyp2Path); err == nil {
		info.HasGypFile = true
	}

	prebuildDir := filepath.Join(pkgDir, "prebuilds")
	if _, err := os.Stat(prebuildDir); err == nil {
		info.HasPrebuild = true
	}

	nodeGypDir := filepath.Join(pkgDir, "node_modules", "node-gyp")
	if _, err := os.Stat(nodeGypDir); err == nil {
		info.HasNodeGyp = true
	}

	for scriptName := range pkg.Scripts {
		if scriptName == "install" || scriptName == "postinstall" {
			info.HasInstallScript = true
		}
	}

	if info.HasGypFile || info.HasNodeGyp || info.HasInstallScript {
		return info, nil
	}

	buildDir := filepath.Join(pkgDir, "build")
	if _, err := os.Stat(buildDir); err == nil {
		return info, nil
	}

	if pkg.Gypfile {
		info.HasGypFile = true
		return info, nil
	}

	return nil, nil
}

func (bm *BuildManager) RebuildAll(ctx context.Context) error {
	native, err := bm.DetectNativePackages(ctx)
	if err != nil {
		return fmt.Errorf("detect native packages: %w", err)
	}

	if len(native) == 0 {
		return nil
	}

	var failed []string
	for _, pkg := range native {
		if err := bm.rebuildSingle(ctx, pkg); err != nil {
			failed = append(failed, pkg.Name)
		}
	}

	if len(failed) > 0 {
		return fmt.Errorf("rebuild failed for: %s", strings.Join(failed, ", "))
	}
	return nil
}

func (bm *BuildManager) RebuildSequential(ctx context.Context) error {
	native, err := bm.DetectNativePackages(ctx)
	if err != nil {
		return fmt.Errorf("detect native packages: %w", err)
	}

	if len(native) == 0 {
		return nil
	}

	var failed []string
	for _, pkg := range native {
		if err := bm.rebuildSingle(ctx, pkg); err != nil {
			failed = append(failed, pkg.Name)
		}
	}

	if len(failed) > 0 {
		return fmt.Errorf("sequential rebuild failed for: %s", strings.Join(failed, ", "))
	}
	return nil
}

func (bm *BuildManager) rebuildSingle(ctx context.Context, pkg NativePkgInfo) error {
	if err := bm.tryPrebuild(ctx, pkg); err == nil {
		return nil
	}

	if err := bm.tryNodeGypBuild(ctx, pkg); err == nil {
		return nil
	}

	if err := bm.tryNpmRebuild(ctx, pkg); err == nil {
		return nil
	}

	return bm.tryBuildFromSource(ctx, pkg)
}

func (bm *BuildManager) tryPrebuild(ctx context.Context, pkg NativePkgInfo) error {
	for _, tool := range []string{"prebuild-install", "node-pre-gyp", "@mapbox/node-pre-gyp"} {
		cmd := exec.CommandContext(ctx, "npx", "--yes", tool, "--runtime", "node", "--target", getNodeVersion())
		cmd.Dir = filepath.Join(bm.NodesDir, pkg.Path)
		var out bytes.Buffer
		cmd.Stdout = &out
		cmd.Stderr = &out

		if err := cmd.Run(); err == nil {
			return nil
		}
	}
	return fmt.Errorf("no prebuilt binary found")
}

func (bm *BuildManager) tryNodeGypBuild(ctx context.Context, pkg NativePkgInfo) error {
	nodeGyp := findNodeGyp()
	if nodeGyp == "" {
		return fmt.Errorf("node-gyp not found")
	}

	cmd := exec.CommandContext(ctx, nodeGyp, "rebuild")
	cmd.Dir = filepath.Join(bm.NodesDir, pkg.Path)
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr

	return cmd.Run()
}

func (bm *BuildManager) tryNpmRebuild(ctx context.Context, pkg NativePkgInfo) error {
	cmd := exec.CommandContext(ctx, "npm", "rebuild", pkg.Name, "--foreground-scripts")
	cmd.Dir = bm.Dir
	var out bytes.Buffer
	cmd.Stdout = &out
	cmd.Stderr = &out

	return cmd.Run()
}

func (bm *BuildManager) tryBuildFromSource(ctx context.Context, pkg NativePkgInfo) error {
	cmd := exec.CommandContext(ctx, "npm", "rebuild", pkg.Name, "--build-from-source", "--foreground-scripts")
	cmd.Dir = bm.Dir
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr

	return cmd.Run()
}

func (bm *BuildManager) RunInstallScripts(ctx context.Context) error {
	native, err := bm.DetectNativePackages(ctx)
	if err != nil {
		return err
	}

	for _, pkg := range native {
		script := filepath.Join(bm.NodesDir, pkg.Path, "node_modules", ".bin", "node-gyp")
		if _, err := os.Stat(script); err != nil {
			continue
		}

		cmd := exec.CommandContext(ctx, script, "rebuild")
		cmd.Dir = filepath.Join(bm.NodesDir, pkg.Path)
		cmd.Stdout = os.Stdout
		cmd.Stderr = os.Stderr
		cmd.Run()
	}

	return nil
}

func findNodeGyp() string {
	paths := []string{
		"node-gyp",
		filepath.Join(os.Getenv("HOME"), ".node-gyp"),
	}

	for _, p := range paths {
		if _, err := exec.LookPath(p); err == nil {
			return p
		}
	}

	npmRoot, err := exec.Command("npm", "root", "-g").Output()
	if err == nil {
		ngPath := filepath.Join(strings.TrimSpace(string(npmRoot)), "node-gyp", "bin", "node-gyp.js")
		if _, err := os.Stat(ngPath); err == nil {
			return "node " + ngPath
		}
	}

	return ""
}

func getNodeVersion() string {
	out, err := exec.Command("node", "-e", "console.log(process.version.slice(1))").Output()
	if err != nil {
		return "20.0.0"
	}
	return strings.TrimSpace(string(out))
}

func IsNativePackage(pkgDir string) bool {
	gypPath := filepath.Join(pkgDir, "binding.gyp")
	if _, err := os.Stat(gypPath); err == nil {
		return true
	}

	pkgJSON := filepath.Join(pkgDir, "package.json")
	data, err := os.ReadFile(pkgJSON)
	if err != nil {
		return false
	}

	var pkg struct {
		Gypfile bool `json:"gypfile"`
	}
	if err := json.Unmarshal(data, &pkg); err != nil {
		return false
	}

	return pkg.Gypfile
}

func IsHeavyweightBinaryPkg(name string) bool {
	heavy := []string{
		"canvas",
		"sharp",
		"cypress",
		"puppeteer",
		"chromium",
		"electron",
		"playwright",
		"lmdb",
		"re2",
		"msgpackr-extract",
		"esbuild",
		"swc",
		"lightningcss",
	}

	nameLower := strings.ToLower(name)
	for _, h := range heavy {
		if strings.Contains(nameLower, h) {
			return true
		}
	}
	return false
}

func EnvForHeavyweightPkg(name string) []string {
	nameLower := strings.ToLower(name)
	var env []string

	if strings.Contains(nameLower, "cypress") {
		env = append(env, "CYPRESS_INSTALL_BINARY=0")
	}
	if strings.Contains(nameLower, "puppeteer") || strings.Contains(nameLower, "chromium") {
		env = append(env, "PUPPETEER_SKIP_DOWNLOAD=true")
		env = append(env, "PUPPETEER_SKIP_CHROMIUM_DOWNLOAD=true")
	}
	if strings.Contains(nameLower, "playwright") {
		env = append(env, "PLAYWRIGHT_SKIP_BROWSER_DOWNLOAD=1")
	}
	if strings.Contains(nameLower, "sharp") {
		env = append(env, "SHARP_IGNORE_GLOBAL_LIBVIPS=1")
	}

	return env
}
