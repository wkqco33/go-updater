package installer

import (
	"archive/zip"
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"io"
	"log/slog"
	"net"
	"net/http"
	"os"
	"os/exec"
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

// progressWriter는 io.Writer를 감싸 다운로드 진행률을 출력한다.
type progressWriter struct {
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
			fmt.Printf("\r  %d%% (%d / %d MB)", pct, p.written/1024/1024, p.total/1024/1024)
		}
	} else {
		fmt.Printf("\r  %.1f MB", float64(p.written)/1024/1024)
	}
	return n, nil
}

// DownloadFile downloads a file from the URL to the destination.
// SHA256 해시를 다운로드와 동시에 계산해 반환한다 (파일 재읽기 없음).
func DownloadFile(url, dest string) (string, error) {
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

	fmt.Printf("Downloading %s...\n", url)
	pw := &progressWriter{total: resp.ContentLength}
	h := sha256.New()
	// TeeReader: body → hash 계산, MultiWriter: file 저장 + 진행률 동시 출력
	written, err := io.Copy(io.MultiWriter(out, pw), io.TeeReader(resp.Body, h))
	fmt.Println() // 진행률 줄 마무리
	if err != nil {
		return "", fmt.Errorf("failed to write to file: %w", err)
	}
	slog.Debug("download complete", "bytes", written)
	return hex.EncodeToString(h.Sum(nil)), nil
}

// VerifyChecksum verifies the SHA256 checksum of a file.
func VerifyChecksum(filePath, expectedSha256 string) error {
	slog.Debug("verifying checksum", "file", filePath, "expected", expectedSha256)
	f, err := os.Open(filePath)
	if err != nil {
		return err
	}
	defer f.Close()

	h := sha256.New()
	if _, err := io.Copy(h, f); err != nil {
		return err
	}

	actualSha256 := hex.EncodeToString(h.Sum(nil))
	slog.Debug("checksum calculated", "actual", actualSha256)
	if actualSha256 != expectedSha256 {
		return fmt.Errorf("checksum mismatch: expected %s, got %s", expectedSha256, actualSha256)
	}
	return nil
}

// ExtractArchive extracts the downloaded archive.
func ExtractArchive(archivePath, destDir string) error {
	slog.Debug("extracting archive", "archive", archivePath, "dest", destDir, "os", runtime.GOOS)
	fmt.Printf("Extracting %s to %s...\n", archivePath, destDir)

	if err := os.MkdirAll(destDir, 0755); err != nil {
		return fmt.Errorf("failed to create destination directory: %w", err)
	}

	if runtime.GOOS == "windows" {
		slog.Debug("extracting zip with archive/zip")
		if err := extractZip(archivePath, destDir); err != nil {
			return fmt.Errorf("failed to extract zip: %w", err)
		}
	} else {
		slog.Debug("executing tar -xzf")
		cmd := exec.Command("tar", "-xzf", archivePath, "-C", destDir)
		cmd.Stdout = os.Stdout
		cmd.Stderr = os.Stderr
		if err := cmd.Run(); err != nil {
			return fmt.Errorf("failed to extract tar.gz: %w", err)
		}
	}
	slog.Debug("extraction complete")
	return nil
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
		fpath := filepath.Join(destDir, f.Name)

		if !strings.HasPrefix(filepath.Clean(fpath), filepath.Clean(destDir)+string(os.PathSeparator)) {
			return fmt.Errorf("illegal file path: %s", fpath)
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
			cmd := exec.Command("cmd", "/c", "mklink", "/J", currentLink, goDir)
			if out, mklinkErr := cmd.CombinedOutput(); mklinkErr != nil {
				return fmt.Errorf("failed to create symlink/junction for current version: %w (%s)\nWindows에서 심볼릭 링크를 생성하려면 개발자 모드를 활성화하거나 관리자 권한으로 실행하세요.", err, strings.TrimSpace(string(out)))
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

// InstallGo handles the complete flow of downloading and installing.
func InstallGo(url, sha256Str string, targetDir string, version string) error {
	tmpDir, err := os.MkdirTemp("", "go-updater-install-*")
	if err != nil {
		return fmt.Errorf("failed to create install temp directory: %w", err)
	}
	defer os.RemoveAll(tmpDir)
	archivePath := filepath.Join(tmpDir, filepath.Base(url))
	slog.Debug("install process started", "tmp_archive", archivePath, "target_dir", targetDir, "version", version)

	// 1. Download + SHA256 동시 계산 (단일 패스)
	fmt.Println("Verifying checksum...")
	actualSha256, err := DownloadFile(url, archivePath)
	if err != nil {
		return err
	}

	// 2. Verify
	slog.Debug("checksum calculated", "actual", actualSha256, "expected", sha256Str)
	if actualSha256 != sha256Str {
		return fmt.Errorf("checksum mismatch: expected %s, got %s", sha256Str, actualSha256)
	}
	fmt.Println("Checksum OK.")

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

	if err := ExtractArchive(archivePath, extractTmp); err != nil {
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

	fmt.Printf("\n설치가 완료되었습니다! Go %s 버전이 %s 에 설치되었습니다.\n", version, finalGoDir)
	currentLink := filepath.Join(targetDir, "current")
	binDir := filepath.Join(currentLink, "bin")

	fmt.Println("\n[환경 변수(PATH) 설정 안내]")
	if runtime.GOOS == "windows" {
		fmt.Printf("Windows 시스템 환경 변수 편집에서 다음 경로를 PATH에 추가하세요:\n  %s\n", binDir)
	} else {
		shell := os.Getenv("SHELL")
		configFiles := []string{".bashrc", ".profile"}
		if filepath.Base(shell) == "zsh" {
			configFiles = []string{".zshrc"}
		}

		fmt.Println("터미널에서 아래 명령어를 실행하여 PATH를 설정할 수 있습니다:")
		for _, file := range configFiles {
			fmt.Printf("  echo 'export PATH=$PATH:%s' >> ~/%s\n", binDir, file)
		}
		fmt.Println("\n설정 후에는 터미널을 재시작하거나 'source' 명령어로 설정을 적용하세요.")
		fmt.Printf("  source ~/%s\n", configFiles[0])
	}

	return nil
}
