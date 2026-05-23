package config

import (
	"os"
	"path/filepath"
	"testing"
)

func TestParseTOML(t *testing.T) {
	data := []byte(`
[core]
parallel_limit = 8
auto_fix_peers = true
retry_count = 3

[behavior]
safe_mode = false
auto_prune = false
confirm_destructive = true

[ui]
mode = "developer"

[profile.strict]
ui = "minimal"
safe_mode = true
`)

	sections, err := parseTOML(data)
	if err != nil {
		t.Fatalf("parseTOML failed: %v", err)
	}

	if sections["core"]["parallel_limit"] != "8" {
		t.Errorf("expected parallel_limit=8, got %q", sections["core"]["parallel_limit"])
	}
	if sections["core"]["auto_fix_peers"] != "true" {
		t.Errorf("expected auto_fix_peers=true, got %q", sections["core"]["auto_fix_peers"])
	}
	if sections["behavior"]["safe_mode"] != "false" {
		t.Errorf("expected safe_mode=false, got %q", sections["behavior"]["safe_mode"])
	}
	if sections["ui"]["mode"] != `"developer"` {
		t.Errorf("expected mode=\"developer\", got %q", sections["ui"]["mode"])
	}
	if sections["profile.strict"]["ui"] != `"minimal"` {
		t.Errorf("expected profile.strict.ui=\"minimal\", got %q", sections["profile.strict"]["ui"])
	}
}

func TestParseTOMLInlineComment(t *testing.T) {
	data := []byte(`
[core]
parallel_limit = 8 # max parallelism
auto_fix_peers = true  # fix peers automatically
`)

	sections, err := parseTOML(data)
	if err != nil {
		t.Fatalf("parseTOML failed: %v", err)
	}

	if sections["core"]["parallel_limit"] != "8" {
		t.Errorf("expected parallel_limit=8, got %q", sections["core"]["parallel_limit"])
	}
	if sections["core"]["auto_fix_peers"] != "true" {
		t.Errorf("expected auto_fix_peers=true, got %q", sections["core"]["auto_fix_peers"])
	}
}

func TestParseTOMLFullLineComment(t *testing.T) {
	data := []byte(`
# This is a comment
[core]
parallel_limit = 8
`)

	sections, err := parseTOML(data)
	if err != nil {
		t.Fatalf("parseTOML failed: %v", err)
	}

	if sections["core"]["parallel_limit"] != "8" {
		t.Errorf("expected parallel_limit=8, got %q", sections["core"]["parallel_limit"])
	}
}

func TestParseTOMLUnclosedSection(t *testing.T) {
	data := []byte(`[core`)
	_, err := parseTOML(data)
	if err == nil {
		t.Fatal("expected error for unclosed section")
	}
}

func TestParseTOMLEmptySectionName(t *testing.T) {
	data := []byte(`[]`)
	_, err := parseTOML(data)
	if err == nil {
		t.Fatal("expected error for empty section name")
	}
}

func TestEncodeTOML(t *testing.T) {
	sections := map[string]map[string]string{
		"core": {
			"parallel_limit": "8",
			"auto_fix_peers": "true",
		},
		"behavior": {
			"safe_mode": "false",
		},
		"ui": {
			"mode": `"developer"`,
		},
	}

	data, err := encodeTOML(sections)
	if err != nil {
		t.Fatalf("encodeTOML failed: %v", err)
	}

	reparsed, err := parseTOML(data)
	if err != nil {
		t.Fatalf("re-parse failed: %v", err)
	}

	for sec, kv := range sections {
		for k, v := range kv {
			if reparsed[sec][k] != v {
				t.Errorf("round-trip mismatch: %s.%s: expected %q, got %q", sec, k, v, reparsed[sec][k])
			}
		}
	}
}

