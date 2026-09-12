package fetcher

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"log/slog"
	"net/http"
	"runtime"
	"strings"
	"time"
)

const ReleaseURL = "https://go.dev/dl/?mode=json"
const AllReleaseURL = "https://go.dev/dl/?mode=json&include=all"

// Client retrieves and resolves Go release metadata. Keeping the HTTP client
// and endpoint on a value makes the network boundary replaceable in tests.
type Client struct {
	HTTPClient *http.Client
	BaseURL    string
	AllURL     string
}

func NewClient(httpClient *http.Client) *Client {
	if httpClient == nil {
		httpClient = &http.Client{Timeout: 30 * time.Second}
	}
	return &Client{HTTPClient: httpClient, BaseURL: ReleaseURL, AllURL: AllReleaseURL}
}

var defaultClient = NewClient(nil)

func (c *Client) FetchReleases(ctx context.Context, includeAll bool) ([]GoRelease, error) {
	url := c.BaseURL
	if includeAll {
		url = c.AllURL
	}
	slog.Debug("fetching releases from", "url", url)
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to create release request: %w", err)
	}
	resp, err := c.HTTPClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("failed to fetch releases: %w", err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		_, _ = io.Copy(io.Discard, resp.Body)
		return nil, fmt.Errorf("failed to fetch releases: unexpected HTTP status %s", resp.Status)
	}

	var releases []GoRelease
	if err := json.NewDecoder(resp.Body).Decode(&releases); err != nil {
		return nil, fmt.Errorf("failed to parse releases JSON: %w", err)
	}
	slog.Debug("successfully fetched and parsed release info", "count", len(releases))
	return releases, nil
}

func (c *Client) FindReleaseByVersion(ctx context.Context, version string) (*GoRelease, error) {
	if version == "" {
		releases, err := c.FetchReleases(ctx, false)
		if err != nil {
			return nil, err
		}
		for i := range releases {
			if releases[i].Stable {
				return &releases[i], nil
			}
		}
		return nil, fmt.Errorf("no stable release found")
	}

	targetVersion := version
	if !strings.HasPrefix(targetVersion, "go") {
		targetVersion = "go" + targetVersion
	}
	releases, err := c.FetchReleases(ctx, true)
	if err != nil {
		return nil, err
	}

	// go.dev returns releases newest first, so the first match is the highest
	// patch release for a minor request like "1.20".
	for i := range releases {
		if matchesReleaseVersion(releases[i].Version, targetVersion, releases[i].Stable) {
			return &releases[i], nil
		}
	}
	return nil, fmt.Errorf("release not found for version: %s", version)
}

// matchesReleaseVersion reports whether releaseVersion satisfies the requested
// version. Exact requests accept any release the user named explicitly (for
// example "1.21rc1"); a partial request like "1.20" only matches a stable
// patch release of that minor version and never a different minor such as
// 1.200 or a pre-release such as 1.20rc1.
func matchesReleaseVersion(releaseVersion, target string, stable bool) bool {
	if releaseVersion == target {
		return true
	}
	if !strings.HasPrefix(releaseVersion, target) {
		return false
	}
	return stable && strings.HasPrefix(releaseVersion[len(target):], ".")
}

func FindMatchingFile(release *GoRelease, osName, arch string) (*GoFile, error) {
	if release == nil {
		return nil, fmt.Errorf("release is nil")
	}
	for i := range release.Files {
		if release.Files[i].OS == osName && release.Files[i].Arch == arch && release.Files[i].Kind == "archive" {
			return &release.Files[i], nil
		}
	}
	return nil, fmt.Errorf("no matching archive found for os: %s, arch: %s", osName, arch)
}

func (c *Client) GetDownloadURL(ctx context.Context, version, osName, arch string) (string, *GoFile, error) {
	release, err := c.FindReleaseByVersion(ctx, version)
	if err != nil {
		return "", nil, err
	}
	file, err := FindMatchingFile(release, osName, arch)
	if err != nil {
		return "", nil, err
	}
	return fmt.Sprintf("https://dl.google.com/go/%s", file.Filename), file, nil
}

// Compatibility wrappers. New code should use Client methods so tests can
// provide an httptest server and a context.
func FetchReleases(includeAll bool) ([]GoRelease, error) {
	return defaultClient.FetchReleases(context.Background(), includeAll)
}

func FindReleaseByVersion(version string) (*GoRelease, error) {
	return defaultClient.FindReleaseByVersion(context.Background(), version)
}

func GetDownloadURL(version string) (string, *GoFile, error) {
	return defaultClient.GetDownloadURL(context.Background(), version, runtime.GOOS, runtime.GOARCH)
}
