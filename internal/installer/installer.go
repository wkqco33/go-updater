package installer

import (
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"io"
	"log/slog"
	"net/http"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
)

// DownloadFile downloads a file from the URL to the destination.
func DownloadFile(url, dest string) error {
	slog.Debug("starting file download", "url", url, "dest", dest)
	resp, err := http.Get(url)
	if err != nil {
		return fmt.Errorf("failed to download file: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("bad status: %s", resp.Status)
	}

	out, err := os.Create(dest)
	if err != nil {
		return fmt.Errorf("failed to create file: %w", err)
	}
	defer out.Close()

	fmt.Printf("Downloading %s...\n", url)
	written, err := io.Copy(out, resp.Body)
	if err != nil {
		return fmt.Errorf("failed to write to file: %w", err)
	}
	slog.Debug("download complete", "bytes", written)
	return nil
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
		slog.Debug("executing powershell Expand-Archive")
		cmd := exec.Command("powershell", "-command", fmt.Sprintf("Expand-Archive -Path '%s' -DestinationPath '%s' -Force", archivePath, destDir))
		cmd.Stdout = os.Stdout
		cmd.Stderr = os.Stderr
		if err := cmd.Run(); err != nil {
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

// InstallGo handles the complete flow of downloading and installing.
func InstallGo(url, sha256Str string, targetDir string, version string) error {
	tmpDir := os.TempDir()
	archivePath := filepath.Join(tmpDir, filepath.Base(url))
	slog.Debug("install process started", "tmp_archive", archivePath, "target_dir", targetDir, "version", version)

	// 1. Download
	if err := DownloadFile(url, archivePath); err != nil {
		return err
	}
	defer os.Remove(archivePath)

	// 2. Verify
	fmt.Println("Verifying checksum...")
	if err := VerifyChecksum(archivePath, sha256Str); err != nil {
		return err
	}
	fmt.Println("Checksum OK.")

	// 3. Prepare target directory (~/.go/versions/go<version>)
	versionsDir := filepath.Join(targetDir, "versions")
	finalGoDir := filepath.Join(versionsDir, version)

	if err := os.MkdirAll(versionsDir, 0755); err != nil {
		return fmt.Errorf("failed to create versions directory: %w", err)
	}

	slog.Debug("checking for existing installation", "path", finalGoDir)
	if _, err := os.Stat(finalGoDir); err == nil {
		fmt.Printf("Removing existing Go installation at %s...\n", finalGoDir)
		if err := os.RemoveAll(finalGoDir); err != nil {
			return fmt.Errorf("failed to remove existing go installation: %w", err)
		}
	}

	// 4. Extract (archive contains a 'go' folder, so extract to temp, then rename)
	extractTmp := filepath.Join(versionsDir, fmt.Sprintf("tmp_%s", version))
	if err := os.RemoveAll(extractTmp); err != nil {
		return err
	}
	
	if err := ExtractArchive(archivePath, extractTmp); err != nil {
		os.RemoveAll(extractTmp)
		return err
	}
	
	// The archive extracts a directory named "go", rename it to the version name
	extractedGoDir := filepath.Join(extractTmp, "go")
	if err := os.Rename(extractedGoDir, finalGoDir); err != nil {
		os.RemoveAll(extractTmp)
		return fmt.Errorf("failed to rename extracted directory: %w", err)
	}
	os.RemoveAll(extractTmp)

	// 5. Update current symlink
	currentLink := filepath.Join(targetDir, "current")
	slog.Debug("updating current symlink", "link", currentLink, "target", finalGoDir)
	os.Remove(currentLink) // Remove if exists
	
	if err := os.Symlink(finalGoDir, currentLink); err != nil {
		return fmt.Errorf("failed to create symlink for current version: %w", err)
	}

	fmt.Printf("\n설치가 완료되었습니다! Go %s 버전이 %s 에 설치되었습니다.\n", version, finalGoDir)
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
