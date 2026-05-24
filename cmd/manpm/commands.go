package main

import (
	"bufio"
	"context"
	"encoding/json"
	"flag"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"sort"
	"strings"
	"unicode"

	"manpm/pkg/binlink"
	"manpm/pkg/buildmgr"
	"manpm/pkg/config"
	"manpm/pkg/extractor"
	"manpm/pkg/graph"
	"manpm/pkg/intel"
	"manpm/pkg/lockfile"
	"manpm/pkg/preflight"
	"manpm/pkg/ui"
)

type Flag struct {
	Name        string
	Description string
	Default     string
	Short       string
}

type Command struct {
	Name        string
	Description string
	Usage       string
	Run         func(args []string) error
	Subcommands []Command
	Flags       []Flag
}

type Config struct {
	Threads      int
	LaneMode     string
	PriorityLock bool
	DryRun       bool
	SkipRebuild  bool
	SkipBinlink  bool
	SkipScripts  bool
	ForceRebuild bool
	Retries      int
}

func buildGraph(dir string) (*graph.DependencyGraph, error) {
	lfPath, err := lockfile.FindLockfile(dir)
	if err != nil {
		return nil, err
	}
	lf, err := lockfile.Parse(lfPath)
	if err != nil {
		return nil, err
	}
	dag := graph.NewDependencyGraph()
	for key, pkg := range lf.Packages {
		if key == "" {
			continue
		}
		name := strings.TrimPrefix(key, "node_modules/")
		dag.AddNode(name, pkg.Version, pkg.Resolved, pkg.Integrity, pkg.Dependencies)
	}
	if err := dag.TopologicalSort(); err != nil {
		if len(dag.Levels) == 0 {
			flat := make([]*graph.PackageNode, 0, len(dag.Nodes))
			for _, node := range dag.Nodes {
				flat = append(flat, node)
			}
			dag.Levels = [][]*graph.PackageNode{flat}
		}
	}
	return dag, nil
}

