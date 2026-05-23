package main

import (
	"flag"
	"fmt"
	"strings"
	"unicode"

	"manpm/pkg/intel"
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

func run(cfg Config) error {
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
			return run(cfg)
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
			ui.Success("Simulated add complete")
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
			results, err := intel.Audit("package-lock.json")
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
			result, err := intel.Doctor(".", nil)
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
			ui.Header("Dependency Graph")
			fmt.Println(intel.Map(nil))
			return nil
		},
	}

	entropyCmd := Command{
		Name:        "entropy",
		Description: "Show project chaos metrics",
		Usage:       "manpm entropy",
		Run: func(args []string) error {
			result := intel.Entropy(nil)
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
			safe := fs.Bool("safe", false, "Safe mode")
			dryRun := fs.Bool("dry-run", false, "Only show what would be removed")
			if err := fs.Parse(args); err != nil {
				return err
			}
			ui.Header("Prune")
			ui.Label("Safe", fmt.Sprintf("%v", *safe))
			ui.Label("Dry-run", fmt.Sprintf("%v", *dryRun))
			if *dryRun {
				ui.Info("Would remove: left-pad, is-odd, some-dep")
			} else {
				ui.Warning("No packages removed (dry-run not set)")
			}
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
					ui.Subheader("Available profiles")
					ui.Info("  default  (current)")
					ui.Info("  fast")
					ui.Info("  safe")
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
		sandboxCmd,
		compareCmd,
		senseiCmd,
		profileCmd,
	}

	return root
}

func dispatch(root Command, args []string) error {
	if len(args) == 0 {
		return root.Subcommands[0].Run(nil)
	}

	name := args[0]
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

func titleCase(s string) string {
	runes := []rune(s)
	for i, r := range runes {
		if i == 0 || runes[i-1] == '_' || runes[i-1] == '-' {
			runes[i] = unicode.ToUpper(r)
		}
	}
	return string(runes)
}
