package extractor

import (
	"archive/tar"
	"bytes"
	"compress/gzip"
	"context"
	"crypto/sha512"
	"encoding/base64"
	"fmt"
	"hash"
	"io"
	"net/http"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"sync"
	"time"

	"manpm/pkg/platform"
)

type PackageJob struct {
	Name       string
	Path       string
	TarballURL string
	Integrity  string
}

type ExtractResult struct {
	PackageName string
	Error       error
}

type Extractor struct {
	Client      *http.Client
	BaseDir     string
	NumWorkers  int
	Concurrency int
	FallbackDir string
	MaxRetries  int
	OnProgress  func(completed, total int, name string, err error)
}

func NewExtractor(baseDir string, numWorkers int) *Extractor {
	if numWorkers < 1 {
		numWorkers = platform.DefaultConcurrencyLimit()
	}

	tr := &http.Transport{
		MaxIdleConnsPerHost:   numWorkers * 3,
		IdleConnTimeout:       90 * time.Second,
		TLSHandshakeTimeout:   30 * time.Second,
		ResponseHeaderTimeout: 60 * time.Second,
	}

	return &Extractor{
		Client: &http.Client{
			Transport: tr,
			Timeout:   0,
		},
		BaseDir:     baseDir,
		NumWorkers:  numWorkers,
		Concurrency: numWorkers,
		FallbackDir: platform.TempDir(),
		MaxRetries:  3,
	}
}

func (e *Extractor) ExtractPackage(ctx context.Context, job PackageJob) error {
	nodeModulesDir := filepath.Join(e.BaseDir, "node_modules")
	targetDir := filepath.Join(nodeModulesDir, job.Path)

	if err := os.MkdirAll(targetDir, 0755); err != nil {
		return &ExtractError{Op: "mkdir", Pkg: job.Name, Err: err}
	}

	if err := e.downloadAndExtract(ctx, job, targetDir); err != nil {
		os.RemoveAll(targetDir)
		return err
	}

	return nil
}

type ExtractError struct {
	Op  string
	Pkg string
	Err error
}

func (e *ExtractError) Error() string {
	return fmt.Sprintf("%s failed for %s: %v", e.Op, e.Pkg, e.Err)
}

func (e *ExtractError) Unwrap() error {
	return e.Err
}

func (e *Extractor) downloadAndExtract(ctx context.Context, job PackageJob, targetDir string) error {
	var lastErr error

	for attempt := 0; attempt <= e.MaxRetries; attempt++ {
		if attempt > 0 {
			backoff := time.Duration(attempt*attempt) * time.Second
			select {
			case <-time.After(backoff):
			case <-ctx.Done():
				return ctx.Err()
			}
		}

		err := e.tryDownloadExtract(ctx, job, targetDir)
		if err == nil {
			return nil
		}

		lastErr = err

		if isRetryable(err) {
			continue
		}

		if strings.Contains(err.Error(), "integrity") {
			return err
		}

		break
	}

	return lastErr
}

func isRetryable(err error) bool {
	msg := err.Error()
	return strings.Contains(msg, "timeout") ||
		strings.Contains(msg, "connection") ||
		strings.Contains(msg, "reset") ||
		strings.Contains(msg, "refused") ||
		strings.Contains(msg, "EOF") ||
		strings.Contains(msg, "TLS") ||
		strings.Contains(msg, "HTTP 5")
}