func runInstall(cfg Config, dir string) error {
	ctx := context.Background()

	ui.Header("manpm install")
	ui.Label("Threads", fmt.Sprintf("%d", cfg.Threads))
	ui.Label("Mode", cfg.LaneMode)
	ui.Label("Dry-run", fmt.Sprintf("%v", cfg.DryRun))
	ui.Label("Retries", fmt.Sprintf("%d", cfg.Retries))
	if cfg.PriorityLock {
		ui.Label("Priority", "lock")
	}
	if cfg.SkipRebuild {
		ui.Info("Skipping rebuild")
	}
	if cfg.SkipBinlink {
		ui.Info("Skipping binlink")
	}
	if cfg.SkipScripts {
		ui.Info("Skipping scripts")
	}
	if cfg.ForceRebuild {
		ui.Info("Force rebuild enabled")
	}

	ui.Subheader("Preflight")
	res, err := preflight.Run(dir)
	if err != nil {
		return fmt.Errorf("preflight: %w", err)
	}
	preflight.PrintSummary(res)

	ui.Subheader("Lockfile")
	lf, err := lockfile.Parse(res.LockfilePath)
	if err != nil {
		return fmt.Errorf("parse lockfile: %w", err)
	}
	ui.Label("Packages", fmt.Sprintf("%d", len(lf.Packages)))

	ui.Subheader("Dependency Graph")
	dag := graph.NewDependencyGraph()
	for key, pkg := range lf.Packages {
		if key == "" {
			continue
		}
		name := strings.TrimPrefix(key, "node_modules/")
		dag.AddNode(name, pkg.Version, pkg.Resolved, pkg.Integrity, pkg.Dependencies)
	}
	if err := dag.TopologicalSort(); err != nil {
		ui.Warning(fmt.Sprintf("Dependency graph: %v — extracting all packages in one pass", err))
		if len(dag.Levels) == 0 {
			flat := make([]*graph.PackageNode, 0, len(dag.Nodes))
			for _, node := range dag.Nodes {
				flat = append(flat, node)
			}
			dag.Levels = [][]*graph.PackageNode{flat}
		}
	}
	ui.Label("Levels", fmt.Sprintf("%d", len(dag.Levels)))
	ui.Label("Packages", fmt.Sprintf("%d", len(dag.Nodes)))

	if dag.HasCycle() {
		ui.Warning("Circular dependencies detected")
	}

	if cfg.DryRun {
		for i, level := range dag.Levels {
			names := make([]string, 0, len(level))
			for _, n := range level {
				names = append(names, n.Name)
			}
			ui.Label(fmt.Sprintf("Level %d", i), fmt.Sprintf("%d (%s)", len(level), strings.Join(names, ", ")))
		}
		ui.Success("Dry-run complete")
		return nil
	}

	ui.Subheader("Extraction")
	extr := extractor.NewExtractor(dir, cfg.Threads)
	extr.MaxRetries = cfg.Retries

	totalExtracted := 0
	var failedExtractions []string
	for levelIdx, level := range dag.Levels {
		jobs := make([]extractor.PackageJob, 0, len(level))
		for _, node := range level {
			jobs = append(jobs, extractor.PackageJob{
				Name:       node.Name,
				Path:       node.Name,
				TarballURL: node.Resolved,
				Integrity:  node.Integrity,
			})
		}
		prog := ui.NewProgress(len(jobs))
		prog.Start()
		extr.OnProgress = func(completed, total int, name string, err error) {
			if err != nil {
				prog.Update(name + " ⚠")
			} else {
				prog.Inc(name)
			}
		}
		results := extr.ExtractLevel(ctx, jobs)
		prog.Done(fmt.Sprintf("Level %d: %d packages", levelIdx, len(jobs)))
		for _, r := range results {
			if r.Error != nil {
				failedExtractions = append(failedExtractions, r.PackageName)
			} else {
				totalExtracted++
			}
		}
	}
	ui.Label("Extracted", fmt.Sprintf("%d packages", totalExtracted))
	if len(failedExtractions) > 0 {
		ui.Label("Failed", fmt.Sprintf("%d packages", len(failedExtractions)))
	}

	bm := buildmgr.NewBuildManager(dir)
	bm.Verbose = cfg.ForceRebuild
	if !cfg.SkipRebuild {
		ui.Subheader("Native Rebuild")
		var rebuildErr error
		if cfg.LaneMode == "sequential" {
			rebuildErr = bm.RebuildSequential(ctx)
		} else {
			rebuildErr = bm.RebuildAll(ctx)
		}
		if rebuildErr != nil {
			ui.Warning(fmt.Sprintf("Rebuild: %v", rebuildErr))
		} else {
			ui.Success("Native rebuilds complete")
		}
	} else {
		ui.Info("Skipping native rebuild")
	}

	if !cfg.SkipBinlink {
		ui.Subheader("Binary Linking")
		var pkgPaths []string
		for _, level := range dag.Levels {
			for _, node := range level {
				pkgPaths = append(pkgPaths, node.Name)
			}
		}
		linker := binlink.NewLinker(filepath.Join(dir, "node_modules"))
		if err := linker.LinkAllPackages(ctx, pkgPaths); err != nil {
			ui.Warning(fmt.Sprintf("Binlink: %v", err))
		} else {
			ui.Label("Linked", fmt.Sprintf("%d packages", len(pkgPaths)))
		}
	} else {
		ui.Info("Skipping binlink")
	}

	if !cfg.SkipScripts {
		ui.Subheader("Lifecycle Scripts")
		if err := bm.RunPostinstallScripts(ctx); err != nil {
			ui.Warning(fmt.Sprintf("Scripts: %v", err))
		} else {
			ui.Success("Lifecycle scripts complete")
		}
	}

	ui.Success("Install complete")
	return nil
}

