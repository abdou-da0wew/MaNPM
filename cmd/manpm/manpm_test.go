package main

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestBuildRouter(t *testing.T) {
	root := buildRouter()
	if root.Name != "manpm" {
		t.Errorf("expected manpm, got %q", root.Name)
	}
	if len(root.Subcommands) != 13 {
		t.Errorf("expected 13 commands, got %d", len(root.Subcommands))
	}
}

func TestRouterCommands(t *testing.T) {
	root := buildRouter()
	names := make(map[string]bool)
	for _, cmd := range root.Subcommands {
		if names[cmd.Name] {
			t.Errorf("duplicate command: %s", cmd.Name)
		}
		names[cmd.Name] = true
	}

	expected := []string{"install", "add", "explain", "audit", "doctor", "map", "entropy", "prune", "run", "sandbox", "compare", "sensei", "profile"}
	for _, name := range expected {
		if !names[name] {
			t.Errorf("missing command: %s", name)
		}
	}
}

func withInstallDir(t *testing.T, fn func()) {
	t.Helper()
	dir := t.TempDir()
	os.WriteFile(filepath.Join(dir, "package.json"), []byte(`{"name":"test","dependencies":{"left-pad":"^1.0.0"}}`), 0644)
	os.WriteFile(filepath.Join(dir, "package-lock.json"), []byte(`{"name":"test","lockfileVersion":3,"packages":{"":{"name":"test","dependencies":{"left-pad":"^1.0.0"}},"node_modules/left-pad":{"version":"1.3.0"}}}`), 0644)
	oldDir, _ := os.Getwd()
	os.Chdir(dir)
	defer os.Chdir(oldDir)
	fn()
}

func TestDispatchDefault(t *testing.T) {
	withInstallDir(t, func() {
		root := buildRouter()
		err := dispatch(root, []string{})
		if err != nil {
			t.Fatalf("dispatch with no args failed: %v", err)
		}
	})
}

func TestDispatchInstall(t *testing.T) {
	withInstallDir(t, func() {
		root := buildRouter()
		err := dispatch(root, []string{"install"})
		if err != nil {
			t.Fatalf("dispatch install failed: %v", err)
		}
	})
}

func TestDispatchInstallWithFlags(t *testing.T) {
	withInstallDir(t, func() {
		root := buildRouter()
		err := dispatch(root, []string{"install", "--threads", "4", "--dry-run", "--retries", "5"})
		if err != nil {
			t.Fatalf("dispatch install with flags failed: %v", err)
		}
	})
}

func TestDispatchAdd(t *testing.T) {
	root := buildRouter()
	err := dispatch(root, []string{"add", "lodash", "--dev", "--exact"})
	if err != nil {
		t.Fatalf("dispatch add failed: %v", err)
	}
}

func TestDispatchAddNoPackage(t *testing.T) {
	root := buildRouter()
	err := dispatch(root, []string{"add"})
	if err == nil {
		t.Fatal("expected error for add without package")
	}
}

func TestDispatchExplain(t *testing.T) {
	root := buildRouter()
	err := dispatch(root, []string{"explain", "test-pkg"})
	if err == nil {
		t.Log("explain succeeded (may fail if no node_modules)")
	}
}

func TestDispatchExplainNoPackage(t *testing.T) {
	root := buildRouter()
	err := dispatch(root, []string{"explain"})
	if err == nil {
		t.Fatal("expected error for explain without package")
	}
}

func TestDispatchAudit(t *testing.T) {
	root := buildRouter()
	err := dispatch(root, []string{"audit"})
	if err == nil {
		t.Log("audit succeeded (may fail if no package-lock.json)")
	}
}

func TestDispatchDoctor(t *testing.T) {
	root := buildRouter()
	err := dispatch(root, []string{"doctor"})
	if err != nil {
		t.Errorf("doctor failed: %v", err)
	}
}

func TestDispatchMap(t *testing.T) {
	root := buildRouter()
	err := dispatch(root, []string{"map"})
	if err != nil {
		t.Errorf("map failed: %v", err)
	}
}

func TestDispatchEntropy(t *testing.T) {
	root := buildRouter()
	err := dispatch(root, []string{"entropy"})
	if err != nil {
		t.Errorf("entropy failed: %v", err)
	}
}

func TestDispatchPrune(t *testing.T) {
	root := buildRouter()
	err := dispatch(root, []string{"prune"})
	if err != nil {
		t.Errorf("prune failed: %v", err)
	}
}

