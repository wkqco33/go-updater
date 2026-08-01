//go:build darwin

package systemgo

import (
	"os"
	"os/exec"
	"path/filepath"
	"slices"
	"strings"
)

// knownPkgIDs are the pkgutil package-ids used by the official go.dev macOS
// installer across versions. Matched exactly to avoid false positives like
// "org.golang.gomobile".
var knownPkgIDs = []string{"org.golang.go", "com.googlecode.go"}

// runCommand is overridden in tests to avoid invoking real system binaries.
var runCommand = func(name string, args ...string) (string, error) {
	out, err := exec.Command(name, args...).Output()
	return string(out), err
}

// pathsDGoFile and defaultGoRoot are overridden in tests so Detect can be
// exercised against a t.TempDir() instead of the real /etc/paths.d/go and
// /usr/local/go on the machine running the tests.
var (
	pathsDGoFile     = PathsDGo
	defaultGoRoot    = "/usr/local/go"
	homebrewPrefixes = []string{"/opt/homebrew", "/usr/local"}
)

// Detect looks for traces of a go.dev .pkg installation (installer receipt,
// GOROOT directory, /etc/paths.d entry) and, separately, a Homebrew-managed
// Go install. It returns an empty slice if nothing at all is found.
func Detect() ([]Present, error) {
	var receiptID, version, volume, location string
	if out, err := runCommand("pkgutil", "--pkgs"); err == nil {
		for line := range strings.SplitSeq(out, "\n") {
			line = strings.TrimSpace(line)
			if slices.Contains(knownPkgIDs, line) {
				receiptID = line
			}
		}
	}

	receiptExists := receiptID != ""
	if receiptExists {
		if out, err := runCommand("pkgutil", "--pkg-info", receiptID); err == nil {
			info := parsePkgInfo(out)
			version = info["version"]
			volume = info["volume"]
			location = info["location"]
		}
	}

	var pathsDExists bool
	var pathsDTarget string
	if data, err := os.ReadFile(pathsDGoFile); err == nil {
		pathsDExists = true
		pathsDTarget = goRootFromPathsD(string(data))
	}

	goRoot := findExistingGoRoot(volume, location, pathsDTarget)
	goRootExists := goRoot != ""
	if !goRootExists {
		goRoot = defaultGoRoot
	}

	if !receiptExists && !pathsDExists && !goRootExists {
		if hb, ok := detectHomebrew(); ok {
			return []Present{{Artifact: hb, Exists: true}}, nil
		}
		return nil, nil
	}

	var results []Present

	results = append(results, Present{
		Artifact: Artifact{Kind: KindDir, Path: goRoot, Detail: version, Managed: true},
		Exists:   goRootExists,
	})

	pathsDDetail := ""
	if pathsDExists {
		pathsDDetail = "→ " + pathsDTarget
	}
	results = append(results, Present{
		Artifact: Artifact{Kind: KindFile, Path: pathsDGoFile, Detail: pathsDDetail, Managed: true},
		Exists:   pathsDExists,
	})

	if receiptExists {
		results = append(results, Present{
			Artifact: Artifact{Kind: KindReceipt, Path: receiptID, Detail: version, Managed: true},
			Exists:   true,
		})
	} else {
		results = append(results, Present{
			Artifact: Artifact{Kind: KindReceipt, Path: knownPkgIDs[0]},
			Exists:   false,
		})
	}

	if hb, ok := detectHomebrew(); ok {
		results = append(results, Present{Artifact: hb, Exists: true})
	}

	return results, nil
}

// findExistingGoRoot returns the first GOROOT candidate that actually exists
// on disk among the receipt-derived location, the /etc/paths.d target, and
// the go.dev default. Returns "" if none exist.
func findExistingGoRoot(volume, location, pathsDTarget string) string {
	var candidates []string
	if volume != "" && location != "" {
		candidates = append(candidates, filepath.Join(volume, location, "usr/local/go"))
	}
	if pathsDTarget != "" {
		candidates = append(candidates, pathsDTarget)
	}
	candidates = append(candidates, defaultGoRoot)

	seen := map[string]bool{}
	for _, c := range candidates {
		c = filepath.Clean(c)
		if seen[c] {
			continue
		}
		seen[c] = true
		if _, err := os.Lstat(c); err == nil {
			return c
		}
	}
	return ""
}

// detectHomebrew reports a Homebrew-managed Go install, if any. It is never
// removed by gu; only surfaced so the user knows to run `brew uninstall go`.
func detectHomebrew() (Artifact, bool) {
	for _, prefix := range homebrewPrefixes {
		p := filepath.Join(prefix, "opt", "go")
		if _, err := os.Lstat(p); err == nil {
			return Artifact{Kind: KindHomebrew, Path: p}, true
		}
	}
	return Artifact{}, false
}

// parsePkgInfo parses the "key: value" lines produced by
// `pkgutil --pkg-info <id>` into a map.
func parsePkgInfo(out string) map[string]string {
	result := make(map[string]string)
	for line := range strings.SplitSeq(out, "\n") {
		key, value, ok := strings.Cut(line, ":")
		if !ok {
			continue
		}
		key = strings.TrimSpace(key)
		if key == "" {
			continue
		}
		result[key] = strings.TrimSpace(value)
	}
	return result
}

// goRootFromPathsD derives the GOROOT directory from the contents of
// /etc/paths.d/go, which contains a single line like "/usr/local/go/bin".
func goRootFromPathsD(content string) string {
	line := strings.TrimSpace(content)
	if line == "" {
		return ""
	}
	// Only the first line matters; paths.d entries are one path per line.
	if idx := strings.IndexAny(line, "\r\n"); idx >= 0 {
		line = line[:idx]
	}
	line = strings.TrimSuffix(line, "/bin")
	if line == "" {
		return ""
	}
	return filepath.Clean(line)
}