func buildRouter() Command {
	root := Command{
		Name:        "manpm",
		Description: "The blazing-fast Go orchestrator for npm packages",
		Usage:       "manpm <command> [options]",
	}

	installCmd := Command{
		Name:        "install",
		Description: "Install all dependencies (default command)",
		Usage:       "manpm install [options]",
		Flags: []Flag{
			{Name: "threads", Description: "Number of parallel workers", Default: "0", Short: "t"},
			{Name: "lane-mode", Description: "Execution lane mode (parallel|sequential)", Default: "parallel", Short: "m"},
			{Name: "priority-lock", Description: "Prioritize lockfile parsing", Default: "false", Short: "p"},
			{Name: "dry-run", Description: "Simulate without installing", Default: "false"},
			{Name: "skip-rebuild", Description: "Skip native rebuild", Default: "false"},
			{Name: "skip-binlink", Description: "Skip binary linking", Default: "false"},
			{Name: "skip-scripts", Description: "Skip lifecycle scripts", Default: "false"},
			{Name: "force-rebuild", Description: "Force native rebuild", Default: "false"},
			{Name: "retries", Description: "Max retries per package", Default: "3", Short: "r"},
		},
		Run: func(args []string) error {
			fs := flag.NewFlagSet("install", flag.ContinueOnError)
			threads := fs.Int("threads", 0, "Number of parallel workers")
			laneMode := fs.String("lane-mode", "parallel", "Execution lane mode")
			priorityLock := fs.Bool("priority-lock", false, "Prioritize lockfile parsing")
			dryRun := fs.Bool("dry-run", false, "Simulate without installing")
			skipRebuild := fs.Bool("skip-rebuild", false, "Skip native rebuild")
			skipBinlink := fs.Bool("skip-binlink", false, "Skip binary linking")
			skipScripts := fs.Bool("skip-scripts", false, "Skip lifecycle scripts")
			forceRebuild := fs.Bool("force-rebuild", false, "Force native rebuild")
			retries := fs.Int("retries", 3, "Max retries per package")
			if err := fs.Parse(args); err != nil {
				return err
			}
			cfg := Config{
				Threads:      *threads,
				LaneMode:     *laneMode,
				PriorityLock: *priorityLock,
				DryRun:       *dryRun,
				SkipRebuild:  *skipRebuild,
				SkipBinlink:  *skipBinlink,
				SkipScripts:  *skipScripts,
				ForceRebuild: *forceRebuild,
				Retries:      *retries,
			}
			dir, _ := os.Getwd()
			return runInstall(cfg, dir)
		},
	}

	addCmd := Command{
		Name:        "add",
		Description: "Add a package and show impact",
		Usage:       "manpm add <package> [options]",
		Flags: []Flag{
			{Name: "smart", Description: "Smart resolution", Default: "false"},
			{Name: "why", Description: "Show why package is needed", Default: "false"},
			{Name: "dry-run", Description: "Simulate add (use =deep for deep analysis)", Default: "false"},
			{Name: "peer-fix", Description: "Auto-fix peer dependencies", Default: "false"},
			{Name: "dev", Description: "Install as dev dependency", Default: "false"},
			{Name: "exact", Description: "Save exact version", Default: "false"},
		},
		Run: func(args []string) error {
			fs := flag.NewFlagSet("add", flag.ContinueOnError)
			smart := fs.Bool("smart", false, "Smart resolution")
			why := fs.Bool("why", false, "Show why package is needed")
			dryRun := fs.String("dry-run", "", "Simulate add")
			peerFix := fs.Bool("peer-fix", false, "Auto-fix peer dependencies")
			dev := fs.Bool("dev", false, "Install as dev dependency")
			exact := fs.Bool("exact", false, "Save exact version")
			if err := fs.Parse(args); err != nil {
				return err
			}
			pkg := fs.Arg(0)
			if pkg == "" {
				return fmt.Errorf("usage: manpm add <package>")
			}
			ui.Header("Adding package: " + pkg)
			ui.Label("Smart", fmt.Sprintf("%v", *smart))
			ui.Label("Why", fmt.Sprintf("%v", *why))
			ui.Label("Dry-run", *dryRun)
			ui.Label("Peer fix", fmt.Sprintf("%v", *peerFix))
			ui.Label("Dev", fmt.Sprintf("%v", *dev))
			ui.Label("Exact", fmt.Sprintf("%v", *exact))

			dir, _ := os.Getwd()
			if *why {
				info, err := intel.Explain(".", pkg)
				if err == nil {
					fmt.Print(info)
				}
				return nil
			}

			if *dryRun != "" {
				ui.Success("Would install: " + pkg)
				return nil
			}

			npmArgs := []string{"install", pkg}
			if *dev {
				npmArgs = append(npmArgs, "--save-dev")
			}
			if *exact {
				npmArgs = append(npmArgs, "--save-exact")
			}

			ui.Subheader("Installing")
			ui.Label("Running", "npm "+strings.Join(npmArgs, " "))

			cmd := exec.CommandContext(context.Background(), "npm", npmArgs...)
			cmd.Dir = dir
			cmd.Stdout = os.Stderr
			cmd.Stderr = os.Stderr
			if err := cmd.Run(); err != nil {
				return fmt.Errorf("npm install failed: %w", err)
			}

			ui.Success("Added " + pkg)
			return nil
		},
	}

	explainCmd := Command{
		Name:        "explain",
		Description: "Show why a package is installed",
		Usage:       "manpm explain <package>",
		Run: func(args []string) error {
			if len(args) == 0 {
				return fmt.Errorf("usage: manpm explain <package>")
			}
			info, err := intel.Explain(".", args[0])
			if err != nil {
				return err
			}
			ui.Header("Package: " + args[0])
			fmt.Print(info)
			return nil
		},
	}

	auditCmd := Command{
		Name:        "audit",
		Description: "Run vulnerability analysis",
		Usage:       "manpm audit",
		Run: func(args []string) error {
			dir, _ := os.Getwd()
			lfPath, err := lockfile.FindLockfile(dir)
			if err != nil {
				return err
			}
			results, err := intel.Audit(lfPath)
			if err != nil {
				return err
			}
			ui.Header("Security Audit")
			if len(results) == 0 {
				ui.Success("No vulnerabilities found")
				return nil
			}
			for _, r := range results {
				ui.Errorf("%s [%s] %s (CVE: %s)", r.PackageName, r.Severity, r.Title, r.CVE)
				if r.FixAvailable != "" {
					ui.Info("  Fix available: " + r.FixAvailable)
				}
			}
			return nil
		},
	}

	doctorCmd := Command{
		Name:        "doctor",
		Description: "Analyze project health",
		Usage:       "manpm doctor",
		Run: func(args []string) error {
			dir, _ := os.Getwd()
			dag, err := buildGraph(dir)
			if err != nil {
				dag = nil
			}
			result, err := intel.Doctor(dir, dag)
			if err != nil {
				return err
			}
			ui.Header("Project Health")
			ui.Label("Score", fmt.Sprintf("%.1f%%", result.Score))
			for _, iss := range result.Issues {
				switch iss.Severity {
				case "error":
					ui.Error(iss.Message)
				case "warning":
					ui.Warning(iss.Message)
				default:
					ui.Info(iss.Message)
				}
				if iss.Fix != "" {
					ui.Label("Fix", iss.Fix)
				}
			}
			if len(result.Issues) == 0 {
				ui.Success("All checks passed")
			}
			return nil
		},
	}

	mapCmd := Command{
		Name:        "map",
		Description: "Show ASCII dependency graph",
		Usage:       "manpm map",
		Run: func(args []string) error {
			dir, _ := os.Getwd()
			dag, err := buildGraph(dir)
			if err != nil {
				ui.Header("Dependency Graph")
				fmt.Println(intel.Map(nil))
				return nil
			}
			ui.Header("Dependency Graph")
			fmt.Println(intel.Map(dag))
			return nil
		},
	}

	entropyCmd := Command{
		Name:        "entropy",
		Description: "Show project chaos metrics",
		Usage:       "manpm entropy",
		Run: func(args []string) error {
			dir, _ := os.Getwd()
			dag, err := buildGraph(dir)
			if err != nil {
				dag = nil
			}
			result := intel.Entropy(dag)
			ui.Header("Project Entropy")
			ui.Label("Score", fmt.Sprintf("%.2f", result.Score))
			ui.Label("Total packages", fmt.Sprintf("%d", result.TotalPackages))
			ui.Label("Unique libraries", fmt.Sprintf("%d", result.UniqueLibraries))
			ui.Label("Avg depth", fmt.Sprintf("%.2f", result.AvgDepth))
			ui.Label("Circular deps", fmt.Sprintf("%d", result.CircularDeps))
			if len(result.RedundantGroups) > 0 {
				ui.Warning("Redundant: " + strings.Join(result.RedundantGroups, ", "))
			}
			return nil
		},
	}

	pruneCmd := Command{
		Name:        "prune",
		Description: "Show and remove unused packages",
		Usage:       "manpm prune [options]",
		Flags: []Flag{
			{Name: "safe", Description: "Safe mode (keep packages with recent usage)", Default: "false"},
			{Name: "dry-run", Description: "Only show what would be removed", Default: "false"},
		},
		Run: func(args []string) error {
			fs := flag.NewFlagSet("prune", flag.ContinueOnError)
			safe := fs.Bool("safe", false, "Safe mode (keep packages with recent usage)")
			dryRun := fs.Bool("dry-run", false, "Only show what would be removed")
			if err := fs.Parse(args); err != nil {
				return err
			}
			ui.Header("Prune")
			ui.Label("Safe", fmt.Sprintf("%v", *safe))
			ui.Label("Dry-run", fmt.Sprintf("%v", *dryRun))

			dir, _ := os.Getwd()

			pkgJSONPath := filepath.Join(dir, "package.json")
			data, err := os.ReadFile(pkgJSONPath)
			if err != nil {
				return fmt.Errorf("read package.json: %w", err)
			}
			var pkgJSON struct {
				Dependencies     map[string]string `json:"dependencies"`
				DevDependencies  map[string]string `json:"devDependencies"`
				PeerDependencies map[string]string `json:"peerDependencies"`
			}
			if err := json.Unmarshal(data, &pkgJSON); err != nil {
				return fmt.Errorf("parse package.json: %w", err)
			}

			dag, err := buildGraph(dir)
			if err != nil {
				ui.Warning("No lockfile graph — falling back to node_modules scan")
				dag = nil
			}

			imported := scanImports(dir)

			used := map[string]bool{}
			for name := range imported {
				used[name] = true
			}

			if dag != nil {
				for _, node := range dag.Nodes {
					if used[node.Name] {
						for _, dep := range node.Dependencies {
							used[dep] = true
						}
					}
				}
			}

			topLevelDeps := map[string]bool{}
			for name := range pkgJSON.Dependencies {
				topLevelDeps[name] = true
			}
			for name := range pkgJSON.DevDependencies {
				topLevelDeps[name] = true
			}
			for name := range pkgJSON.PeerDependencies {
				topLevelDeps[name] = true
			}

			var unusedTopLevel []string
			for name := range topLevelDeps {
				if !used[name] {
					unusedTopLevel = append(unusedTopLevel, name)
				}
			}
			sort.Strings(unusedTopLevel)

			var unusedTransitive []string
			if dag != nil {
				for _, node := range dag.Nodes {
					if !topLevelDeps[node.Name] && !used[node.Name] {
						unusedTransitive = append(unusedTransitive, node.Name)
					}
				}
				sort.Strings(unusedTransitive)
			}

			if len(unusedTopLevel) == 0 && len(unusedTransitive) == 0 {
				ui.Success("All packages are in use")
				return nil
			}

			if len(unusedTopLevel) > 0 {
				ui.Label("Unused top-level", fmt.Sprintf("%d packages", len(unusedTopLevel)))
				for _, name := range unusedTopLevel {
					ui.Info("  " + name)
				}
			}
			if len(unusedTransitive) > 0 {
				ui.Label("Unused transitive", fmt.Sprintf("%d packages", len(unusedTransitive)))
			}

			if *dryRun || *safe {
				return nil
			}

			nmDir := filepath.Join(dir, "node_modules")
			for _, name := range unusedTopLevel {
				if err := os.RemoveAll(filepath.Join(nmDir, name)); err != nil {
					ui.Warning(fmt.Sprintf("Remove %s: %v", name, err))
					continue
				}
				delete(pkgJSON.Dependencies, name)
				delete(pkgJSON.DevDependencies, name)
				delete(pkgJSON.PeerDependencies, name)
				ui.Label("Removed", name)
			}

			updated, _ := json.MarshalIndent(pkgJSON, "", "  ")
			os.WriteFile(pkgJSONPath, updated, 0644)
			ui.Success(fmt.Sprintf("Removed %d unused packages and updated package.json", len(unusedTopLevel)))
			return nil
		},
	}

	sandboxCmd := Command{
		Name:        "sandbox",
		Description: "Show info about isolated installation",
		Usage:       "manpm sandbox <package>",
		Run: func(args []string) error {
			if len(args) == 0 {
				return fmt.Errorf("usage: manpm sandbox <package>")
			}
			info := intel.SandboxInfo(args[0])
			ui.Header("Sandbox: " + args[0])
			fmt.Println(info)
			return nil
		},
	}

	compareCmd := Command{
		Name:        "compare",
		Description: "Compare two packages",
		Usage:       "manpm compare <package1> <package2>",
		Run: func(args []string) error {
			if len(args) < 2 {
				return fmt.Errorf("usage: manpm compare <package1> <package2>")
			}
			result, err := intel.Compare(".", args[0], args[1])
			if err != nil {
				return err
			}
			ui.Header(fmt.Sprintf("Comparing: %s vs %s", args[0], args[1]))
			fmt.Print(result)
			return nil
		},
	}

	senseiCmd := Command{
		Name:        "sensei",
		Description: "Full project review",
		Usage:       "manpm sensei",
		Run: func(args []string) error {
			review, err := intel.Sensei(".")
			if err != nil {
				return err
			}
			fmt.Println(review)
			return nil
		},
	}

	runCmd := Command{
		Name:        "run",
		Description: "Run a project script (e.g. manpm run build)",
		Usage:       "manpm run <script>",
		Run: func(args []string) error {
			if len(args) == 0 {
				return fmt.Errorf("usage: manpm run <script>")
			}
			dir, _ := os.Getwd()
			bm := buildmgr.NewBuildManager(dir)
			return bm.RunProjectScript(context.Background(), args[0])
		},
	}

	var profileCmd Command
	profileCmd = Command{
		Name:        "profile",
		Description: "Manage installation profiles",
		Usage:       "manpm profile <subcommand> [name]",
		Subcommands: []Command{
			{
				Name:        "list",
				Description: "List available profiles",
				Usage:       "manpm profile list",
				Run: func(args []string) error {
					dir, _ := os.Getwd()
					cfg, err := config.LoadConfig(dir)
					if err != nil {
						return err
					}
					ui.Subheader("Available profiles")
					profiles := cfg.ListProfiles()
					if len(profiles) == 0 {
						ui.Info("  (no profiles defined)")
						return nil
					}
					for _, name := range profiles {
						marker := "  "
						if name == cfg.Profile {
							marker = "  " + ui.BoldText("➜")
						}
						fmt.Printf("%s %s\n", marker, name)
					}
					if cfg.Profile != "" {
						ui.Label("Active", cfg.Profile)
					}
					return nil
				},
			},
			{
				Name:        "use",
				Description: "Switch to a profile",
				Usage:       "manpm profile use <name>",
				Run: func(args []string) error {
					if len(args) == 0 {
						return fmt.Errorf("usage: manpm profile use <name>")
					}
					dir, _ := os.Getwd()
					cfg, err := config.LoadConfig(dir)
					if err != nil {
						return err
					}
					if cfg.GetProfile(args[0]) == nil {
						available := cfg.ListProfiles()
						return fmt.Errorf("profile %q not found (available: %v)", args[0], strings.Join(available, ", "))
					}
					cfg.SetActiveProfile(args[0])
					if err := cfg.Save(dir); err != nil {
						return fmt.Errorf("save config: %w", err)
					}
					ui.Success("Switched to profile: " + args[0])
					return nil
				},
			},
			{
				Name:        "create",
				Description: "Create a new profile",
				Usage:       "manpm profile create <name>",
				Run: func(args []string) error {
					if len(args) == 0 {
						return fmt.Errorf("usage: manpm profile create <name>")
					}
					dir, _ := os.Getwd()
					cfg, err := config.LoadConfig(dir)
					if err != nil {
						return err
					}
					if cfg.GetProfile(args[0]) != nil {
						return fmt.Errorf("profile %q already exists", args[0])
					}
					cfg.AddProfile(config.Profile{
						Name:            args[0],
						VersionStrategy: "stable",
					})
					if err := cfg.Save(dir); err != nil {
						return fmt.Errorf("save config: %w", err)
					}
					ui.Success("Created profile: " + args[0])
					return nil
				},
			},
			{
				Name:        "delete",
				Description: "Delete a profile",
				Usage:       "manpm profile delete <name>",
				Run: func(args []string) error {
					if len(args) == 0 {
						return fmt.Errorf("usage: manpm profile delete <name>")
					}
					dir, _ := os.Getwd()
					cfg, err := config.LoadConfig(dir)
					if err != nil {
						return err
					}
					if !cfg.DeleteProfile(args[0]) {
						return fmt.Errorf("profile %q not found", args[0])
					}
					if err := cfg.Save(dir); err != nil {
						return fmt.Errorf("save config: %w", err)
					}
					ui.Success("Deleted profile: " + args[0])
					return nil
				},
			},
		},
		Run: func(args []string) error {
			if len(args) == 0 {
				return fmt.Errorf("usage: manpm profile <subcommand> [name]\n\nSubcommands: list, use, create, delete")
			}
			return dispatchSub(profileCmd, args)
		},
	}

	root.Subcommands = []Command{
		installCmd,
		addCmd,
		explainCmd,
		auditCmd,
		doctorCmd,
		mapCmd,
		entropyCmd,
		pruneCmd,
		runCmd,
		sandboxCmd,
		compareCmd,
		senseiCmd,
		profileCmd,
	}

	return root
}