func TestDefaultConfig(t *testing.T) {
	cfg := DefaultConfig()

	if cfg.Core.ParallelLimit != 8 {
		t.Errorf("expected ParallelLimit=8, got %d", cfg.Core.ParallelLimit)
	}
	if !cfg.Core.AutoFixPeers {
		t.Error("expected AutoFixPeers=true")
	}
	if cfg.Core.RetryCount != 3 {
		t.Errorf("expected RetryCount=3, got %d", cfg.Core.RetryCount)
	}
	if cfg.Behavior.SafeMode {
		t.Error("expected SafeMode=false")
	}
	if cfg.Behavior.AutoPrune {
		t.Error("expected AutoPrune=false")
	}
	if !cfg.Behavior.ConfirmDestructive {
		t.Error("expected ConfirmDestructive=true")
	}
	if !cfg.Install.PreferLockfile {
		t.Error("expected PreferLockfile=true")
	}
	if cfg.Install.VersionStrategy != "stable" {
		t.Errorf("expected VersionStrategy=stable, got %q", cfg.Install.VersionStrategy)
	}
	if cfg.Install.IgnoreScripts {
		t.Error("expected IgnoreScripts=false")
	}
	if cfg.Install.SkipBinLink {
		t.Error("expected SkipBinLink=false")
	}
	if cfg.UI.Mode != "developer" {
		t.Errorf("expected Mode=developer, got %q", cfg.UI.Mode)
	}
}

func TestLoadConfigNoFile(t *testing.T) {
	dir := t.TempDir()
	cfg, err := LoadConfig(dir)
	if err != nil {
		t.Fatalf("LoadConfig failed: %v", err)
	}
	if cfg == nil {
		t.Fatal("LoadConfig returned nil")
	}
	if cfg.Core.ParallelLimit != 8 {
		t.Errorf("expected default ParallelLimit=8, got %d", cfg.Core.ParallelLimit)
	}
}

func TestLoadConfigFromFile(t *testing.T) {
	dir := t.TempDir()
	content := []byte(`
[core]
parallel_limit = 16
auto_fix_peers = false
retry_count = 5

[behavior]
safe_mode = true
auto_prune = true
confirm_destructive = false

[install_policy]
prefer_lockfile = false
version_strategy = "testing"
ignore_scripts = true
skip_binlink = true

[ui]
mode = "minimal"
`)
	path := filepath.Join(dir, configFileName)
	if err := os.WriteFile(path, content, 0644); err != nil {
		t.Fatalf("WriteFile failed: %v", err)
	}

	cfg, err := LoadConfig(dir)
	if err != nil {
		t.Fatalf("LoadConfig failed: %v", err)
	}

	if cfg.Core.ParallelLimit != 16 {
		t.Errorf("expected ParallelLimit=16, got %d", cfg.Core.ParallelLimit)
	}
	if cfg.Core.AutoFixPeers {
		t.Error("expected AutoFixPeers=false")
	}
	if cfg.Core.RetryCount != 5 {
		t.Errorf("expected RetryCount=5, got %d", cfg.Core.RetryCount)
	}
	if !cfg.Behavior.SafeMode {
		t.Error("expected SafeMode=true")
	}
	if !cfg.Behavior.AutoPrune {
		t.Error("expected AutoPrune=true")
	}
	if cfg.Behavior.ConfirmDestructive {
		t.Error("expected ConfirmDestructive=false")
	}
	if cfg.Install.PreferLockfile {
		t.Error("expected PreferLockfile=false")
	}
	if cfg.Install.VersionStrategy != "testing" {
		t.Errorf("expected VersionStrategy=testing, got %q", cfg.Install.VersionStrategy)
	}
	if !cfg.Install.IgnoreScripts {
		t.Error("expected IgnoreScripts=true")
	}
	if !cfg.Install.SkipBinLink {
		t.Error("expected SkipBinLink=true")
	}
	if cfg.UI.Mode != "minimal" {
		t.Errorf("expected Mode=minimal, got %q", cfg.UI.Mode)
	}
}