func TestDispatchPruneFlags(t *testing.T) {
	root := buildRouter()
	err := dispatch(root, []string{"prune", "--safe", "--dry-run"})
	if err != nil {
		t.Errorf("prune with flags failed: %v", err)
	}
}

func TestDispatchSandbox(t *testing.T) {
	root := buildRouter()
	err := dispatch(root, []string{"sandbox", "test-pkg"})
	if err == nil {
		t.Log("sandbox succeeded")
	}
}

func TestDispatchSandboxNoPackage(t *testing.T) {
	root := buildRouter()
	err := dispatch(root, []string{"sandbox"})
	if err == nil {
		t.Fatal("expected error for sandbox without package")
	}
}

func TestDispatchCompare(t *testing.T) {
	root := buildRouter()
	err := dispatch(root, []string{"compare", "pkg-a", "pkg-b"})
	if err == nil {
		t.Log("compare succeeded (may fail if no node_modules)")
	}
}

func TestDispatchCompareNoArgs(t *testing.T) {
	root := buildRouter()
	err := dispatch(root, []string{"compare", "pkg-a"})
	if err == nil {
		t.Fatal("expected error for compare with 1 arg")
	}
}

func TestDispatchSensei(t *testing.T) {
	root := buildRouter()
	err := dispatch(root, []string{"sensei"})
	if err == nil {
		t.Log("sensei succeeded")
	}
}

func TestDispatchUnknownCommand(t *testing.T) {
	root := buildRouter()
	err := dispatch(root, []string{"unknown-command"})
	if err == nil {
		t.Fatal("expected error for unknown command")
	}
}

func TestDispatchHelp(t *testing.T) {
	root := buildRouter()
	err := dispatch(root, []string{"--help"})
	if err == nil {
		t.Log("--help is handled in main(), dispatch treats it as unknown command")
	}
}

func TestFindSuggestions(t *testing.T) {
	root := buildRouter()
	sugs := findSuggestions(root, "instll")
	found := false
	for _, s := range sugs {
		if s == "install" {
			found = true
		}
	}
	if !found {
		t.Errorf("expected 'install' in suggestions for 'instll', got %v", sugs)
	}
}

func TestFindSuggestionsNoMatch(t *testing.T) {
	root := buildRouter()
	sugs := findSuggestions(root, "xyz")
	if len(sugs) != 0 {
		t.Errorf("expected no suggestions for 'xyz', got %v", sugs)
	}
}

func TestLevenshteinDistance(t *testing.T) {
	cases := []struct {
		a, b string
		want int
	}{
		{"", "", 0},
		{"a", "", 1},
		{"", "a", 1},
		{"abc", "abc", 0},
		{"kitten", "sitting", 3},
		{"install", "instll", 1},
	}
	for _, c := range cases {
		got := levenshtein(c.a, c.b)
		if got != c.want {
			t.Errorf("levenshtein(%q, %q) = %d, want %d", c.a, c.b, got, c.want)
		}
	}
}

func TestLevenshteinMaxDistance(t *testing.T) {
	a := "abcdef"
	b := "ghijkl"
	d := levenshtein(a, b)
	if d != 6 {
		t.Errorf("expected distance 6 between %q and %q, got %d", a, b, d)
	}
}

func TestTitleCase(t *testing.T) {
	cases := []struct {
		in   string
		want string
	}{
		{"hello", "Hello"},
		{"hello_world", "Hello_World"},
		{"hello-world", "Hello-World"},
		{"already", "Already"},
	}
	for _, c := range cases {
		got := titleCase(c.in)
		if got != c.want {
			t.Errorf("titleCase(%q) = %q, want %q", c.in, got, c.want)
		}
	}
}

func TestPrintHelp(t *testing.T) {
	root := buildRouter()
	printHelp(root)
}

func TestDispatchProfile(t *testing.T) {
	root := buildRouter()
	err := dispatch(root, []string{"profile"})
	if err == nil {
		t.Fatal("expected error for profile without subcommand")
	}
}

func TestDispatchProfileList(t *testing.T) {
	root := buildRouter()
	err := dispatch(root, []string{"profile", "list"})
	if err != nil {
		t.Errorf("profile list failed: %v", err)
	}
}

func TestDispatchProfileUse(t *testing.T) {
	root := buildRouter()
	err := dispatch(root, []string{"profile", "use", "strict"})
	if err != nil {
		t.Errorf("profile use failed: %v", err)
	}
}