var aliases = map[string]string{
	"i":     "install",
	"in":    "install",
	"inst":  "install",
	"r":     "run",
	"ad":    "add",
	"ex":    "explain",
	"au":    "audit",
	"doc":   "doctor",
	"pr":    "prune",
	"sb":    "sandbox",
	"cmp":   "compare",
	"pro":   "profile",
	"se":    "sensei",
	"ls":    "map",
}

func dispatch(root Command, args []string) error {
	if len(args) == 0 {
		return root.Subcommands[0].Run(nil)
	}

	name := args[0]
	if full, ok := aliases[name]; ok {
		name = full
	}
	rest := args[1:]

	for _, cmd := range root.Subcommands {
		if cmd.Name == name {
			if len(cmd.Subcommands) > 0 {
				return dispatchSub(cmd, rest)
			}
			return cmd.Run(rest)
		}
	}

	suggestions := findSuggestions(root, name)
	ui.Errorf("Unknown command: %s", name)
	if len(suggestions) > 0 {
		ui.Info(fmt.Sprintf("Did you mean: %s?", strings.Join(suggestions, ", ")))
	}
	fmt.Println()
	ui.Header("Available commands")
	for _, cmd := range root.Subcommands {
		fmt.Printf("  %s%-13s%s %s\n", ui.Bold, cmd.Name+":", ui.Reset, cmd.Description)
	}
	return fmt.Errorf("unknown command: %s", name)
}