func TestSaveConfig(t *testing.T) {
	dir := t.TempDir()
	cfg := DefaultConfig()
	cfg.Core.ParallelLimit = 24
	cfg.Behavior.SafeMode = true
	cfg.Install.VersionStrategy = "edge"
	cfg.UI.Mode = "psychotic"

	if err := cfg.Save(dir); err != nil {
		t.Fatalf("Save failed: %v", err)
	}

	loaded, err := LoadConfig(dir)
	if err != nil {
		t.Fatalf("LoadConfig after save failed: %v", err)
	}

	if loaded.Core.ParallelLimit != 24 {
		t.Errorf("expected ParallelLimit=24, got %d", loaded.Core.ParallelLimit)
	}
	if !loaded.Behavior.SafeMode {
		t.Error("expected SafeMode=true")
	}
	if loaded.Install.VersionStrategy != "edge" {
		t.Errorf("expected VersionStrategy=edge, got %q", loaded.Install.VersionStrategy)
	}
	if loaded.UI.Mode != "psychotic" {
		t.Errorf("expected Mode=psychotic, got %q", loaded.UI.Mode)
	}
}

func TestProfileMerging(t *testing.T) {
	dir := t.TempDir()
	content := []byte(`
[core]
parallel_limit = 8

[behavior]
safe_mode = false

[profile.strict]
ui = "minimal"
safe_mode = true
`)
	path := filepath.Join(dir, configFileName)
	if err := os.WriteFile(path, content, 0644); err != nil {
		t.Fatalf("WriteFile failed: %v", err)
	}

	cfg, err := LoadConfig(dir)
	if err != nil {
		t.Fatalf("LoadConfig failed: %v", err)
	}

	prof := cfg.GetProfile("strict")
	if prof == nil {
		t.Fatal("expected profile 'strict' to exist")
	}
	if prof.UI != "minimal" {
		t.Errorf("expected profile UI=minimal, got %q", prof.UI)
	}
	if prof.SafeMode == nil || *prof.SafeMode != true {
		t.Error("expected profile SafeMode=true")
	}
	if prof.Name != "strict" {
		t.Errorf("expected profile Name=strict, got %q", prof.Name)
	}

	if err := cfg.ApplyProfile("strict"); err != nil {
		t.Fatalf("ApplyProfile failed: %v", err)
	}
	if cfg.UI.Mode != "minimal" {
		t.Errorf("expected UI.Mode=minimal after apply, got %q", cfg.UI.Mode)
	}
	if !cfg.Behavior.SafeMode {
		t.Error("expected SafeMode=true after apply")
	}
}

func TestGetProfileNotFound(t *testing.T) {
	cfg := DefaultConfig()
	prof := cfg.GetProfile("nonexistent")
	if prof != nil {
		t.Error("expected nil for nonexistent profile")
	}
}

func TestApplyProfileNotFound(t *testing.T) {
	cfg := DefaultConfig()
	err := cfg.ApplyProfile("nonexistent")
	if err == nil {
		t.Fatal("expected error for nonexistent profile")
	}
}

func TestGetEffectiveInstallPolicy(t *testing.T) {
	cfg := DefaultConfig()

	pol := cfg.GetEffectiveInstallPolicy("some-project")
	if pol.VersionStrategy != "stable" {
		t.Errorf("expected VersionStrategy=stable, got %q", pol.VersionStrategy)
	}
}

func TestGetEffectiveInstallPolicyWithOverrides(t *testing.T) {
	cfg := DefaultConfig()
	cfg.projectOverrides["critical-app"] = InstallPolicy{
		PreferLockfile:  true,
		VersionStrategy: "locked",
		IgnoreScripts:   true,
		SkipBinLink:     true,
	}

	pol := cfg.GetEffectiveInstallPolicy("critical-app")
	if pol.VersionStrategy != "locked" {
		t.Errorf("expected VersionStrategy=locked, got %q", pol.VersionStrategy)
	}
	if !pol.IgnoreScripts {
		t.Error("expected IgnoreScripts=true")
	}

	polDefault := cfg.GetEffectiveInstallPolicy("other-project")
	if polDefault.VersionStrategy != "stable" {
		t.Errorf("expected VersionStrategy=stable for other project, got %q", polDefault.VersionStrategy)
	}
}

