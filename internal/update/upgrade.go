package update

import (
	"archive/tar"
	"archive/zip"
	"compress/gzip"
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"time"

	"github.com/PhilipKram/bitbucket-cli/internal/errors"
)

// InstallMethod describes how bb was installed.
type InstallMethod int

const (
	InstallBinary    InstallMethod = iota // standalone binary
	InstallHomebrew                       // Homebrew
	InstallGoInstall                      // go install
)

// UpgradeResult holds the outcome of a successful upgrade.
type UpgradeResult struct {
	PreviousVersion string
	NewVersion      string
}

const upgradeFetchTimeout = 30 * time.Second

// DetectInstallMethod resolves the current executable path and classifies
// how bb was installed.
func DetectInstallMethod() InstallMethod {
	exe, err := os.Executable()
	if err != nil {
		return InstallBinary
	}
	resolved, err := filepath.EvalSymlinks(exe)
	if err != nil {
		return InstallBinary
	}
	return classifyPath(resolved)
}

// classifyPath determines the install method from a resolved binary path.
func classifyPath(resolved string) InstallMethod {
	lower := strings.ToLower(resolved)
	if strings.Contains(lower, "/cellar/") || strings.Contains(lower, "/homebrew/") {
		return InstallHomebrew
	}
	gopath := os.Getenv("GOPATH")
	if gopath == "" {
		home, _ := os.UserHomeDir()
		gopath = filepath.Join(home, "go")
	}
	if gopath != "" && strings.HasPrefix(resolved, filepath.Join(gopath, "bin")) {
		return InstallGoInstall
	}
	return InstallBinary
}

// CheckUpgrade checks the install method and fetches the latest release.
// It returns the release info or an error. A nil release with nil error
// means already up to date.
func CheckUpgrade(currentVersion string, force bool) (*GHRelease, error) {
	method := DetectInstallMethod()

	if method == InstallHomebrew && !force {
		return nil, &errors.BBError{
			Message:    "bb was installed via Homebrew",
			Suggestion: "Run 'brew upgrade bb' to update, or use 'bb upgrade --force' to override.",
		}
	}
	if method == InstallGoInstall && !force {
		return nil, &errors.BBError{
			Message:    "bb was installed via go install",
			Suggestion: "Run 'go install github.com/PhilipKram/bitbucket-cli@latest' to update, or use 'bb upgrade --force' to override.",
		}
	}

	rel, err := FetchLatestRelease(upgradeFetchTimeout)
	if err != nil {
		return nil, &errors.BBError{
			Message:    "Failed to check for updates",
			Suggestion: "Check your internet connection and try again.",
			Err:        err,
		}
	}

	current := strings.TrimPrefix(currentVersion, "v")
	latest := strings.TrimPrefix(rel.TagName, "v")

	if current == latest && !force {
		return nil, nil // already up to date
	}

	return rel, nil
}

// Upgrade performs a self-update of the bb binary. It checks for updates,
// downloads the new version, and replaces the current binary.
func Upgrade(currentVersion string, force bool) (*UpgradeResult, error) {
	rel, err := CheckUpgrade(currentVersion, force)
	if err != nil {
		return nil, err
	}
	if rel == nil {
		return nil, nil
	}
	return ApplyUpgrade(currentVersion, rel)
}

// ApplyUpgrade downloads and installs the release, replacing the current binary.
func ApplyUpgrade(currentVersion string, rel *GHRelease) (*UpgradeResult, error) {
	current := strings.TrimPrefix(currentVersion, "v")
	latest := strings.TrimPrefix(rel.TagName, "v")

	assetName := buildAssetName(latest, runtime.GOOS, runtime.GOARCH)
	assetURL, err := findAssetURL(rel.Assets, assetName)
	if err != nil {
		return nil, err
	}

	exe, err := os.Executable()
	if err != nil {
		return nil, errors.Wrap(err, "failed to determine executable path")
	}
	exePath, err := filepath.EvalSymlinks(exe)
	if err != nil {
		return nil, errors.Wrap(err, "failed to resolve executable path")
	}

	if err := checkWritePermission(exePath); err != nil {
		return nil, &errors.BBError{
			Message:    "Permission denied: cannot write to " + exePath,
			Suggestion: "Try running with elevated permissions: sudo bb upgrade",
			Err:        err,
		}
	}

	tmpDir, err := os.MkdirTemp("", "bb-upgrade-*")
	if err != nil {
		return nil, errors.Wrap(err, "failed to create temp directory")
	}
	defer os.RemoveAll(tmpDir)

	archivePath := filepath.Join(tmpDir, assetName)
	if err := downloadAsset(assetURL, archivePath); err != nil {
		return nil, err
	}

	newBinaryPath := filepath.Join(tmpDir, "bb")
	if runtime.GOOS == "windows" {
		newBinaryPath += ".exe"
	}

	if strings.HasSuffix(assetName, ".zip") {
		err = extractBinaryFromZip(archivePath, newBinaryPath)
	} else {
		err = extractBinaryFromTarGz(archivePath, newBinaryPath)
	}
	if err != nil {
		return nil, err
	}

	if err := replaceBinary(exePath, newBinaryPath); err != nil {
		return nil, err
	}

	return &UpgradeResult{
		PreviousVersion: current,
		NewVersion:      latest,
	}, nil
}