func dispatchSub(cmd Command, args []string) error {
	if len(args) == 0 {
		return fmt.Errorf("usage: %s", cmd.Usage)
	}

	name := args[0]
	rest := args[1:]

	for _, sub := range cmd.Subcommands {
		if sub.Name == name {
			return sub.Run(rest)
		}
	}

	return fmt.Errorf("unknown subcommand: %s %s\n\nSubcommands: list, use, create, delete", cmd.Name, name)
}

func findSuggestions(root Command, input string) []string {
	var suggestions []string
	input = strings.ToLower(input)
	for _, cmd := range root.Subcommands {
		name := strings.ToLower(cmd.Name)
		if strings.HasPrefix(name, input) || levenshtein(input, name) <= 2 {
			suggestions = append(suggestions, cmd.Name)
		}
	}
	return suggestions
}

func levenshtein(a, b string) int {
	la, lb := len(a), len(b)
	if la == 0 {
		return lb
	}
	if lb == 0 {
		return la
	}
	d := make([][]int, la+1)
	for i := range d {
		d[i] = make([]int, lb+1)
		d[i][0] = i
	}
	for j := 0; j <= lb; j++ {
		d[0][j] = j
	}
	for i := 1; i <= la; i++ {
		for j := 1; j <= lb; j++ {
			cost := 1
			if a[i-1] == b[j-1] {
				cost = 0
			}
			d[i][j] = min(d[i-1][j]+1, min(d[i][j-1]+1, d[i-1][j-1]+cost))
		}
	}
	return d[la][lb]
}