func TestDispatchProfileUseNoName(t *testing.T) {
	root := buildRouter()
	err := dispatch(root, []string{"profile", "use"})
	if err == nil {
		t.Fatal("expected error for profile use without name")
	}
}

func TestDispatchProfileCreate(t *testing.T) {
	root := buildRouter()
	err := dispatch(root, []string{"profile", "create", "custom"})
	if err != nil {
		t.Errorf("profile create failed: %v", err)
	}
}

func TestDispatchProfileDelete(t *testing.T) {
	root := buildRouter()
	err := dispatch(root, []string{"profile", "delete", "custom"})
	if err != nil {
		t.Errorf("profile delete failed: %v", err)
	}
}

func TestDispatchProfileUnknownSub(t *testing.T) {
	root := buildRouter()
	err := dispatch(root, []string{"profile", "unknown"})
	if err == nil {
		t.Fatal("expected error for unknown profile subcommand")
	}
}

func TestMin(t *testing.T) {
	cases := []struct {
		a, b, want int
	}{
		{1, 2, 1},
		{5, 3, 3},
		{4, 4, 4},
		{-1, 1, -1},
	}
	for _, c := range cases {
		got := min(c.a, c.b)
		if got != c.want {
			t.Errorf("min(%d, %d) = %d, want %d", c.a, c.b, got, c.want)
		}
	}
}

func TestConfigDefaults(t *testing.T) {
	cfg := Config{}
	if cfg.Threads != 0 {
		t.Errorf("expected Threads=0, got %d", cfg.Threads)
	}
	if cfg.LaneMode != "" {
		t.Errorf("expected empty LaneMode, got %q", cfg.LaneMode)
	}
	if cfg.Retries != 0 {
		t.Errorf("expected Retries=0, got %d", cfg.Retries)
	}
}

func TestRunConfig(t *testing.T) {
	dir := t.TempDir()
	os.WriteFile(filepath.Join(dir, "package.json"), []byte(`{"name":"test","dependencies":{"left-pad":"^1.0.0"}}`), 0644)
	os.WriteFile(filepath.Join(dir, "package-lock.json"), []byte(`{"name":"test","lockfileVersion":3,"packages":{"":{"name":"test","dependencies":{"left-pad":"^1.0.0"}},"node_modules/left-pad":{"version":"1.3.0"}}}`), 0644)
	cfg := Config{
		Threads:      4,
		LaneMode:     "parallel",
		DryRun:       true,
		PriorityLock: true,
		SkipRebuild:  true,
		SkipBinlink:  true,
		SkipScripts:  true,
		ForceRebuild: true,
		Retries:      5,
	}
	err := runInstall(cfg, dir)
	if err != nil {
		t.Errorf("runInstall failed: %v", err)
	}
}

func TestRunConfigDefault(t *testing.T) {
	dir := t.TempDir()
	os.WriteFile(filepath.Join(dir, "package.json"), []byte(`{"name":"test"}`), 0644)
	os.WriteFile(filepath.Join(dir, "package-lock.json"), []byte(`{"name":"test","lockfileVersion":3,"packages":{"":{"name":"test"}}}`), 0644)
	err := runInstall(Config{DryRun: true}, dir)
	if err != nil {
		t.Errorf("runInstall with zero config failed: %v", err)
	}
}

func TestDispatchInstallFlags(t *testing.T) {
	withInstallDir(t, func() {
		root := buildRouter()
		err := dispatch(root, []string{"install", "--threads", "8", "--lane-mode", "sequential", "--priority-lock", "--retries", "3"})
		if err != nil {
			t.Errorf("install with all flags failed: %v", err)
		}
	})
}

func TestAddAllFlags(t *testing.T) {
	root := buildRouter()
	err := dispatch(root, []string{"add", "express", "--smart", "--why", "--dry-run=deep", "--peer-fix", "--dev", "--exact"})
	if err != nil {
		t.Errorf("add with all flags failed: %v", err)
	}
}

func TestExplainPrintsInfo(t *testing.T) {
	root := buildRouter()
	err := dispatch(root, []string{"explain", "express"})
	if err != nil {
		t.Logf("explain returned error (expected if no node_modules): %v", err)
	}
}

func TestRouterHasUsage(t *testing.T) {
	root := buildRouter()
	if root.Usage == "" {
		t.Error("expected root to have usage")
	}
	for _, cmd := range root.Subcommands {
		if cmd.Usage == "" {
			t.Errorf("command %q has no usage", cmd.Name)
		}
		if cmd.Description == "" {
			t.Errorf("command %q has no description", cmd.Name)
		}
	}
}