func TestLoadConfigWithProfile(t *testing.T) {
	dir := t.TempDir()
	content := []byte(`
[core]
parallel_limit = 16

[profile]
active = "strict"

[profile.strict]
ui = "minimal"
safe_mode = true
`)
	path := filepath.Join(dir, configFileName)
	if err := os.WriteFile(path, content, 0644); err != nil {
		t.Fatalf("WriteFile failed: %v", err)
	}

	cfg, err := LoadConfig(dir)
	if err != nil {
		t.Fatalf("LoadConfig failed: %v", err)
	}

	if cfg.Profile != "strict" {
		t.Errorf("expected Profile=strict, got %q", cfg.Profile)
	}
	if cfg.UI.Mode != "minimal" {
		t.Errorf("expected UI.Mode=minimal after auto-apply, got %q", cfg.UI.Mode)
	}
	if !cfg.Behavior.SafeMode {
		t.Error("expected SafeMode=true after auto-apply")
	}
}

func TestConfigSaveAndLoadRoundTrip(t *testing.T) {
	dir := t.TempDir()

	cfg := DefaultConfig()
	cfg.Core.ParallelLimit = 32
	cfg.Core.AutoFixPeers = false
	cfg.Behavior.SafeMode = true
	cfg.Behavior.AutoPrune = true
	cfg.Behavior.ConfirmDestructive = false
	cfg.Install.PreferLockfile = false
	cfg.Install.VersionStrategy = "nightly"
	cfg.Install.IgnoreScripts = true
	cfg.Install.SkipBinLink = true
	cfg.UI.Mode = "psychotic"

	if err := cfg.Save(dir); err != nil {
		t.Fatalf("Save failed: %v", err)
	}

	loaded, err := LoadConfig(dir)
	if err != nil {
		t.Fatalf("LoadConfig failed: %v", err)
	}

	if loaded.Core.ParallelLimit != 32 {
		t.Errorf("ParallelLimit: expected 32, got %d", loaded.Core.ParallelLimit)
	}
	if loaded.Core.AutoFixPeers != false {
		t.Errorf("AutoFixPeers: expected false, got %v", loaded.Core.AutoFixPeers)
	}
	if loaded.Behavior.SafeMode != true {
		t.Errorf("SafeMode: expected true, got %v", loaded.Behavior.SafeMode)
	}
	if loaded.Behavior.AutoPrune != true {
		t.Errorf("AutoPrune: expected true, got %v", loaded.Behavior.AutoPrune)
	}
	if loaded.Behavior.ConfirmDestructive != false {
		t.Errorf("ConfirmDestructive: expected false, got %v", loaded.Behavior.ConfirmDestructive)
	}
	if loaded.Install.PreferLockfile != false {
		t.Errorf("PreferLockfile: expected false, got %v", loaded.Install.PreferLockfile)
	}
	if loaded.Install.VersionStrategy != "nightly" {
		t.Errorf("VersionStrategy: expected nightly, got %q", loaded.Install.VersionStrategy)
	}
	if loaded.Install.IgnoreScripts != true {
		t.Errorf("IgnoreScripts: expected true, got %v", loaded.Install.IgnoreScripts)
	}
	if loaded.Install.SkipBinLink != true {
		t.Errorf("SkipBinLink: expected true, got %v", loaded.Install.SkipBinLink)
	}
	if loaded.UI.Mode != "psychotic" {
		t.Errorf("Mode: expected psychotic, got %q", loaded.UI.Mode)
	}
}
