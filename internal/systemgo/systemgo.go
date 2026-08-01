// Package systemgo detects and removes Go installations that were placed
// outside the gu-managed ~/.go directory, such as the official go.dev
// installer packages.
package systemgo

// ArtifactKind identifies the kind of leftover a system Go install left behind.
type ArtifactKind int

const (
	// KindDir is a directory to remove, e.g. the GOROOT at /usr/local/go.
	KindDir ArtifactKind = iota
	// KindFile is a single file to remove, e.g. /etc/paths.d/go.
	KindFile
	// KindReceipt is a macOS installer receipt tracked by pkgutil.
	KindReceipt
	// KindHomebrew is a Homebrew-managed Go install; detected but never removed.
	KindHomebrew
	// KindWinget is a winget-managed Go install; removed via winget uninstall.
	KindWinget
)

// Artifact describes one leftover trace of a system Go installation.
type Artifact struct {
	Kind ArtifactKind
	// Path is the filesystem path (KindDir/KindFile), or the pkgutil
	// package-id (KindReceipt), or the Homebrew prefix (KindHomebrew).
	Path string
	// Detail is extra context shown to the user, e.g. a version string or
	// the resolved target of /etc/paths.d/go.
	Detail string
	// Managed is false when the artifact is only detected/reported, never
	// removed by gu (currently: Homebrew installs).
	Managed bool
}

// Present pairs an Artifact with whether it currently exists on disk. Detect
// returns entries for candidates that don't exist too, so callers can show a
// full checklist (e.g. "/usr/local/go (없음)") once something related was found.
type Present struct {
	Artifact Artifact
	Exists   bool
}