func (e *Extractor) tryDownloadExtract(ctx context.Context, job PackageJob, targetDir string) error {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, job.TarballURL, nil)
	if err != nil {
		return &ExtractError{Op: "request", Pkg: job.Name, Err: err}
	}

	resp, err := e.Client.Do(req)
	if err != nil {
		return &ExtractError{Op: "download", Pkg: job.Name, Err: err}
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(io.LimitReader(resp.Body, 1024))
		return &ExtractError{
			Op:  "download",
			Pkg: job.Name,
			Err: fmt.Errorf("HTTP %d: %s", resp.StatusCode, string(body)),
		}
	}

	hasher := sha512.New()
	teeReader := io.TeeReader(resp.Body, hasher)

	gzReader, err := gzip.NewReader(teeReader)
	if err != nil {
		return &ExtractError{Op: "gzip", Pkg: job.Name, Err: err}
	}
	defer gzReader.Close()

	tarReader := tar.NewReader(gzReader)

	var savedPkgJSON bytes.Buffer
	extractedFiles := make([]string, 0)

	for {
		header, err := tarReader.Next()
		if err == io.EOF {
			break
		}
		if err != nil {
			return &ExtractError{Op: "tar", Pkg: job.Name, Err: err}
		}

		cleanPath := sanitizeTarPath(header.Name)
		if cleanPath == "" {
			continue
		}

		cleanPath = stripPackagePrefix(cleanPath)
		if cleanPath == "" {
			continue
		}

		fullPath := filepath.Join(targetDir, cleanPath)

		if !isPathSafe(targetDir, fullPath) {
			continue
		}

		switch header.Typeflag {
		case tar.TypeDir:
			if err := os.MkdirAll(fullPath, 0755); err != nil {
				return &ExtractError{Op: "mkdir", Pkg: job.Name, Err: err}
			}

		case tar.TypeReg, tar.TypeRegA:
			if err := os.MkdirAll(filepath.Dir(fullPath), 0755); err != nil {
				return &ExtractError{Op: "mkdir", Pkg: job.Name, Err: err}
			}
			f, err := os.OpenFile(fullPath, os.O_CREATE|os.O_WRONLY|os.O_TRUNC, header.FileInfo().Mode())
			if err != nil {
				return &ExtractError{Op: "create", Pkg: job.Name, Err: err}
			}
			if _, err := io.Copy(f, tarReader); err != nil {
				f.Close()
				os.Remove(fullPath)
				return &ExtractError{Op: "write", Pkg: job.Name, Err: err}
			}
			f.Close()
			extractedFiles = append(extractedFiles, fullPath)

			if strings.HasSuffix(cleanPath, "package.json") {
				pkgData, _ := os.ReadFile(fullPath)
				if pkgData != nil {
					savedPkgJSON.Reset()
					savedPkgJSON.Write(pkgData)
				}
			}

		case tar.TypeSymlink:
			linkTarget := header.Linkname
			if err := os.MkdirAll(filepath.Dir(fullPath), 0755); err != nil {
				return &ExtractError{Op: "mkdir-symlink", Pkg: job.Name, Err: err}
			}
			if platform.SupportsSymlinks() {
				os.Remove(fullPath)
				if err := os.Symlink(linkTarget, fullPath); err != nil {
					return &ExtractError{Op: "symlink", Pkg: job.Name, Err: err}
				}
			}
			extractedFiles = append(extractedFiles, fullPath)

		case tar.TypeLink:
			linkTarget := filepath.Join(targetDir, header.Linkname)
			if err := os.MkdirAll(filepath.Dir(fullPath), 0755); err != nil {
				return &ExtractError{Op: "mkdir-hardlink", Pkg: job.Name, Err: err}
			}
			if err := os.Link(linkTarget, fullPath); err != nil {
				return &ExtractError{Op: "hardlink", Pkg: job.Name, Err: err}
			}
			extractedFiles = append(extractedFiles, fullPath)
		}
	}

	if err := verifyIntegrity(hasher, job.Integrity); err != nil {
		for _, f := range extractedFiles {
			os.Remove(f)
		}
		return err
	}

	if savedPkgJSON.Len() > 0 {
		pkgJSONPath := filepath.Join(targetDir, "package.json")
		os.WriteFile(pkgJSONPath, savedPkgJSON.Bytes(), 0644)
	}

	return nil
}

