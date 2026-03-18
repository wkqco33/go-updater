package fetcher

// GoFile represents a downloadable file for a Go release.
type GoFile struct {
	Filename string `json:"filename"`
	OS       string `json:"os"`
	Arch     string `json:"arch"`
	Version  string `json:"version"`
	Sha256   string `json:"sha256"`
	Size     int64  `json:"size"`
	Kind     string `json:"kind"` // "archive", "installer", "source"
}

// GoRelease represents a single release from go.dev/dl/?mode=json.
type GoRelease struct {
	Version string   `json:"version"`
	Stable  bool     `json:"stable"`
	Files   []GoFile `json:"files"`
}