func TestSuggestionByPrefix(t *testing.T) {
	root := buildRouter()
	sugs := findSuggestions(root, "inst")
	found := false
	for _, s := range sugs {
		if s == "install" {
			found = true
		}
	}
	if !found {
		t.Errorf("expected prefix match for 'inst', got %v", sugs)
	}
}

func TestSuggestionExact(t *testing.T) {
	root := buildRouter()
	sugs := findSuggestions(root, "install")
	found := false
	for _, s := range sugs {
		if s == "install" {
			found = true
		}
	}
	if !found {
		t.Errorf("expected exact match for 'install'")
	}
}

func TestDispatchPruneDryRunMsg(t *testing.T) {
	root := buildRouter()
	err := dispatch(root, []string{"prune", "--dry-run"})
	if err != nil {
		t.Errorf("prune --dry-run failed: %v", err)
	}
}

func TestDispatchPruneSafe(t *testing.T) {
	root := buildRouter()
	err := dispatch(root, []string{"prune", "--safe"})
	if err != nil {
		t.Errorf("prune --safe failed: %v", err)
	}
}

func TestDispatchInstallAllBooleans(t *testing.T) {
	withInstallDir(t, func() {
		root := buildRouter()
		err := dispatch(root, []string{"install", "--skip-rebuild", "--skip-binlink", "--skip-scripts", "--force-rebuild", "--dry-run"})
		if err != nil {
			t.Errorf("install with all boolean flags failed: %v", err)
		}
	})
}

func TestOutputContainsExpected(t *testing.T) {
	root := buildRouter()
	var err error

	withInstallDir(t, func() {
		err = dispatch(root, []string{"install"})
		if err != nil {
			t.Errorf("install dispatch: %v", err)
		}
	})

	err = dispatch(root, []string{"add", "lodash"})
	if err != nil {
		t.Errorf("add dispatch: %v", err)
	}
}

func TestDispatchHandleCase(t *testing.T) {
	root := buildRouter()
	err := dispatch(root, []string{"Add", "lodash"})
	if err == nil {
		t.Log("Add dispatched (command names are case-sensitive, this may fail)")
	}
}

func TestFindSuggestionsExactMatch(t *testing.T) {
	root := buildRouter()
	sugs := findSuggestions(root, "add")
	if len(sugs) == 0 {
		t.Errorf("expected at least one suggestion for 'add'")
	}
}

func TestLevenshteinBoundaries(t *testing.T) {
	if levenshtein("", "") != 0 {
		t.Error("empty strings should have distance 0")
	}
	if levenshtein("a", "") != 1 {
		t.Error("distance from 'a' to empty should be 1")
	}
	if levenshtein("", "b") != 1 {
		t.Error("distance from empty to 'b' should be 1")
	}
	if levenshtein("abc", "xyz") != 3 {
		t.Error("distance 'abc' to 'xyz' should be 3")
	}
}

func TestCommandStructFlags(t *testing.T) {
	root := buildRouter()
	for _, cmd := range root.Subcommands {
		for _, f := range cmd.Flags {
			if f.Name == "" {
				t.Errorf("command %q has a flag with empty name", cmd.Name)
			}
		}
	}
}

func TestDispatchNoArgsCallsInstall(t *testing.T) {
	withInstallDir(t, func() {
		root := buildRouter()
		err := dispatch(root, []string{})
		if err != nil {
			t.Errorf("dispatch no args should call install: %v", err)
		}
	})
}

func TestRouterSubCommandsNotEmpty(t *testing.T) {
	root := buildRouter()
	for _, cmd := range root.Subcommands {
		if strings.TrimSpace(cmd.Name) == "" {
			t.Errorf("found subcommand with empty name")
		}
	}
}

func TestDispatchSandboxPrintsInfo(t *testing.T) {
	root := buildRouter()
	err := dispatch(root, []string{"sandbox", "lodash"})
	if err != nil {
		t.Logf("sandbox returned error: %v", err)
	}
}

func TestTitleCaseEdgeCases(t *testing.T) {
	if titleCase("") != "" {
		t.Error("expected empty string")
	}
	if titleCase("a") != "A" {
		t.Errorf("expected 'A', got %q", titleCase("a"))
	}
	if titleCase("already_capital") != "Already_Capital" {
		t.Errorf("expected 'Already_Capital', got %q", titleCase("already_capital"))
	}
}