func min(a, b int) int {
	if a < b {
		return a
	}
	return b
}

var importRe = regexp.MustCompile(`(?:import\s+(?:\w+\s+from\s+)?["']([^"']+)["']|require\(["']([^"']+)["']\)|import\(["']([^"']+)["']\))`)

func scanImports(dir string) map[string]bool {
	imported := map[string]bool{}
	skipDirs := map[string]bool{"node_modules": true, ".git": true, ".svn": true, "dist": true, ".next": true, "build": true, ".cache": true, "coverage": true}
	exts := map[string]bool{".js": true, ".ts": true, ".mjs": true, ".cjs": true, ".jsx": true, ".tsx": true, ".mts": true, ".cts": true}

	filepath.Walk(dir, func(path string, info os.FileInfo, err error) error {
		if err != nil || info.IsDir() {
			if info != nil && info.IsDir() && skipDirs[info.Name()] {
				return filepath.SkipDir
			}
			return nil
		}
		if !exts[filepath.Ext(info.Name())] {
			return nil
		}
		f, err := os.Open(path)
		if err != nil {
			return nil
		}
		defer f.Close()
		scanner := bufio.NewScanner(f)
		scanner.Buffer(make([]byte, 1024*1024), 1024*1024)
		for scanner.Scan() {
			line := scanner.Text()
			matches := importRe.FindStringSubmatch(line)
			if matches == nil {
				continue
			}
			pkg := matches[1]
			if pkg == "" {
				pkg = matches[2]
			}
			if pkg == "" {
				pkg = matches[3]
			}
			if pkg == "" || strings.HasPrefix(pkg, ".") || strings.HasPrefix(pkg, "/") {
				continue
			}
			parts := strings.SplitN(pkg, "/", 2)
			if strings.HasPrefix(pkg, "@") && len(parts) > 1 {
				imported[parts[0]+"/"+parts[1]] = true
			} else {
				imported[parts[0]] = true
			}
		}
		return nil
	})
	return imported
}

func titleCase(s string) string {
	runes := []rune(s)
	for i, r := range runes {
		if i == 0 || runes[i-1] == '_' || runes[i-1] == '-' {
			runes[i] = unicode.ToUpper(r)
		}
	}
	return string(runes)
}
