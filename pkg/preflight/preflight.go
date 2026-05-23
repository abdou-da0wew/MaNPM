package preflight

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"

	"manpm/pkg/lockfile"
	"manpm/pkg/platform"
)

type Result struct {
	HasPackageJSON    bool
	HasLockfile       bool
	LockfileVersion   int
	LockfilePath      string
	NodeInPATH        bool
	NpmInPATH         bool
	CanWriteNodeMods  bool
	NodeVersion       string
	NpmVersion        string
	OS                platform.OS
	Arch              platform.Arch
	ConcurrencyLimit  int
}

func Run(dir string) (*Result, error) {
	res := &Result{
		OS:   platform.DetectOS(),
		Arch: platform.DetectArch(),
	}

	if err := checkPackageJSON(dir, res); err != nil {
		return nil, err
	}

	if err := checkLockfile(dir, res); err != nil {
		return nil, err
	}

	if err := checkExecutables(res); err != nil {
		return nil, err
	}

	if err := checkPermissions(dir, res); err != nil {
		return nil, err
	}

	res.ConcurrencyLimit = platform.DefaultConcurrencyLimit()

	return res, nil
}

func checkPackageJSON(dir string, res *Result) error {
	path := filepath.Join(dir, "package.json")
	if _, err := os.Stat(path); os.IsNotExist(err) {
		return fmt.Errorf("no package.json found in %s", dir)
	}
	res.HasPackageJSON = true
	return nil
}

func checkLockfile(dir string, res *Result) error {
	path, err := lockfile.FindLockfile(dir)
	if err != nil {
		res.HasLockfile = false
		return fmt.Errorf("must run 'npm install' first to generate package-lock.json: %w", err)
	}

	lf, err := lockfile.Parse(path)
	if err != nil {
		return fmt.Errorf("invalid package-lock.json: %w", err)
	}

	res.HasLockfile = true
	res.LockfileVersion = lf.LockfileVersion
	res.LockfilePath = path
	return nil
}

func checkExecutables(res *Result) error {
	nodePath, err := exec.LookPath("node")
	if err != nil {
		return fmt.Errorf("node not found in $PATH")
	}
	res.NodeInPATH = true

	out, err := exec.Command(nodePath, "--version").Output()
	if err == nil {
		res.NodeVersion = string(out)
	}

	npmPath, err := exec.LookPath("npm")
	if err != nil {
		return fmt.Errorf("npm not found in $PATH")
	}
	res.NpmInPATH = true

	out, err = exec.Command(npmPath, "--version").Output()
	if err == nil {
		res.NpmVersion = string(out)
	}

	return nil
}

func checkPermissions(dir string, res *Result) error {
	testDir := filepath.Join(dir, "node_modules")
	if err := os.MkdirAll(testDir, 0755); err != nil {
		return fmt.Errorf("cannot create node_modules directory: %w", err)
	}

	tmpFile := filepath.Join(testDir, ".manpm_test")
	if err := os.WriteFile(tmpFile, []byte("test"), 0644); err != nil {
		return fmt.Errorf("cannot write to node_modules: %w", err)
	}
	os.Remove(tmpFile)

	res.CanWriteNodeMods = true
	return nil
}

func PrintSummary(res *Result) {
	fmt.Printf("  OS:              %s\n", res.OS)
	fmt.Printf("  Arch:            %s\n", res.Arch)
	fmt.Printf("  Node:            %s\n", strings.TrimSpace(res.NodeVersion))
	fmt.Printf("  npm:             %s\n", strings.TrimSpace(res.NpmVersion))
	fmt.Printf("  Lockfile:        v%d\n", res.LockfileVersion)
	fmt.Printf("  Concurrency:     %d\n", res.ConcurrencyLimit)
	fmt.Printf("  Symlinks:        %v\n", platform.SupportsSymlinks())
}
