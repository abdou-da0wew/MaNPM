package config

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
)

type Config struct {
	Core     CoreConfig     `toml:"core"`
	Behavior BehaviorConfig `toml:"behavior"`
	Install  InstallPolicy  `toml:"install_policy"`
	Profile  string         `toml:"profile"`
	UI       UIConfig       `toml:"ui"`

	profiles        map[string]Profile
	projectOverrides map[string]InstallPolicy
}

type CoreConfig struct {
	ParallelLimit int  `toml:"parallel_limit"`
	AutoFixPeers  bool `toml:"auto_fix_peers"`
	RetryCount    int  `toml:"retry_count"`
}

type BehaviorConfig struct {
	SafeMode           bool `toml:"safe_mode"`
	AutoPrune          bool `toml:"auto_prune"`
	ConfirmDestructive bool `toml:"confirm_destructive"`
}

type InstallPolicy struct {
	PreferLockfile  bool   `toml:"prefer_lockfile"`
	VersionStrategy string `toml:"version_strategy"`
	IgnoreScripts   bool   `toml:"ignore_scripts"`
	SkipBinLink     bool   `toml:"skip_binlink"`
}

type UIConfig struct {
	Mode string `toml:"mode"`
}

type Profile struct {
	Name            string
	UI              string `toml:"ui"`
	Strictness      string `toml:"strictness"`
	VersionStrategy string `toml:"version_strategy"`
	SafeMode        *bool  `toml:"safe_mode"`
}

const configFileName = "manpm.config.toml"

func DefaultConfig() *Config {
	return &Config{
		Core: CoreConfig{
			ParallelLimit: 8,
			AutoFixPeers:  true,
			RetryCount:    3,
		},
		Behavior: BehaviorConfig{
			SafeMode:           false,
			AutoPrune:          false,
			ConfirmDestructive: true,
		},
		Install: InstallPolicy{
			PreferLockfile:  true,
			VersionStrategy: "stable",
			IgnoreScripts:   false,
			SkipBinLink:     false,
		},
		UI: UIConfig{
			Mode: "developer",
		},
		profiles:         map[string]Profile{},
		projectOverrides: map[string]InstallPolicy{},
	}
}

func LoadConfig(dir string) (*Config, error) {
	cfg := DefaultConfig()
	path := filepath.Join(dir, configFileName)

	data, err := os.ReadFile(path)
	if err != nil {
		if os.IsNotExist(err) {
			return cfg, nil
		}
		return nil, fmt.Errorf("config: reading %s: %w", path, err)
	}

	sections, err := parseTOML(data)
	if err != nil {
		return nil, fmt.Errorf("config: parsing %s: %w", path, err)
	}

	if err := applySections(cfg, sections); err != nil {
		return nil, fmt.Errorf("config: applying %s: %w", path, err)
	}

	if cfg.Profile != "" {
		if err := cfg.ApplyProfile(cfg.Profile); err != nil {
			return nil, fmt.Errorf("config: applying profile %q: %w", cfg.Profile, err)
		}
	}

	return cfg, nil
}

func applySections(cfg *Config, sections map[string]map[string]string) error {
	for section, kv := range sections {
		if section == "core" {
			if err := applyCore(&cfg.Core, kv); err != nil {
				return err
			}
		} else if section == "behavior" {
			if err := applyBehavior(&cfg.Behavior, kv); err != nil {
				return err
			}
		} else if section == "install_policy" {
			if err := applyInstallPolicy(&cfg.Install, kv); err != nil {
				return err
			}
		} else if section == "ui" {
			if err := applyUI(&cfg.UI, kv); err != nil {
				return err
			}
		} else if section == "profile" {
			if v, ok := kv["active"]; ok {
				s, err := parseString(v)
				if err != nil {
					return fmt.Errorf("config: profile.active: %w", err)
				}
				cfg.Profile = s
			}
		} else if strings.HasPrefix(section, "profile.") {
			name := strings.TrimPrefix(section, "profile.")
			p := Profile{Name: name}
			if err := applyProfile(&p, kv); err != nil {
				return err
			}
			if cfg.profiles == nil {
				cfg.profiles = map[string]Profile{}
			}
			cfg.profiles[name] = p
		} else if strings.HasPrefix(section, "override.") {
			projectName := strings.TrimPrefix(section, "override.")
			var pol InstallPolicy
			if err := applyInstallPolicy(&pol, kv); err != nil {
				return err
			}
			if cfg.projectOverrides == nil {
				cfg.projectOverrides = map[string]InstallPolicy{}
			}
			cfg.projectOverrides[projectName] = pol
		}
	}
	return nil
}

