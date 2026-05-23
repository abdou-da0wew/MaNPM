package lockfile

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
)

type LockfileV2 struct {
	LockfileVersion int                    `json:"lockfileVersion"`
	Packages        map[string]*PackageDef `json:"packages"`
}

type PackageDef struct {
	Version      string            `json:"version"`
	Resolved     string            `json:"resolved"`
	Integrity    string            `json:"integrity"`
	Dependencies map[string]string `json:"dependencies"`
	Dev          bool              `json:"dev,omitempty"`
	Optional     bool              `json:"optional,omitempty"`
}

func Parse(path string) (*LockfileV2, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("read lockfile: %w", err)
	}

	var lf LockfileV2
	if err := json.Unmarshal(data, &lf); err != nil {
		return nil, fmt.Errorf("parse lockfile: %w", err)
	}

	if lf.LockfileVersion < 2 || lf.LockfileVersion > 3 {
		return nil, fmt.Errorf("unsupported lockfile version %d (need v2 or v3)", lf.LockfileVersion)
	}

	if len(lf.Packages) == 0 {
		return nil, fmt.Errorf("lockfile has no packages entry")
	}

	return &lf, nil
}

func Validate(path string) error {
	_, err := Parse(path)
	return err
}

func FindLockfile(dir string) (string, error) {
	candidates := []string{
		filepath.Join(dir, "package-lock.json"),
	}

	for _, path := range candidates {
		if _, err := os.Stat(path); err == nil {
			return path, nil
		}
	}

	return "", fmt.Errorf("no package-lock.json found in %s", dir)
}
