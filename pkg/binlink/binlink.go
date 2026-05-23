package binlink

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"sync"
)

type Linker struct {
	NodeModulesDir string
	BinDir         string
}

func NewLinker(nodeModulesDir string) *Linker {
	return &Linker{
		NodeModulesDir: nodeModulesDir,
		BinDir:         filepath.Join(nodeModulesDir, ".bin"),
	}
}

func (l *Linker) LinkPackage(pkgPath string) error {
	bins, err := l.ReadPackageBin(pkgPath)
	if err != nil {
		return err
	}

	if err := os.MkdirAll(l.BinDir, 0755); err != nil {
		return err
	}

	for name, target := range bins {
		linkPath := filepath.Join(l.BinDir, name)
		if err := os.MkdirAll(filepath.Dir(linkPath), 0755); err != nil {
			return err
		}
		relTarget := filepath.Join("..", pkgPath, target)
		relTarget = filepath.Clean(relTarget)

		if runtime.GOOS == "windows" {
			linkPath += ".cmd"
			content := fmt.Sprintf(`@echo off
"%%~dp0%s" %%*
`, relTarget)
			existing, err := os.ReadFile(linkPath)
			if err == nil && string(existing) == content {
				continue
			}
			if err := os.WriteFile(linkPath, []byte(content), 0755); err != nil {
				return err
			}
		} else {
			existing, err := os.Readlink(linkPath)
			if err == nil && existing == relTarget {
				continue
			}
			if err := os.Remove(linkPath); err != nil && !os.IsNotExist(err) {
				return err
			}
			if err := os.Symlink(relTarget, linkPath); err != nil {
				return err
			}
		}
	}

	return nil
}

func (l *Linker) LinkAllPackages(ctx context.Context, pkgPaths []string) error {
	var wg sync.WaitGroup
	sem := make(chan struct{}, 10)
	errCh := make(chan error, len(pkgPaths))

	for _, pkg := range pkgPaths {
		pkg := pkg
		wg.Add(1)
		go func() {
			defer wg.Done()
			select {
			case sem <- struct{}{}:
			case <-ctx.Done():
				errCh <- ctx.Err()
				return
			}
			defer func() { <-sem }()

			if err := l.LinkPackage(pkg); err != nil {
				errCh <- fmt.Errorf("%s: %w", pkg, err)
			}
		}()
	}

	wg.Wait()
	close(errCh)

	var errs []error
	for err := range errCh {
		errs = append(errs, err)
	}
	if len(errs) > 0 {
		return fmt.Errorf("errors linking packages: %v", errs)
	}
	return nil
}

func stripScope(name string) string {
	if strings.HasPrefix(name, "@") {
		parts := strings.SplitN(name, "/", 2)
		if len(parts) == 2 {
			return parts[1]
		}
	}
	return name
}

func (l *Linker) ReadPackageBin(pkgPath string) (map[string]string, error) {
	pkgJSONPath := filepath.Join(l.NodeModulesDir, pkgPath, "package.json")
	data, err := os.ReadFile(pkgJSONPath)
	if err != nil {
		return nil, err
	}

	var raw struct {
		Bin json.RawMessage `json:"bin"`
		Name string         `json:"name"`
	}
	if err := json.Unmarshal(data, &raw); err != nil {
		return nil, err
	}

	if raw.Bin == nil {
		return nil, nil
	}

	bins := make(map[string]string)

	var binStr string
	if err := json.Unmarshal(raw.Bin, &binStr); err == nil {
		name := raw.Name
		if name == "" {
			name = filepath.Base(pkgPath)
		}
		name = stripScope(name)
		bins[name] = binStr
		return bins, nil
	}

	var binMap map[string]string
	if err := json.Unmarshal(raw.Bin, &binMap); err != nil {
		return nil, fmt.Errorf("bin field must be a string or map")
	}
	for name, target := range binMap {
		bins[name] = target
	}

	return bins, nil
}
