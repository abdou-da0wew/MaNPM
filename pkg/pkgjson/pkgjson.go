package pkgjson

import (
	"encoding/json"
	"os"
	"os/exec"
	"path/filepath"
	"sort"
	"strings"
)

type PackageJSON struct {
	Name             string            `json:"name,omitempty"`
	Version          string            `json:"version,omitempty"`
	Dependencies     map[string]string `json:"dependencies,omitempty"`
	DevDependencies  map[string]string `json:"devDependencies,omitempty"`
	PeerDependencies map[string]string `json:"peerDependencies,omitempty"`
}

type Mismatch struct {
	Name     string
	Versions []string
	Groups   []string
}

type UpdateResult struct {
	Name    string
	Current string
	Wanted  string
	Latest  string
}

func ReadPackageJSON(dir string) (*PackageJSON, error) {
	data, err := os.ReadFile(filepath.Join(dir, "package.json"))
	if err != nil {
		return nil, err
	}
	var pkg PackageJSON
	if err := json.Unmarshal(data, &pkg); err != nil {
		return nil, err
	}
	return &pkg, nil
}

func WritePackageJSON(dir string, pkg *PackageJSON) error {
	data, err := json.MarshalIndent(pkg, "", "  ")
	if err != nil {
		return err
	}
	data = append(data, '\n')
	return os.WriteFile(filepath.Join(dir, "package.json"), data, 0644)
}

func LockVersions(pkg *PackageJSON, names []string, majorOnly bool) int {
	count := 0
	lockMap := func(m map[string]string) {
		for name, ver := range m {
			if len(names) > 0 && !contains(names, name) {
				continue
			}
			orig := ver
			if majorOnly {
				ver = strings.TrimPrefix(ver, "~")
			} else {
				ver = strings.TrimPrefix(ver, "^")
				ver = strings.TrimPrefix(ver, "~")
			}
			if ver != orig {
				m[name] = ver
				count++
			}
		}
	}
	if pkg.Dependencies != nil {
		lockMap(pkg.Dependencies)
	}
	if pkg.DevDependencies != nil {
		lockMap(pkg.DevDependencies)
	}
	if pkg.PeerDependencies != nil {
		lockMap(pkg.PeerDependencies)
	}
	return count
}

func contains(slice []string, s string) bool {
	for _, v := range slice {
		if v == s {
			return true
		}
	}
	return false
}

func DetectMismatches(pkg *PackageJSON) []Mismatch {
	type entry struct {
		version string
		group   string
	}
	collected := map[string][]entry{}

	for name, ver := range pkg.Dependencies {
		collected[name] = append(collected[name], entry{ver, "dependencies"})
	}
	for name, ver := range pkg.DevDependencies {
		collected[name] = append(collected[name], entry{ver, "devDependencies"})
	}
	for name, ver := range pkg.PeerDependencies {
		collected[name] = append(collected[name], entry{ver, "peerDependencies"})
	}

	var mismatches []Mismatch
	for name, entries := range collected {
		if len(entries) < 2 {
			continue
		}
		verSet := map[string]bool{}
		for _, e := range entries {
			verSet[e.version] = true
		}
		if len(verSet) <= 1 {
			continue
		}

		groupSet := map[string]bool{}
		seenVer := map[string]bool{}
		var versions []string
		for _, e := range entries {
			groupSet[e.group] = true
			if !seenVer[e.version] {
				versions = append(versions, e.version)
				seenVer[e.version] = true
			}
		}

		sort.Strings(versions)

		var groups []string
		for g := range groupSet {
			groups = append(groups, g)
		}
		sort.Strings(groups)

		mismatches = append(mismatches, Mismatch{
			Name:     name,
			Versions: versions,
			Groups:   groups,
		})
	}

	sort.Slice(mismatches, func(i, j int) bool {
		return mismatches[i].Name < mismatches[j].Name
	})

	return mismatches
}

func FixMismatches(pkg *PackageJSON, strategy string) int {
	_ = strategy
	mismatches := DetectMismatches(pkg)
	count := 0

	setVersion := func(m map[string]string, name, ver string) {
		if m == nil {
			return
		}
		if _, ok := m[name]; ok && m[name] != ver {
			m[name] = ver
			count++
		}
	}

	for _, m := range mismatches {
		target := m.Versions[len(m.Versions)-1]
		for _, g := range m.Groups {
			switch g {
			case "dependencies":
				setVersion(pkg.Dependencies, m.Name, target)
			case "devDependencies":
				setVersion(pkg.DevDependencies, m.Name, target)
			case "peerDependencies":
				setVersion(pkg.PeerDependencies, m.Name, target)
			}
		}
	}

	return count
}

func CheckOutdated(dir string, names []string) ([]UpdateResult, error) {
	cmd := exec.Command("npm", "outdated", "--json")
	cmd.Dir = dir
	out, err := cmd.Output()
	if err != nil {
		if len(out) == 0 {
			return nil, nil
		}
	}

	var raw map[string]struct {
		Current string `json:"current"`
		Wanted  string `json:"wanted"`
		Latest  string `json:"latest"`
	}
	if err := json.Unmarshal(out, &raw); err != nil {
		return nil, nil
	}

	var results []UpdateResult
	for name, pkg := range raw {
		if len(names) > 0 && !contains(names, name) {
			continue
		}
		results = append(results, UpdateResult{
			Name:    name,
			Current: pkg.Current,
			Wanted:  pkg.Wanted,
			Latest:  pkg.Latest,
		})
	}

	sort.Slice(results, func(i, j int) bool {
		return results[i].Name < results[j].Name
	})

	return results, nil
}
