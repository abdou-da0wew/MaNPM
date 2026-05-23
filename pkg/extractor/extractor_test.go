package extractor

import (
	"archive/tar"
	"bytes"
	"compress/gzip"
	"context"
	"crypto/sha512"
	"encoding/base64"
	"io"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func buildTestTarGz(entries map[string]string) *bytes.Buffer {
	var buf bytes.Buffer
	gz := gzip.NewWriter(&buf)
	tw := tar.NewWriter(gz)

	for path, content := range entries {
		if content == "" {
			tw.WriteHeader(&tar.Header{
				Name:     path,
				Typeflag: tar.TypeDir,
				Mode:     0755,
			})
		} else {
			tw.WriteHeader(&tar.Header{
				Name:     path,
				Typeflag: tar.TypeReg,
				Mode:     0644,
				Size:     int64(len(content)),
			})
			io.WriteString(tw, content)
		}
	}

	tw.Close()
	gz.Close()
	return &buf
}

func checkFileContent(t *testing.T, path, expected string) {
	t.Helper()
	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("read %s: %v", path, err)
	}
	if string(data) != expected {
		t.Fatalf("%s content = %q, want %q", path, string(data), expected)
	}
}

func TestParseIntegrity(t *testing.T) {
	hash := sha512.Sum512([]byte("test data"))
	encoded := base64.StdEncoding.EncodeToString(hash[:])
	valid := "sha512-" + encoded

	got, err := parseIntegrity(valid)
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}
	if !bytes.Equal(got, hash[:]) {
		t.Fatalf("expected hash %x, got %x", hash[:], got)
	}

	_, err = parseIntegrity("sha256-abc")
	if err == nil {
		t.Fatal("expected error for unsupported prefix")
	}

	_, err = parseIntegrity("sha512-!!!invalidbase64")
	if err == nil {
		t.Fatal("expected error for invalid base64")
	}
}

func TestTarPathSanitize(t *testing.T) {
	tests := []struct {
		input    string
		expected string
	}{
		{"package/index.js", "package/index.js"},
		{".", ""},
		{"/", ""},
		{"foo/../bar", "bar"},
		{"a/b/c", "a/b/c"},
		{"", ""},
	}
	for _, tt := range tests {
		got := sanitizeTarPath(tt.input)
		if got != tt.expected {
			t.Errorf("sanitizeTarPath(%q) = %q, want %q", tt.input, got, tt.expected)
		}
	}
}

func TestStripPackagePrefix(t *testing.T) {
	tests := []struct {
		input    string
		expected string
	}{
		{"package/index.js", "index.js"},
		{"package/foo/bar", "foo/bar"},
		{"index.js", "index.js"},
		{"foo/bar", "foo/bar"},
		{"package", "package"},
	}
	for _, tt := range tests {
		got := stripPackagePrefix(tt.input)
		if got != tt.expected {
			t.Errorf("stripPackagePrefix(%q) = %q, want %q", tt.input, got, tt.expected)
		}
	}
}

func TestExtractPackage(t *testing.T) {
	pkgJSON := `{"name":"test-pkg","bin":{"test-bin":"./index.js"}}`
	indexJS := `module.exports = function() { return 42; }`

	tarballContent := buildTestTarGz(map[string]string{
		"package/":             "",
		"package/index.js":     indexJS,
		"package/package.json": pkgJSON,
	})

	raw := tarballContent.Bytes()
	h := sha512.Sum512(raw)
	integrity := "sha512-" + base64.StdEncoding.EncodeToString(h[:])

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Write(raw)
	}))
	defer server.Close()

	baseDir := t.TempDir()
	extractor := NewExtractor(baseDir, 1)

	err := extractor.ExtractPackage(context.Background(), PackageJob{
		Name:       "test-pkg",
		Path:       "test-pkg",
		TarballURL: server.URL,
		Integrity:  integrity,
	})
	if err != nil {
		t.Fatalf("ExtractPackage: %v", err)
	}

	targetDir := filepath.Join(baseDir, "node_modules", "test-pkg")

	checkFileContent(t, filepath.Join(targetDir, "index.js"), indexJS)
	checkFileContent(t, filepath.Join(targetDir, "package.json"), pkgJSON)
}