func applyCore(c *CoreConfig, kv map[string]string) error {
	for k, v := range kv {
		switch k {
		case "parallel_limit":
			n, err := parseInt(v)
			if err != nil {
				return fmt.Errorf("config: core.parallel_limit: %w", err)
			}
			c.ParallelLimit = n
		case "auto_fix_peers":
			b, err := parseBool(v)
			if err != nil {
				return fmt.Errorf("config: core.auto_fix_peers: %w", err)
			}
			c.AutoFixPeers = b
		case "retry_count":
			n, err := parseInt(v)
			if err != nil {
				return fmt.Errorf("config: core.retry_count: %w", err)
			}
			c.RetryCount = n
		}
	}
	return nil
}

func applyBehavior(c *BehaviorConfig, kv map[string]string) error {
	for k, v := range kv {
		switch k {
		case "safe_mode":
			b, err := parseBool(v)
			if err != nil {
				return fmt.Errorf("config: behavior.safe_mode: %w", err)
			}
			c.SafeMode = b
		case "auto_prune":
			b, err := parseBool(v)
			if err != nil {
				return fmt.Errorf("config: behavior.auto_prune: %w", err)
			}
			c.AutoPrune = b
		case "confirm_destructive":
			b, err := parseBool(v)
			if err != nil {
				return fmt.Errorf("config: behavior.confirm_destructive: %w", err)
			}
			c.ConfirmDestructive = b
		}
	}
	return nil
}

func applyInstallPolicy(p *InstallPolicy, kv map[string]string) error {
	for k, v := range kv {
		switch k {
		case "prefer_lockfile":
			b, err := parseBool(v)
			if err != nil {
				return fmt.Errorf("config: install_policy.prefer_lockfile: %w", err)
			}
			p.PreferLockfile = b
		case "version_strategy":
			s, err := parseString(v)
			if err != nil {
				return fmt.Errorf("config: install_policy.version_strategy: %w", err)
			}
			p.VersionStrategy = s
		case "ignore_scripts":
			b, err := parseBool(v)
			if err != nil {
				return fmt.Errorf("config: install_policy.ignore_scripts: %w", err)
			}
			p.IgnoreScripts = b
		case "skip_binlink":
			b, err := parseBool(v)
			if err != nil {
				return fmt.Errorf("config: install_policy.skip_binlink: %w", err)
			}
			p.SkipBinLink = b
		}
	}
	return nil
}

func applyUI(c *UIConfig, kv map[string]string) error {
	for k, v := range kv {
		switch k {
		case "mode":
			s, err := parseString(v)
			if err != nil {
				return fmt.Errorf("config: ui.mode: %w", err)
			}
			c.Mode = s
		}
	}
	return nil
}

func applyProfile(p *Profile, kv map[string]string) error {
	for k, v := range kv {
		switch k {
		case "ui":
			s, err := parseString(v)
			if err != nil {
				return fmt.Errorf("config: profile.ui: %w", err)
			}
			p.UI = s
		case "strictness":
			s, err := parseString(v)
			if err != nil {
				return fmt.Errorf("config: profile.strictness: %w", err)
			}
			p.Strictness = s
		case "version_strategy":
			s, err := parseString(v)
			if err != nil {
				return fmt.Errorf("config: profile.version_strategy: %w", err)
			}
			p.VersionStrategy = s
		case "safe_mode":
			b, err := parseBool(v)
			if err != nil {
				return fmt.Errorf("config: profile.safe_mode: %w", err)
			}
			p.SafeMode = &b
		}
	}
	return nil
}

