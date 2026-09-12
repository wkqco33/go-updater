package installer

import (
	"archive/tar"
	"archive/zip"
	"compress/gzip"
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"io"
	"log/slog"
	"net"
	"net/http"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"time"
)

// downloadClient: 연결·헤더 타임아웃은 설정하되, 대용량 파일 수신 중 강제 종료를 막기 위해
// 전체 Timeout은 설정하지 않는다.
var downloadClient = &http.Client{
	Transport: &http.Transport{
		DialContext: (&net.Dialer{
			Timeout: 30 * time.Second,
		}).DialContext,
		ResponseHeaderTimeout: 30 * time.Second,
	},
}

// Options controls installer output. A nil Out or Err discards that stream so
// the library never writes to the process streams on its own.
type Options struct {
	// Out receives results such as the final summary and PATH guidance.
	Out io.Writer
	// Err receives progress and status messages.
	Err io.Writer
	// Quiet suppresses progress and status messages.
	Quiet bool
	// Progress renders download percentages (terminal output only).
	Progress bool
}

func (o Options) out() io.Writer {
	if o.Out == nil {
		return io.Discard
	}
	return o.Out
}

func (o Options) err() io.Writer {
	if o.Err == nil {
		return io.Discard
	}
	return o.Err
}

func (o Options) status(format string, args ...any) {
	if o.Quiet {
		return
	}
	fmt.Fprintf(o.err(), format, args...)
}

// progressWriter renders download progress to a caller-provided writer.
type progressWriter struct {
	out     io.Writer
	total   int64 // 전체 크기 (0이면 알 수 없음)
	written int64
	lastPct int
}

func (p *progressWriter) Write(b []byte) (int, error) {
	n := len(b)
	p.written += int64(n)
	if p.total > 0 {
		pct := int(p.written * 100 / p.total)
		if pct >= p.lastPct+5 {
			p.lastPct = pct
			fmt.Fprintf(p.out, "\r  %d%% (%d / %d MB)", pct, p.written/1024/1024, p.total/1024/1024)
		}
	} else {
		fmt.Fprintf(p.out, "\r  %.1f MB", float64(p.written)/1024/1024)
	}
	return n, nil
}

