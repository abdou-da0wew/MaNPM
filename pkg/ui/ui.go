package ui

import (
	"fmt"
	"os"
	"strings"
)

const (
	Reset   = "\033[0m"
	Bold    = "\033[1m"
	Red     = "\033[31m"
	Green   = "\033[32m"
	Yellow  = "\033[33m"
	Cyan    = "\033[36m"
	Magenta = "\033[35m"
	White   = "\033[97m"
	Orange  = "\033[38;5;208m"
	Gray    = "\033[90m"
)

func Error(msg string) {
	fmt.Fprintf(os.Stderr, "%s%s%s %s\n", Bold, Red, "✖", Reset+msg)
}

func Errorf(format string, args ...interface{}) {
	Error(fmt.Sprintf(format, args...))
}

func Warning(msg string) {
	fmt.Fprintf(os.Stderr, "%s%s%s %s\n", Bold, Yellow, "⚠", Reset+msg)
}

func Success(msg string) {
	fmt.Printf("%s%s%s %s\n", Bold, Green, "✓", Reset+msg)
}

func Info(msg string) {
	fmt.Printf("%s%s%s %s\n", Bold, Cyan, "ℹ", Reset+msg)
}

func Header(title string) {
	line := strings.Repeat("─", len(title)+4)
	fmt.Printf("\n%s%s%s\n", Bold, Orange, line)
	fmt.Printf("%s  %s%s\n", Bold, White+title, Reset)
	fmt.Printf("%s%s%s\n", Orange, line, Reset)
}

func Subheader(title string) {
	fmt.Printf("\n%s%s%s%s\n", Bold, Cyan, "▸ ", Reset+title)
}

func Label(k, v string) {
	fmt.Printf("  %s%-12s%s %s\n", Bold+Gray, k+":", Reset, v)
}

func BoldText(s string) string {
	return Bold + s + Reset
}

func Colorize(color, s string) string {
	return color + s + Reset
}

func Dim(s string) string {
	return Gray + s + Reset
}

func Separator() {
	fmt.Println(strings.Repeat("─", 50))
}