func (c *Config) Save(dir string) error {
	sections := map[string]map[string]string{}

	coreKV := map[string]string{}
	coreKV["parallel_limit"] = fmt.Sprintf("%d", c.Core.ParallelLimit)
	coreKV["auto_fix_peers"] = fmt.Sprintf("%t", c.Core.AutoFixPeers)
	coreKV["retry_count"] = fmt.Sprintf("%d", c.Core.RetryCount)
	sections["core"] = coreKV

	behKV := map[string]string{}
	behKV["safe_mode"] = fmt.Sprintf("%t", c.Behavior.SafeMode)
	behKV["auto_prune"] = fmt.Sprintf("%t", c.Behavior.AutoPrune)
	behKV["confirm_destructive"] = fmt.Sprintf("%t", c.Behavior.ConfirmDestructive)
	sections["behavior"] = behKV

	instKV := map[string]string{}
	instKV["prefer_lockfile"] = fmt.Sprintf("%t", c.Install.PreferLockfile)
	instKV["version_strategy"] = fmt.Sprintf("%q", c.Install.VersionStrategy)
	instKV["ignore_scripts"] = fmt.Sprintf("%t", c.Install.IgnoreScripts)
	instKV["skip_binlink"] = fmt.Sprintf("%t", c.Install.SkipBinLink)
	sections["install_policy"] = instKV

	uiKV := map[string]string{}
	uiKV["mode"] = fmt.Sprintf("%q", c.UI.Mode)
	sections["ui"] = uiKV

	if c.Profile != "" {
		profileKV := map[string]string{}
		profileKV["active"] = fmt.Sprintf("%q", c.Profile)
		sections["profile"] = profileKV
	}

	for name, p := range c.profiles {
		pKV := map[string]string{}
		if p.UI != "" {
			pKV["ui"] = fmt.Sprintf("%q", p.UI)
		}
		if p.Strictness != "" {
			pKV["strictness"] = fmt.Sprintf("%q", p.Strictness)
		}
		if p.VersionStrategy != "" {
			pKV["version_strategy"] = fmt.Sprintf("%q", p.VersionStrategy)
		}
		if p.SafeMode != nil {
			pKV["safe_mode"] = fmt.Sprintf("%t", *p.SafeMode)
		}
		if len(pKV) > 0 {
			sections["profile."+name] = pKV
		}
	}

	for project, pol := range c.projectOverrides {
		ovKV := map[string]string{}
		ovKV["prefer_lockfile"] = fmt.Sprintf("%t", pol.PreferLockfile)
		ovKV["version_strategy"] = fmt.Sprintf("%q", pol.VersionStrategy)
		ovKV["ignore_scripts"] = fmt.Sprintf("%t", pol.IgnoreScripts)
		ovKV["skip_binlink"] = fmt.Sprintf("%t", pol.SkipBinLink)
		sections["override."+project] = ovKV
	}

	data, err := encodeTOML(sections)
	if err != nil {
		return fmt.Errorf("config: encoding: %w", err)
	}

	path := filepath.Join(dir, configFileName)
	if err := os.WriteFile(path, data, 0644); err != nil {
		return fmt.Errorf("config: writing %s: %w", path, err)
	}

	return nil
}

func (c *Config) GetProfile(name string) *Profile {
	if c.profiles == nil {
		return nil
	}
	p, ok := c.profiles[name]
	if !ok {
		return nil
	}
	return &p
}

func (c *Config) ApplyProfile(name string) error {
	p := c.GetProfile(name)
	if p == nil {
		return fmt.Errorf("config: profile %q not found", name)
	}

	if p.UI != "" {
		c.UI.Mode = p.UI
	}
	if p.Strictness != "" {
		if p.Strictness == "safe" {
			c.Behavior.SafeMode = true
		}
	}
	if p.VersionStrategy != "" {
		c.Install.VersionStrategy = p.VersionStrategy
	}
	if p.SafeMode != nil {
		c.Behavior.SafeMode = *p.SafeMode
	}
	if p.Name != "" {
		c.Profile = p.Name
	}

	return nil
}

func (c *Config) GetEffectiveInstallPolicy(projectName string) InstallPolicy {
	pol := c.Install

	if ov, ok := c.projectOverrides[projectName]; ok {
		pol = ov
	}

	return pol
}
