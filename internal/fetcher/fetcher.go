package fetcher

import (
	"encoding/json"
	"fmt"
	"log/slog"
	"net/http"
	"runtime"
	"strings"
	"time"
)

const ReleaseURL = "https://go.dev/dl/?mode=json"
const AllReleaseURL = "https://go.dev/dl/?mode=json&include=all"

var httpClient = &http.Client{
	Timeout: 30 * time.Second,
}

func FetchReleases(includeAll bool) ([]GoRelease, error) {
	url := ReleaseURL
	if includeAll {
		url = AllReleaseURL
	}
	slog.Debug("fetching releases from", "url", url)
	resp, err := httpClient.Get(url)
	if err != nil {
		return nil, fmt.Errorf("failed to fetch releases: %w", err)
	}
	defer resp.Body.Close()

	var releases []GoRelease
	if err := json.NewDecoder(resp.Body).Decode(&releases); err != nil {
		return nil, fmt.Errorf("failed to parse releases JSON: %w", err)
	}

	slog.Debug("successfully fetched and parsed release info", "count", len(releases))
	return releases, nil
}

func FindReleaseByVersion(version string) (*GoRelease, error) {
	if version == "" {
		releases, err := FetchReleases(false)
		if err != nil {
			return nil, err
		}
		for _, r := range releases {
			if r.Stable {
				slog.Debug("found latest stable release", "version", r.Version)
				return &r, nil
			}
		}
		return nil, fmt.Errorf("no stable release found")
	}

	// Format user input. e.g. "1.20" -> "go1.20", "1.20.5" -> "go1.20.5"
	targetVersion := version
	if !strings.HasPrefix(targetVersion, "go") {
		targetVersion = "go" + targetVersion
	}

	releases, err := FetchReleases(true)
	if err != nil {
		return nil, err
	}

	// If it's an exact match (e.g. go1.20.5)
	if strings.Count(targetVersion, ".") == 2 {
		for _, r := range releases {
			if r.Version == targetVersion {
				slog.Debug("found exact release", "version", r.Version)
				return &r, nil
			}
		}
	} else {
		// Prefix match (e.g. "go1.20" matches the first "go1.20.X" in the descending list)
		for _, r := range releases {
			if strings.HasPrefix(r.Version, targetVersion) {
				slog.Debug("found prefix matched release", "target", targetVersion, "matched", r.Version)
				return &r, nil
			}
		}
	}

	return nil, fmt.Errorf("release not found for version: %s", version)
}

func FindMatchingFile(release *GoRelease, os, arch string) (*GoFile, error) {
	if release == nil {
		return nil, fmt.Errorf("release is nil")
	}

	slog.Debug("finding matching file", "os", os, "arch", arch, "version", release.Version)

	for _, file := range release.Files {
		if file.OS == os && file.Arch == arch && file.Kind == "archive" {
			slog.Debug("found matching archive", "filename", file.Filename)
			return &file, nil
		}
	}
	return nil, fmt.Errorf("no matching archive found for os: %s, arch: %s", os, arch)
}

func GetDownloadURL(version string) (string, *GoFile, error) {
	slog.Debug("starting process to get download URL", "target_version", version, "runtime_os", runtime.GOOS, "runtime_arch", runtime.GOARCH)
	release, err := FindReleaseByVersion(version)
	if err != nil {
		return "", nil, err
	}

	file, err := FindMatchingFile(release, runtime.GOOS, runtime.GOARCH)
	if err != nil {
		return "", nil, err
	}

	downloadURL := fmt.Sprintf("https://dl.google.com/go/%s", file.Filename)
	slog.Debug("final download URL determined", "url", downloadURL)
	return downloadURL, file, nil
}