func TestExtractPackageIntegrityFail(t *testing.T) {
	tarballContent := buildTestTarGz(map[string]string{
		"package/":         "",
		"package/index.js": "content",
	})
	raw := tarballContent.Bytes()

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Write(raw)
	}))
	defer server.Close()

	baseDir := t.TempDir()
	extractor := NewExtractor(baseDir, 1)

	err := extractor.ExtractPackage(context.Background(), PackageJob{
		Name:       "bad-pkg",
		Path:       "bad-pkg",
		TarballURL: server.URL,
		Integrity:  "sha512-" + base64.StdEncoding.EncodeToString(make([]byte, 64)),
	})
	if err == nil {
		t.Fatal("expected integrity error, got nil")
	}
	if !strings.Contains(err.Error(), "integrity") {
		t.Fatalf("expected integrity error, got: %v", err)
	}
}

func TestExtractLevel(t *testing.T) {
	pkgJSON := `{"name":"pkg-a","bin":{"a":"./index.js"}}`
	indexJS := `// a`

	tarballContent := buildTestTarGz(map[string]string{
		"package/":             "",
		"package/index.js":     indexJS,
		"package/package.json": pkgJSON,
	})
	raw := tarballContent.Bytes()
	h := sha512.Sum512(raw)
	integrity := "sha512-" + base64.StdEncoding.EncodeToString(h[:])

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Write(raw)
	}))
	defer server.Close()

	baseDir := t.TempDir()
	extractor := NewExtractor(baseDir, 4)

	jobs := []PackageJob{
		{Name: "pkg-a", Path: "pkg-a", TarballURL: server.URL, Integrity: integrity},
		{Name: "pkg-b", Path: "pkg-b", TarballURL: server.URL, Integrity: integrity},
	}

	results := extractor.ExtractLevel(context.Background(), jobs)
	if len(results) != 2 {
		t.Fatalf("expected 2 results, got %d", len(results))
	}

	for _, r := range results {
		if r.Error != nil {
			t.Errorf("package %s error: %v", r.PackageName, r.Error)
		}
	}

	for _, job := range jobs {
		targetDir := filepath.Join(baseDir, "node_modules", job.Path)
		if _, err := os.Stat(targetDir); os.IsNotExist(err) {
			t.Errorf("expected %s to exist", targetDir)
		}
	}
}

func TestZipSlipPrevention(t *testing.T) {
	tarballContent := buildTestTarGz(map[string]string{
		"package/":                     "",
		"package/index.js":            "good",
		"package/package.json":        "{}",
		"../../../etc/passwd":         "escaped",
		"package/../../../etc/passwd": "escaped2",
	})
	raw := tarballContent.Bytes()
	h := sha512.Sum512(raw)
	integrity := "sha512-" + base64.StdEncoding.EncodeToString(h[:])

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Write(raw)
	}))
	defer server.Close()

	baseDir := t.TempDir()
	extractor := NewExtractor(baseDir, 1)

	err := extractor.ExtractPackage(context.Background(), PackageJob{
		Name:       "pkg",
		Path:       "pkg",
		TarballURL: server.URL,
		Integrity:  integrity,
	})
	if err != nil {
		t.Fatalf("ExtractPackage: %v", err)
	}

	targetDir := filepath.Join(baseDir, "node_modules", "pkg")

	checkFileContent(t, filepath.Join(targetDir, "index.js"), "good")

	escapedPasswd := filepath.Join(baseDir, "..", "etc", "passwd")
	if _, err := os.Stat(escapedPasswd); err == nil {
		t.Fatal("zip slip: ../../../etc/passwd was extracted outside target dir")
	}
}