// buildAssetName constructs the expected archive filename for the given
// version, OS, and architecture (matching .goreleaser.yml naming).
func buildAssetName(version, goos, goarch string) string {
	version = strings.TrimPrefix(version, "v")
	ext := ".tar.gz"
	if goos == "windows" {
		ext = ".zip"
	}
	return fmt.Sprintf("bb_%s_%s_%s%s", version, goos, goarch, ext)
}

// findAssetURL searches the release assets for the matching archive name.
func findAssetURL(assets []GHReleaseAsset, name string) (string, error) {
	for _, a := range assets {
		if a.Name == name {
			return a.BrowserDownloadURL, nil
		}
	}
	return "", &errors.BBError{
		Message:    fmt.Sprintf("No binary available for %s/%s", runtime.GOOS, runtime.GOARCH),
		Suggestion: "Download manually from https://github.com/PhilipKram/Bitbucket-CLI/releases",
	}
}

// downloadAsset downloads a file from url to destPath.
func downloadAsset(url, destPath string) error {
	client := &http.Client{Timeout: 5 * time.Minute}
	resp, err := client.Get(url)
	if err != nil {
		return &errors.BBError{
			Message:    "Failed to download update",
			Suggestion: "Check your internet connection and try again.",
			Err:        err,
		}
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return &errors.BBError{
			Message: fmt.Sprintf("Download failed with status %d", resp.StatusCode),
		}
	}

	out, err := os.Create(destPath)
	if err != nil {
		return errors.Wrap(err, "failed to create download file")
	}
	defer out.Close()

	_, err = io.Copy(out, resp.Body)
	if err != nil {
		return errors.Wrap(err, "failed to write download file")
	}
	return nil
}

// extractBinaryFromTarGz extracts the "bb" binary from a tar.gz archive.
func extractBinaryFromTarGz(archivePath, destPath string) error {
	f, err := os.Open(archivePath)
	if err != nil {
		return errors.Wrap(err, "failed to open archive")
	}
	defer f.Close()

	gz, err := gzip.NewReader(f)
	if err != nil {
		return errors.Wrap(err, "failed to decompress archive")
	}
	defer gz.Close()

	tr := tar.NewReader(gz)
	binaryName := "bb"
	if runtime.GOOS == "windows" {
		binaryName = "bb.exe"
	}

	for {
		hdr, err := tr.Next()
		if err == io.EOF {
			break
		}
		if err != nil {
			return errors.Wrap(err, "failed to read archive")
		}
		if filepath.Base(hdr.Name) == binaryName && hdr.Typeflag == tar.TypeReg {
			out, err := os.OpenFile(destPath, os.O_CREATE|os.O_WRONLY|os.O_TRUNC, 0755)
			if err != nil {
				return errors.Wrap(err, "failed to create binary file")
			}
			defer out.Close()
			if _, err := io.Copy(out, tr); err != nil {
				return errors.Wrap(err, "failed to extract binary")
			}
			return nil
		}
	}
	return errors.New("binary not found in archive")
}

// extractBinaryFromZip extracts the "bb.exe" binary from a zip archive.
func extractBinaryFromZip(archivePath, destPath string) error {
	r, err := zip.OpenReader(archivePath)
	if err != nil {
		return errors.Wrap(err, "failed to open zip archive")
	}
	defer r.Close()

	binaryName := "bb"
	if runtime.GOOS == "windows" {
		binaryName = "bb.exe"
	}

	for _, f := range r.File {
		if filepath.Base(f.Name) == binaryName {
			rc, err := f.Open()
			if err != nil {
				return errors.Wrap(err, "failed to read zip entry")
			}
			defer rc.Close()

			out, err := os.OpenFile(destPath, os.O_CREATE|os.O_WRONLY|os.O_TRUNC, 0755)
			if err != nil {
				return errors.Wrap(err, "failed to create binary file")
			}
			defer out.Close()

			if _, err := io.Copy(out, rc); err != nil {
				return errors.Wrap(err, "failed to extract binary")
			}
			return nil
		}
	}
	return errors.New("binary not found in zip archive")
}

// replaceBinary atomically replaces the current binary with the new one.
func replaceBinary(currentPath, newPath string) error {
	oldPath := currentPath + ".old"

	// Remove any leftover .old file from a previous upgrade.
	os.Remove(oldPath)

	// Rename current → .old
	if err := os.Rename(currentPath, oldPath); err != nil {
		return &errors.BBError{
			Message:    "Failed to replace binary",
			Suggestion: "Try running with elevated permissions: sudo bb upgrade",
			Err:        err,
		}
	}

	// Rename new → current
	if err := os.Rename(newPath, currentPath); err != nil {
		// Try to rollback
		_ = os.Rename(oldPath, currentPath)
		return &errors.BBError{
			Message:    "Failed to install new binary",
			Suggestion: "Try running with elevated permissions: sudo bb upgrade",
			Err:        err,
		}
	}

	// Clean up .old file
	os.Remove(oldPath)
	return nil
}

// checkWritePermission tests whether the binary path is writable.
func checkWritePermission(path string) error {
	dir := filepath.Dir(path)
	tmp, err := os.CreateTemp(dir, ".bb-write-test-*")
	if err != nil {
		return err
	}
	name := tmp.Name()
	tmp.Close()
	os.Remove(name)
	return nil
}
