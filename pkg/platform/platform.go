package platform

import (
	"fmt"
	"os"
	"runtime"
	"strings"
)

type OS string

const (
	Windows OS = "windows"
	Linux   OS = "linux"
	Darwin  OS = "darwin"
	Android OS = "android"
)

type Arch string

const (
	ArchARM   Arch = "arm"
	ArchARM64 Arch = "arm64"
	ArchAMD64 Arch = "amd64"
	Arch386   Arch = "386"
)

func DetectOS() OS {
	switch runtime.GOOS {
	case "windows":
		return Windows
	case "darwin":
		return Darwin
	case "linux":
		if _, err := os.Stat("/system/build.prop"); err == nil {
			return Android
		}
		return Linux
	default:
		return Linux
	}
}

func DetectArch() Arch {
	return Arch(runtime.GOARCH)
}

func IsARM() bool {
	arch := DetectArch()
	return arch == ArchARM || arch == ArchARM64
}

func IsAndroid() bool {
	return DetectOS() == Android
}

func OptimalWorkerCount() int {
	cpu := runtime.NumCPU()
	mem := approxRAMGB()

	if IsARM() && mem <= 1 {
		if cpu > 2 {
			return 2
		}
		return cpu
	}
	if IsARM() && mem <= 2 {
		if cpu > 4 {
			return 4
		}
		return cpu
	}
	if mem <= 1 {
		if cpu > 2 {
			return 2
		}
		return cpu
	}
	if cpu > 1 {
		return cpu
	}
	return 1
}

func approxRAMGB() int {
	if IsAndroid() {
		return androidApproxRAM()
	}
	return 4
}

func androidApproxRAM() int {
	data, err := os.ReadFile("/proc/meminfo")
	if err != nil {
		return 2
	}
	lines := strings.Split(string(data), "\n")
	for _, line := range lines {
		if strings.HasPrefix(line, "MemTotal:") {
			fields := strings.Fields(line)
			if len(fields) >= 2 {
				var kb int
				_, err := fmt.Sscanf(fields[1], "%d", &kb)
				if err == nil && kb > 0 {
					return kb / (1024 * 1024)
				}
			}
		}
	}
	return 2
}

func DefaultConcurrencyLimit() int {
	workers := OptimalWorkerCount()
	if workers < 1 {
		return 1
	}
	return workers
}

func SupportsSymlinks() bool {
	if runtime.GOOS == "windows" {
		return os.Getenv("MANPM_FORCE_SYMLINKS") != ""
	}
	return true
}

func TempDir() string {
	if IsAndroid() {
		dir := os.Getenv("MANPM_TMPDIR")
		if dir != "" {
			return dir
		}
		return "/data/local/tmp/manpm"
	}
	return os.TempDir()
}
