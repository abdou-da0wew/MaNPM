package platform

import (
	"testing"
)

func TestDetectOS(t *testing.T) {
	os := DetectOS()
	switch os {
	case Windows, Linux, Darwin, Android:
	default:
		t.Errorf("DetectOS() returned unexpected OS value: %q", os)
	}
}

func TestDetectArch(t *testing.T) {
	arch := DetectArch()
	switch arch {
	case ArchARM, ArchARM64, ArchAMD64, Arch386:
	default:
		t.Errorf("DetectArch() returned unexpected Arch value: %q", arch)
	}
}

func TestOptimalWorkerCount(t *testing.T) {
	n := OptimalWorkerCount()
	if n < 1 {
		t.Errorf("OptimalWorkerCount() = %d, want at least 1", n)
	}
}

func TestSupportsSymlinks(t *testing.T) {
	_, ok := interface{}(SupportsSymlinks()).(bool)
	if !ok {
		t.Errorf("SupportsSymlinks() did not return a bool")
	}
}

func TestTempDir(t *testing.T) {
	dir := TempDir()
	if dir == "" {
		t.Errorf("TempDir() returned empty string")
	}
}

func TestIsARM(t *testing.T) {
	_, ok := interface{}(IsARM()).(bool)
	if !ok {
		t.Errorf("IsARM() did not return a bool")
	}
}

func TestIsAndroid(t *testing.T) {
	val := IsAndroid()
	if val {
		t.Log("IsAndroid() returned true (unusual but possible in this environment)")
	}
}
