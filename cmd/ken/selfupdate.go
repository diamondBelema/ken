package main

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"runtime"
	"strconv"
	"strings"
	"time"

	"github.com/spf13/cobra"
)

const releasesAPI = "https://api.github.com/repos/diamondBelema/ken/releases/latest"

type githubRelease struct {
	TagName string `json:"tag_name"`
	Assets  []struct {
		Name               string `json:"name"`
		BrowserDownloadURL string `json:"browser_download_url"`
	} `json:"assets"`
}

func fetchLatestRelease() (githubRelease, error) {
	client := &http.Client{Timeout: 10 * time.Second}
	resp, err := client.Get(releasesAPI)
	if err != nil {
		return githubRelease{}, fmt.Errorf("failed to check for updates: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return githubRelease{}, fmt.Errorf("GitHub API returned status %d", resp.StatusCode)
	}

	var release githubRelease
	if err := json.NewDecoder(resp.Body).Decode(&release); err != nil {
		return githubRelease{}, fmt.Errorf("failed to parse release info: %w", err)
	}

	return release, nil
}

func downloadBinary(url string) (io.ReadCloser, error) {
	client := &http.Client{Timeout: 120 * time.Second}
	resp, err := client.Get(url)
	if err != nil {
		return nil, fmt.Errorf("download failed: %w", err)
	}
	if resp.StatusCode != http.StatusOK {
		resp.Body.Close()
		return nil, fmt.Errorf("download returned status %d", resp.StatusCode)
	}
	return resp.Body, nil
}

func expectedAssetName() string {
	osName := runtime.GOOS
	arch := runtime.GOARCH
	if osName == "windows" {
		return fmt.Sprintf("ken-%s-%s.exe", osName, arch)
	}
	return fmt.Sprintf("ken-%s-%s", osName, arch)
}

var selfUpdateCmd = &cobra.Command{
	Use:   "self-update",
	Short: "Update ken to the latest version",
	Long: `Check GitHub Releases for the latest version of ken and replace
the current binary.

Examples:
  ken self-update`,
	RunE: func(cmd *cobra.Command, args []string) error {
		fmt.Printf("Current: %s\n", version)

		fmt.Println("Checking for updates...")
		release, err := fetchLatestRelease()
		if err != nil {
			return err
		}

		latestVersion := strings.TrimPrefix(release.TagName, "v")
		currentClean := strings.TrimPrefix(version, "v")

		fmt.Printf("Latest : %s\n", latestVersion)

		if !versionLess(currentClean, latestVersion) {
			fmt.Println("\nAlready up to date!")
			return nil
		}

		assetName := expectedAssetName()
		var downloadURL string
		for _, asset := range release.Assets {
			if asset.Name == assetName {
				downloadURL = asset.BrowserDownloadURL
				break
			}
		}

		if downloadURL == "" {
			return fmt.Errorf("no binary found for %s/%s in release %s\nAvailable: %s",
				runtime.GOOS, runtime.GOARCH, release.TagName, listAssetNames(release))
		}

		fmt.Printf("\nDownloading %s...\n", assetName)

		reader, err := downloadBinary(downloadURL)
		if err != nil {
			return err
		}
		defer reader.Close()

		// Get current executable path
		exePath, err := os.Executable()
		if err != nil {
			return fmt.Errorf("cannot find current executable: %w", err)
		}

		// Write to temp file first, then rename (atomic on same filesystem)
		tmpPath := exePath + ".tmp"
		out, err := os.Create(tmpPath)
		if err != nil {
			return fmt.Errorf("cannot create temp file: %w", err)
		}

		if _, err := io.Copy(out, reader); err != nil {
			out.Close()
			os.Remove(tmpPath)
			return fmt.Errorf("download interrupted: %w", err)
		}
		out.Close()

		// Make executable
		if err := os.Chmod(tmpPath, 0755); err != nil {
			os.Remove(tmpPath)
			return fmt.Errorf("cannot make executable: %w", err)
		}

		// Replace
		if err := os.Rename(tmpPath, exePath); err != nil {
			os.Remove(tmpPath)
			return fmt.Errorf("cannot replace binary: %w (you may need to run with sudo)", err)
		}

		fmt.Printf("\nUpdated to %s\n", latestVersion)
		return nil
	},
}

func listAssetNames(release githubRelease) string {
	var names []string
	for _, a := range release.Assets {
		names = append(names, a.Name)
	}
	if len(names) == 0 {
		return "(no assets)"
	}
	return strings.Join(names, ", ")
}

// parseVersion splits "1.2.3" into [1, 2, 3]. Returns nil if invalid.
func parseVersion(v string) []int {
	v = strings.TrimPrefix(v, "v")
	parts := strings.Split(v, ".")
	var nums []int
	for _, p := range parts {
		n, err := strconv.Atoi(p)
		if err != nil {
			return nil
		}
		nums = append(nums, n)
	}
	return nums
}

// versionLess returns true if a < b (semver).
func versionLess(a, b string) bool {
	av := parseVersion(a)
	bv := parseVersion(b)
	if av == nil || bv == nil {
		return a != b
	}
	for i := 0; i < len(av) && i < len(bv); i++ {
		if av[i] < bv[i] {
			return true
		}
		if av[i] > bv[i] {
			return false
		}
	}
	return len(av) < len(bv)
}

func init() {
	rootCmd.AddCommand(selfUpdateCmd)
}