// DownloadFile downloads url into dest and returns the SHA256 of the stream as
// it is written (no second pass over the file). progress receives percentage
// updates when non-nil.
func DownloadFile(url, dest string, progress io.Writer) (string, error) {
	slog.Debug("starting file download", "url", url, "dest", dest)
	resp, err := downloadClient.Get(url)
	if err != nil {
		return "", fmt.Errorf("failed to download file: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return "", fmt.Errorf("bad status: %s", resp.Status)
	}

	out, err := os.Create(dest)
	if err != nil {
		return "", fmt.Errorf("failed to create file: %w", err)
	}
	defer out.Close()

	sink := io.Writer(out)
	if progress != nil {
		sink = io.MultiWriter(out, &progressWriter{out: progress, total: resp.ContentLength})
	}

	h := sha256.New()
	written, err := io.Copy(sink, io.TeeReader(resp.Body, h))
	if progress != nil {
		fmt.Fprintln(progress)
	}
	if err != nil {
		return "", fmt.Errorf("failed to write to file: %w", err)
	}
	slog.Debug("download complete", "bytes", written)
	return hex.EncodeToString(h.Sum(nil)), nil
}

// ExtractArchive extracts the downloaded archive, reporting progress to
// status when it is non-nil.
func ExtractArchive(archivePath, destDir string, status io.Writer) error {
	slog.Debug("extracting archive", "archive", archivePath, "dest", destDir, "os", runtime.GOOS)
	if status != nil {
		fmt.Fprintf(status, "Extracting %s to %s...\n", archivePath, destDir)
	}

	if err := os.MkdirAll(destDir, 0755); err != nil {
		return fmt.Errorf("failed to create destination directory: %w", err)
	}

	if runtime.GOOS == "windows" {
		slog.Debug("extracting zip with archive/zip")
		if err := extractZip(archivePath, destDir); err != nil {
			return fmt.Errorf("failed to extract zip: %w", err)
		}
	} else {
		slog.Debug("extracting tar.gz with archive/tar")
		if err := extractTarGz(archivePath, destDir); err != nil {
			return fmt.Errorf("failed to extract tar.gz: %w", err)
		}
	}
	slog.Debug("extraction complete")
	return nil
}

func archivePathWithin(destDir, name string) (string, error) {
	cleanDest := filepath.Clean(destDir)
	path := filepath.Join(cleanDest, filepath.FromSlash(name))
	cleanPath := filepath.Clean(path)
	if cleanPath != cleanDest && !strings.HasPrefix(cleanPath, cleanDest+string(os.PathSeparator)) {
		return "", fmt.Errorf("illegal file path: %s", name)
	}
	return cleanPath, nil
}

// extractTarGz extracts a tar.gz archive without invoking an external tar
// binary. It rejects traversal and non-regular entries so an archive cannot
// create a symlink that escapes the destination directory.
func extractTarGz(archivePath, destDir string) error {
	file, err := os.Open(archivePath)
	if err != nil {
		return fmt.Errorf("failed to open archive: %w", err)
	}
	defer file.Close()
	gz, err := gzip.NewReader(file)
	if err != nil {
		return fmt.Errorf("failed to read gzip archive: %w", err)
	}
	defer gz.Close()

	tr := tar.NewReader(gz)
	for {
		header, err := tr.Next()
		if err == io.EOF {
			return nil
		}
		if err != nil {
			return fmt.Errorf("failed to read tar archive: %w", err)
		}
		path, err := archivePathWithin(destDir, header.Name)
		if err != nil {
			return err
		}
		switch header.Typeflag {
		case tar.TypeDir:
			if err := os.MkdirAll(path, 0755); err != nil {
				return fmt.Errorf("failed to create directory %s: %w", path, err)
			}
		case tar.TypeReg, tar.TypeRegA:
			if err := os.MkdirAll(filepath.Dir(path), 0755); err != nil {
				return fmt.Errorf("failed to create parent directory for %s: %w", path, err)
			}
			out, err := os.OpenFile(path, os.O_WRONLY|os.O_CREATE|os.O_TRUNC, os.FileMode(header.Mode)&0777)
			if err != nil {
				return fmt.Errorf("failed to create file %s: %w", path, err)
			}
			_, copyErr := io.Copy(out, tr)
			closeErr := out.Close()
			if copyErr != nil {
				return fmt.Errorf("failed to write file %s: %w", path, copyErr)
			}
			if closeErr != nil {
				return fmt.Errorf("failed to close file %s: %w", path, closeErr)
			}
		default:
			return fmt.Errorf("unsupported tar entry type for %s", header.Name)
		}
	}
}

// extractZip extracts a ZIP archive to the destination directory using Go's
// standard archive/zip package, avoiding dependency on PowerShell.
func extractZip(archivePath, destDir string) error {
	r, err := zip.OpenReader(archivePath)
	if err != nil {
		return fmt.Errorf("failed to open zip archive: %w", err)
	}
	defer r.Close()

	for _, f := range r.File {
		fpath, err := archivePathWithin(destDir, f.Name)
		if err != nil {
			return err
		}

		if f.FileInfo().IsDir() {
			if err := os.MkdirAll(fpath, 0755); err != nil {
				return fmt.Errorf("failed to create directory %s: %w", fpath, err)
			}
			continue
		}

		if err := os.MkdirAll(filepath.Dir(fpath), 0755); err != nil {
			return fmt.Errorf("failed to create parent directory for %s: %w", fpath, err)
		}

		rc, err := f.Open()
		if err != nil {
			return fmt.Errorf("failed to open %s in archive: %w", f.Name, err)
		}

		outFile, err := os.OpenFile(fpath, os.O_WRONLY|os.O_CREATE|os.O_TRUNC, f.Mode())
		if err != nil {
			rc.Close()
			return fmt.Errorf("failed to create file %s: %w", fpath, err)
		}

		_, err = io.Copy(outFile, rc)
		rc.Close()
		outFile.Close()
		if err != nil {
			return fmt.Errorf("failed to write file %s: %w", fpath, err)
		}
	}

	return nil
}

func removeCurrentLink(path string) {
	// os.RemoveAll은 symlink를 따라가지 않고 링크 자체만 제거한다.
	// Windows의 기존 디렉터리 심볼릭 링크는 os.Remove로 안 지워질 수 있어
	// RemoveAll을 사용한다.
	if err := os.RemoveAll(path); err != nil {
		slog.Debug("failed to remove existing current link", "path", path, "error", err)
	}
}

// UpdateCurrentSymlink updates the 'current' symlink in targetDir to point to goDir.
// On Windows, falls back to directory junction (mklink /J) if symlink creation
// fails due to insufficient privileges.
func UpdateCurrentSymlink(targetDir, goDir string) error {
	currentLink := filepath.Join(targetDir, "current")
	tmpLink := filepath.Join(targetDir, ".current.tmp")
	slog.Debug("updating current symlink", "link", currentLink, "target", goDir)
	_ = os.Remove(tmpLink)

	if err := os.Symlink(goDir, tmpLink); err != nil {
		if runtime.GOOS == "windows" {
			slog.Debug("symlink failed, trying mklink /J", "error", err)
			// mklink /J는 대상 경로가 이미 존재하면 실패하므로 기존 링크를 먼저 제거한다.
			removeCurrentLink(currentLink)
			if mklinkErr := createWindowsJunction(currentLink, goDir); mklinkErr != nil {
				return fmt.Errorf("failed to create symlink/junction for current version: %w\nWindows에서 심볼릭 링크를 생성하려면 개발자 모드를 활성화하거나 관리자 권한으로 실행하세요.", err)
			}
			return nil
		}
		return fmt.Errorf("failed to create symlink for current version: %w", err)
	}
	if runtime.GOOS == "windows" {
		// Windows는 os.Rename으로 기존 디렉터리 심볼릭 링크를 원자적으로 덮어쓸 수
		// 없어 "Access is denied"가 발생한다. 따라서 기존 링크를 제거 후 교체한다.
		removeCurrentLink(currentLink)
	}
	if err := os.Rename(tmpLink, currentLink); err != nil {
		_ = os.Remove(tmpLink)
		return fmt.Errorf("failed to atomically replace current link: %w", err)
	}
	return nil
}

// InstallGo handles the complete flow of downloading and installing. Output
// is fully caller-controlled through opts so the CLI can keep stdout reserved
// for results.
func InstallGo(url, sha256Str string, targetDir string, version string, opts Options) error {
	tmpDir, err := os.MkdirTemp("", "go-updater-install-*")
	if err != nil {
		return fmt.Errorf("failed to create install temp directory: %w", err)
	}
	defer os.RemoveAll(tmpDir)
	archivePath := filepath.Join(tmpDir, filepath.Base(url))
	slog.Debug("install process started", "tmp_archive", archivePath, "target_dir", targetDir, "version", version)

	// 1. Download + SHA256 동시 계산 (단일 패스)
	opts.status("Downloading %s...\n", url)
	var progress io.Writer
	if opts.Progress && !opts.Quiet {
		progress = opts.err()
	}
	actualSha256, err := DownloadFile(url, archivePath, progress)
	if err != nil {
		return err
	}

	// 2. Verify
	slog.Debug("checksum calculated", "actual", actualSha256, "expected", sha256Str)
	if actualSha256 != sha256Str {
		return fmt.Errorf("checksum mismatch: expected %s, got %s", sha256Str, actualSha256)
	}
	opts.status("Checksum OK.\n")

	// 3. Prepare target directory (~/.go/versions/go<version>)
	versionsDir := filepath.Join(targetDir, "versions")
	finalGoDir := filepath.Join(versionsDir, version)

	if err := os.MkdirAll(versionsDir, 0755); err != nil {
		return fmt.Errorf("failed to create versions directory: %w", err)
	}

	slog.Debug("checking for existing installation", "path", finalGoDir)

	// 4. Extract into staging. The existing installation is preserved until
	// the staged archive has been completely prepared.

	extractTmp := filepath.Join(versionsDir, fmt.Sprintf("tmp_%s", version))
	if err := os.RemoveAll(extractTmp); err != nil {
		return err
	}

	var extractStatus io.Writer
	if !opts.Quiet {
		extractStatus = opts.err()
	}
	if err := ExtractArchive(archivePath, extractTmp, extractStatus); err != nil {
		os.RemoveAll(extractTmp)
		return err
	}

	// The archive extracts a directory named "go", rename it to the version name.
	extractedGoDir := filepath.Join(extractTmp, "go")
	if _, err := os.Stat(extractedGoDir); err != nil {
		os.RemoveAll(extractTmp)
		return fmt.Errorf("archive does not contain a go directory: %w", err)
	}
	backupDir := finalGoDir + ".old"
	_ = os.RemoveAll(backupDir)
	if _, err := os.Stat(finalGoDir); err == nil {
		if err := os.Rename(finalGoDir, backupDir); err != nil {
			os.RemoveAll(extractTmp)
			return fmt.Errorf("failed to stage existing Go installation: %w", err)
		}
	}
	if err := os.Rename(extractedGoDir, finalGoDir); err != nil {
		_ = os.Rename(backupDir, finalGoDir)
		os.RemoveAll(extractTmp)
		return fmt.Errorf("failed to rename extracted directory: %w", err)
	}
	os.RemoveAll(extractTmp)
	defer os.RemoveAll(backupDir)

	// 5. Update current symlink
	if err := UpdateCurrentSymlink(targetDir, finalGoDir); err != nil {
		_ = os.RemoveAll(finalGoDir)
		_ = os.Rename(backupDir, finalGoDir)
		return err
	}

	fmt.Fprintf(opts.out(), "\n설치가 완료되었습니다! Go %s 버전이 %s 에 설치되었습니다.\n", version, finalGoDir)
	currentLink := filepath.Join(targetDir, "current")
	binDir := filepath.Join(currentLink, "bin")

	fmt.Fprintln(opts.out(), "\n[환경 변수(PATH) 설정 안내]")
	if runtime.GOOS == "windows" {
		fmt.Fprintf(opts.out(), "Windows 시스템 환경 변수 편집에서 다음 경로를 PATH에 추가하세요:\n  %s\n", binDir)
	} else {
		shell := os.Getenv("SHELL")
		configFiles := []string{".bashrc", ".profile"}
		if filepath.Base(shell) == "zsh" {
			configFiles = []string{".zshrc"}
		}

		fmt.Fprintln(opts.out(), "터미널에서 아래 명령어를 실행하여 PATH를 설정할 수 있습니다:")
		for _, file := range configFiles {
			fmt.Fprintf(opts.out(), "  echo 'export PATH=$PATH:%s' >> ~/%s\n", binDir, file)
		}
		fmt.Fprintln(opts.out(), "\n설정 후에는 터미널을 재시작하거나 'source' 명령어로 설정을 적용하세요.")
		fmt.Fprintf(opts.out(), "  source ~/%s\n", configFiles[0])
	}

	return nil
}