func isPathSafe(targetDir, fullPath string) bool {
	targetDir = filepath.Clean(targetDir)
	fullPath = filepath.Clean(fullPath)
	return strings.HasPrefix(fullPath, targetDir+string(filepath.Separator)) || fullPath == targetDir
}

func (e *Extractor) ExtractLevel(ctx context.Context, jobs []PackageJob) []ExtractResult {
	if len(jobs) == 0 {
		return nil
	}

	ctx, cancel := context.WithCancel(ctx)
	defer cancel()

	jobCh := make(chan PackageJob, len(jobs))
	resultCh := make(chan ExtractResult, len(jobs))

	workerCount := e.Concurrency
	if workerCount < 1 {
		workerCount = 1
	}
	if workerCount > len(jobs) {
		workerCount = len(jobs)
	}

	var wg sync.WaitGroup
	for w := 0; w < workerCount; w++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			for job := range jobCh {
				err := e.ExtractPackage(ctx, job)
				if err != nil {
					err = e.fallbackInstall(ctx, job, err)
				}
				resultCh <- ExtractResult{PackageName: job.Name, Error: err}
			}
		}()
	}

	for _, job := range jobs {
		jobCh <- job
	}
	close(jobCh)

	wg.Wait()
	close(resultCh)

	var results []ExtractResult
	completed := 0
	for r := range resultCh {
		results = append(results, r)
		completed++
		if e.OnProgress != nil {
			e.OnProgress(completed, len(jobs), r.PackageName, r.Error)
		}
	}
	return results
}

func (e *Extractor) fallbackInstall(ctx context.Context, job PackageJob, originalErr error) error {
	tmpCache := filepath.Join(e.FallbackDir, fmt.Sprintf("%x", []byte(job.Name)))
	os.MkdirAll(tmpCache, 0755)

	args := []string{
		"install", job.Name + "@" + extractVersionFromPath(job.Path),
		"--cache=" + tmpCache,
		"--no-audit",
		"--ignore-scripts",
		"--no-package-lock",
		"--no-save",
		"--prefix", e.BaseDir,
	}

	cmd := exec.CommandContext(ctx, "npm", args...)
	cmd.Dir = e.BaseDir

	var stderr bytes.Buffer
	cmd.Stderr = &stderr

	output, err := cmd.CombinedOutput()
	if err != nil {
		return fmt.Errorf("extraction failed: %v; npm fallback also failed: %w\nstderr: %s\noutput: %s",
			originalErr, err, stderr.String(), string(output))
	}

	return nil
}

func extractVersionFromPath(pkgPath string) string {
	parts := strings.SplitN(pkgPath, "/", 2)
	return parts[0]
}

func parseIntegrity(integrity string) ([]byte, error) {
	if !strings.HasPrefix(integrity, "sha512-") {
		return nil, fmt.Errorf("unsupported integrity format: %s", integrity)
	}
	encoded := strings.TrimPrefix(integrity, "sha512-")
	decoded, err := base64.StdEncoding.DecodeString(encoded)
	if err != nil {
		decoded, err = base64.StdEncoding.WithPadding(base64.NoPadding).DecodeString(encoded)
		if err != nil {
			return nil, fmt.Errorf("decode integrity: %w", err)
		}
	}
	return decoded, nil
}

func verifyIntegrity(h hash.Hash, integrity string) error {
	if integrity == "" {
		return nil
	}

	expected, err := parseIntegrity(integrity)
	if err != nil {
		return err
	}

	computed := h.Sum(nil)
	if !bytes.Equal(computed, expected) {
		return fmt.Errorf("integrity mismatch")
	}
	return nil
}

func sanitizeTarPath(tarPath string) string {
	clean := filepath.Clean(tarPath)
	if clean == "." || clean == "/" {
		return ""
	}
	return clean
}

func stripPackagePrefix(path string) string {
	parts := strings.SplitN(path, "/", 2)
	if len(parts) == 2 && parts[0] == "package" {
		return parts[1]
	}
	return path
}
