package main

import (
	"fmt"
	"os"

	"manpm/pkg/ui"
)

func main() {
	cmd := buildRouter()
	args := os.Args[1:]

	if len(args) == 0 || args[0] == "--help" || args[0] == "-h" {
		printHelp(cmd)
		return
	}

	if err := dispatch(cmd, args); err != nil {
		ui.Errorf("%v", err)
		os.Exit(1)
	}
}

func printHelp(cmd Command) {
	ui.Header("MaNPM - The blazing-fast Go orchestrator")
	ui.Label("Usage", cmd.Usage)
	fmt.Println()
	ui.Subheader("Commands")
	for _, sub := range cmd.Subcommands {
		fmt.Printf("  %s%-12s%s %s\n", ui.Bold, sub.Name, ui.Reset, sub.Description)
	}
	fmt.Println()
	ui.Label("Docs", "https://github.com/abdou-da0wew/MaNPM")
}
