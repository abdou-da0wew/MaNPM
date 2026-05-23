package ui

import (
	"strings"
	"testing"
)

func TestError(t *testing.T) {
	Error("test error")
}

func TestErrorf(t *testing.T) {
	Errorf("error %d: %s", 42, "something")
}

func TestWarning(t *testing.T) {
	Warning("test warning")
}

func TestSuccess(t *testing.T) {
	Success("test success")
}

func TestInfo(t *testing.T) {
	Info("test info")
}

func TestHeader(t *testing.T) {
	Header("Test Title")
}

func TestSubheader(t *testing.T) {
	Subheader("Test Sub")
}

func TestLabel(t *testing.T) {
	Label("Name", "value")
}

func TestLabelEmptyValue(t *testing.T) {
	Label("Key", "")
}

func TestBoldText(t *testing.T) {
	r := BoldText("hello")
	if !strings.Contains(r, "hello") {
		t.Errorf("expected hello in output, got %q", r)
	}
}

func TestColorize(t *testing.T) {
	r := Colorize(Red, "danger")
	if !strings.Contains(r, "danger") {
		t.Errorf("expected danger in output, got %q", r)
	}
}

func TestDim(t *testing.T) {
	r := Dim("faint")
	if !strings.Contains(r, "faint") {
		t.Errorf("expected faint in output, got %q", r)
	}
}

func TestSeparator(t *testing.T) {
	Separator()
}

func TestBoldTextReset(t *testing.T) {
	r := BoldText("x")
	if !strings.HasSuffix(r, Reset) {
		t.Errorf("expected Reset suffix, got %q", r)
	}
}

func TestColorizeReset(t *testing.T) {
	r := Colorize(Red, "x")
	if !strings.HasSuffix(r, Reset) {
		t.Errorf("expected Reset suffix, got %q", r)
	}
}

func TestDimReset(t *testing.T) {
	r := Dim("x")
	if !strings.HasSuffix(r, Reset) {
		t.Errorf("expected Reset suffix, got %q", r)
	}
}

func TestHeaderFormat(t *testing.T) {
	Header("")
}
