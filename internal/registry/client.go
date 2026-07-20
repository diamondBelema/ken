package registry

import (
	"archive/tar"
	"compress/gzip"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"time"
)

func FetchIndex() (RegistryIndex, error) {
	return FetchIndexFromURL(registryIndexURL)
}

func FetchIndexFromURL(url string) (RegistryIndex, error) {
	client := &http.Client{Timeout: 15 * time.Second}

	resp, err := client.Get(url)
	if err != nil {
		return RegistryIndex{}, fmt.Errorf("failed to fetch registry index: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return RegistryIndex{}, fmt.Errorf("registry index returned status %d", resp.StatusCode)
	}

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return RegistryIndex{}, fmt.Errorf("failed to read registry response: %w", err)
	}

	// GitHub API returns base64-encoded content
	var apiResp struct {
		Content string `json:"content"`
	}
	if err := json.Unmarshal(body, &apiResp); err == nil && apiResp.Content != "" {
		decoded, err := base64.StdEncoding.DecodeString(strings.TrimSpace(apiResp.Content))
		if err != nil {
			return RegistryIndex{}, fmt.Errorf("failed to decode registry content: %w", err)
		}
		body = decoded
	}

	var index RegistryIndex
	if err := json.Unmarshal(body, &index); err != nil {
		return RegistryIndex{}, fmt.Errorf("failed to parse registry index: %w", err)
	}

	return index, nil
}

func SearchIndex(index RegistryIndex, query string) []Package {
	var results []Package
	q := toLower(query)

	for _, pkg := range index.Packages {
		if matchesQuery(pkg, q) {
			results = append(results, pkg)
		}
	}
	return results
}

func matchesQuery(pkg Package, q string) bool {
	if strings.Contains(toLower(pkg.Name), q) {
		return true
	}
	if strings.Contains(toLower(pkg.ID), q) {
		return true
	}
	if strings.Contains(toLower(pkg.Description), q) {
		return true
	}
	for _, tag := range pkg.Tags {
		if strings.Contains(toLower(tag), q) {
			return true
		}
	}
	return false
}

func toLower(s string) string {
	b := make([]byte, len(s))
	for i := 0; i < len(s); i++ {
		c := s[i]
		if c >= 'A' && c <= 'Z' {
			c += 'a' - 'A'
		}
		b[i] = c
	}
	return string(b)
}

func tarballURL(owner, repo, version string) string {
	if version == "" || version == "latest" {
		return fmt.Sprintf("https://github.com/%s/%s/archive/refs/heads/main.tar.gz", owner, repo)
	}
	return fmt.Sprintf("https://github.com/%s/%s/archive/refs/tags/%s.tar.gz", owner, repo, version)
}

func DownloadTarball(owner, repo, version string) (io.ReadCloser, error) {
	// Try main first, then master
	branches := []string{"main", "master"}
	
	if version != "" && version != "latest" {
		// Specific version — try as tag first
		url := fmt.Sprintf("https://github.com/%s/%s/archive/refs/tags/%s.tar.gz", owner, repo, version)
		client := &http.Client{Timeout: 60 * time.Second}
		resp, err := client.Get(url)
		if err == nil && resp.StatusCode == http.StatusOK {
			return resp.Body, nil
		}
		if resp != nil {
			resp.Body.Close()
		}
	}

	for _, branch := range branches {
		url := fmt.Sprintf("https://github.com/%s/%s/archive/refs/heads/%s.tar.gz", owner, repo, branch)
		client := &http.Client{Timeout: 60 * time.Second}
		resp, err := client.Get(url)
		if err != nil {
			continue
		}
		if resp.StatusCode == http.StatusOK {
			return resp.Body, nil
		}
		resp.Body.Close()
	}

	return nil, fmt.Errorf("package %s/%s@%s not found", owner, repo, versionLabel(version))
}

func ExtractTarball(reader io.Reader, destDir string, stripTopLevel bool) error {
	gz, err := gzip.NewReader(reader)
	if err != nil {
		return fmt.Errorf("failed to decompress: %w", err)
	}
	defer gz.Close()

	tr := tar.NewReader(gz)
	for {
		header, err := tr.Next()
		if err == io.EOF {
			break
		}
		if err != nil {
			return fmt.Errorf("failed to read tar: %w", err)
		}

		name := header.Name
		if stripTopLevel {
			parts := strings.SplitN(name, "/", 2)
			if len(parts) < 2 || parts[1] == "" {
				continue
			}
			name = parts[1]
		}

		if name == "" || name == "." {
			continue
		}

		target := filepath.Join(destDir, filepath.FromSlash(name))

		if header.Typeflag == tar.TypeDir {
			if err := os.MkdirAll(target, 0755); err != nil {
				return fmt.Errorf("failed to create directory %s: %w", target, err)
			}
			continue
		}

		dir := filepath.Dir(target)
		if err := os.MkdirAll(dir, 0755); err != nil {
			return fmt.Errorf("failed to create directory %s: %w", dir, err)
		}

		outFile, err := os.Create(target)
		if err != nil {
			return fmt.Errorf("failed to create file %s: %w", target, err)
		}

		if _, err := io.Copy(outFile, tr); err != nil {
			outFile.Close()
			return fmt.Errorf("failed to write %s: %w", target, err)
		}
		outFile.Close()
	}

	return nil
}
