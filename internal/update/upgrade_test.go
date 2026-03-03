package update

import (
	"archive/tar"
	"compress/gzip"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"testing"
)

func TestBuildAssetName(t *testing.T) {
	tests := []struct {
		version string
		goos    string
		goarch  string
		want    string
	}{
		{"0.0.8", "darwin", "arm64", "bb_0.0.8_darwin_arm64.tar.gz"},
		{"0.0.8", "linux", "amd64", "bb_0.0.8_linux_amd64.tar.gz"},
		{"0.0.8", "windows", "amd64", "bb_0.0.8_windows_amd64.zip"},
		{"v0.0.8", "darwin", "amd64", "bb_0.0.8_darwin_amd64.tar.gz"},
		{"1.2.3", "linux", "arm64", "bb_1.2.3_linux_arm64.tar.gz"},
	}

	for _, tt := range tests {
		t.Run(tt.want, func(t *testing.T) {
			got := buildAssetName(tt.version, tt.goos, tt.goarch)
			if got != tt.want {
				t.Errorf("buildAssetName(%q, %q, %q) = %q, want %q",
					tt.version, tt.goos, tt.goarch, got, tt.want)
			}
		})
	}
}

func TestClassifyPath(t *testing.T) {
	tests := []struct {
		name string
		path string
		want InstallMethod
	}{
		{"homebrew cellar", "/opt/homebrew/Cellar/bb/0.0.7/bin/bb", InstallHomebrew},
		{"homebrew linuxbrew", "/home/linuxbrew/.linuxbrew/Cellar/bb/0.0.7/bin/bb", InstallHomebrew},
		{"homebrew path", "/usr/local/homebrew/bin/bb", InstallHomebrew},
		{"plain binary", "/usr/local/bin/bb", InstallBinary},
		{"tmp binary", "/tmp/bb", InstallBinary},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := classifyPath(tt.path)
			if got != tt.want {
				t.Errorf("classifyPath(%q) = %d, want %d", tt.path, got, tt.want)
			}
		})
	}
}

func TestClassifyPath_GoInstall(t *testing.T) {
	tmpDir := t.TempDir()
	t.Setenv("GOPATH", tmpDir)

	path := filepath.Join(tmpDir, "bin", "bb")
	got := classifyPath(path)
	if got != InstallGoInstall {
		t.Errorf("classifyPath(%q) = %d, want InstallGoInstall (%d)", path, got, InstallGoInstall)
	}
}

func TestClassifyPath_GoInstall_BoundaryCheck(t *testing.T) {
	tmpDir := t.TempDir()
	t.Setenv("GOPATH", tmpDir)

	// $GOPATH/bin2/bb should NOT be classified as go-install
	path := filepath.Join(tmpDir, "bin2", "bb")
	got := classifyPath(path)
	if got != InstallBinary {
		t.Errorf("classifyPath(%q) = %d, want InstallBinary (%d)", path, got, InstallBinary)
	}
}

func TestFindAssetURL(t *testing.T) {
	assets := []GHReleaseAsset{
		{Name: "bb_0.0.8_darwin_arm64.tar.gz", BrowserDownloadURL: "https://example.com/darwin_arm64.tar.gz"},
		{Name: "bb_0.0.8_linux_amd64.tar.gz", BrowserDownloadURL: "https://example.com/linux_amd64.tar.gz"},
		{Name: "bb_0.0.8_windows_amd64.zip", BrowserDownloadURL: "https://example.com/windows_amd64.zip"},
	}

	t.Run("found", func(t *testing.T) {
		url, err := findAssetURL(assets, "bb_0.0.8_darwin_arm64.tar.gz")
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if url != "https://example.com/darwin_arm64.tar.gz" {
			t.Errorf("got %q, want %q", url, "https://example.com/darwin_arm64.tar.gz")
		}
	})

	t.Run("not found", func(t *testing.T) {
		_, err := findAssetURL(assets, "bb_0.0.8_freebsd_amd64.tar.gz")
		if err == nil {
			t.Fatal("expected error for missing asset")
		}
	})
}

func TestExtractBinaryFromTarGz(t *testing.T) {
	tmpDir := t.TempDir()

	// Create a tar.gz archive with a fake bb binary
	archivePath := filepath.Join(tmpDir, "bb_0.0.8_test.tar.gz")
	binaryContent := []byte("#!/bin/sh\necho fake-bb\n")

	f, err := os.Create(archivePath)
	if err != nil {
		t.Fatalf("failed to create archive file: %v", err)
	}

	gw := gzip.NewWriter(f)
	tw := tar.NewWriter(gw)

	// Add the bb binary to the archive
	if err := tw.WriteHeader(&tar.Header{
		Name:     "bb",
		Mode:     0755,
		Size:     int64(len(binaryContent)),
		Typeflag: tar.TypeReg,
	}); err != nil {
		t.Fatalf("failed to write tar header: %v", err)
	}
	if _, err := tw.Write(binaryContent); err != nil {
		t.Fatalf("failed to write tar content: %v", err)
	}

	// Add a LICENSE file (should be skipped)
	licenseContent := []byte("MIT License\n")
	if err := tw.WriteHeader(&tar.Header{
		Name:     "LICENSE",
		Mode:     0644,
		Size:     int64(len(licenseContent)),
		Typeflag: tar.TypeReg,
	}); err != nil {
		t.Fatalf("failed to write license header: %v", err)
	}
	if _, err := tw.Write(licenseContent); err != nil {
		t.Fatalf("failed to write license content: %v", err)
	}

	tw.Close()
	gw.Close()
	f.Close()

	// Extract
	destPath := filepath.Join(tmpDir, "bb-extracted")
	if err := extractBinaryFromTarGz(archivePath, destPath); err != nil {
		t.Fatalf("extractBinaryFromTarGz() error: %v", err)
	}

	// Verify
	content, err := os.ReadFile(destPath)
	if err != nil {
		t.Fatalf("failed to read extracted binary: %v", err)
	}
	if string(content) != string(binaryContent) {
		t.Errorf("extracted content = %q, want %q", content, binaryContent)
	}

	info, err := os.Stat(destPath)
	if err != nil {
		t.Fatalf("failed to stat extracted binary: %v", err)
	}
	if info.Mode()&0111 == 0 {
		t.Errorf("expected executable permissions, got %v", info.Mode())
	}
}

func TestExtractBinaryFromTarGz_NotFound(t *testing.T) {
	tmpDir := t.TempDir()

	// Create a tar.gz archive without a bb binary
	archivePath := filepath.Join(tmpDir, "no_bb.tar.gz")
	f, err := os.Create(archivePath)
	if err != nil {
		t.Fatalf("failed to create archive: %v", err)
	}

	gw := gzip.NewWriter(f)
	tw := tar.NewWriter(gw)

	content := []byte("MIT License\n")
	tw.WriteHeader(&tar.Header{
		Name:     "LICENSE",
		Mode:     0644,
		Size:     int64(len(content)),
		Typeflag: tar.TypeReg,
	})
	tw.Write(content)
	tw.Close()
	gw.Close()
	f.Close()

	destPath := filepath.Join(tmpDir, "bb-extracted")
	err = extractBinaryFromTarGz(archivePath, destPath)
	if err == nil {
		t.Fatal("expected error when binary not found in archive")
	}
}

func TestReplaceBinary(t *testing.T) {
	tmpDir := t.TempDir()

	currentPath := filepath.Join(tmpDir, "bb")
	newPath := filepath.Join(tmpDir, "bb-new")

	// Create current binary
	if err := os.WriteFile(currentPath, []byte("old-binary"), 0755); err != nil {
		t.Fatalf("failed to write current binary: %v", err)
	}

	// Create new binary
	if err := os.WriteFile(newPath, []byte("new-binary"), 0755); err != nil {
		t.Fatalf("failed to write new binary: %v", err)
	}

	// Replace
	if err := replaceBinary(currentPath, newPath); err != nil {
		t.Fatalf("replaceBinary() error: %v", err)
	}

	// Verify current path has new content
	content, err := os.ReadFile(currentPath)
	if err != nil {
		t.Fatalf("failed to read replaced binary: %v", err)
	}
	if string(content) != "new-binary" {
		t.Errorf("replaced binary content = %q, want %q", content, "new-binary")
	}

	// Verify .old file is cleaned up
	if _, err := os.Stat(currentPath + ".old"); !os.IsNotExist(err) {
		t.Error("expected .old file to be removed")
	}
}

func TestDownloadAsset(t *testing.T) {
	content := []byte("fake-archive-content")
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		w.Write(content)
	}))
	defer server.Close()

	tmpDir := t.TempDir()
	destPath := filepath.Join(tmpDir, "downloaded.tar.gz")

	if err := downloadAsset(server.URL, destPath); err != nil {
		t.Fatalf("downloadAsset() error: %v", err)
	}

	got, err := os.ReadFile(destPath)
	if err != nil {
		t.Fatalf("failed to read downloaded file: %v", err)
	}
	if string(got) != string(content) {
		t.Errorf("downloaded content = %q, want %q", got, content)
	}
}

func TestDownloadAsset_ServerError(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusInternalServerError)
	}))
	defer server.Close()

	tmpDir := t.TempDir()
	destPath := filepath.Join(tmpDir, "downloaded.tar.gz")

	err := downloadAsset(server.URL, destPath)
	if err == nil {
		t.Fatal("expected error for server error response")
	}
}

func TestCheckUpgrade_AlreadyUpToDate(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		w.Write([]byte(`{"tag_name": "v1.0.0", "assets": []}`))
	}))
	defer server.Close()

	// Override releaseURL to use mock server.
	old := releaseURL
	releaseURL = server.URL
	t.Cleanup(func() { releaseURL = old })

	rel, err := CheckUpgrade("v1.0.0", false)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if rel != nil {
		t.Errorf("expected nil release for same version, got %+v", rel)
	}
}

func TestCheckUpgrade_NewerAvailable(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		w.Write([]byte(`{"tag_name": "v2.0.0", "assets": [{"name": "bb_2.0.0_linux_amd64.tar.gz", "browser_download_url": "https://example.com/bb.tar.gz"}]}`))
	}))
	defer server.Close()

	old := releaseURL
	releaseURL = server.URL
	t.Cleanup(func() { releaseURL = old })

	rel, err := CheckUpgrade("v1.0.0", false)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if rel == nil {
		t.Fatal("expected non-nil release for newer version")
	}
	if rel.TagName != "v2.0.0" {
		t.Errorf("expected tag v2.0.0, got %s", rel.TagName)
	}
}

func TestCheckWritePermission(t *testing.T) {
	tmpDir := t.TempDir()
	path := filepath.Join(tmpDir, "bb")

	err := checkWritePermission(path)
	if err != nil {
		t.Errorf("expected writable temp dir, got error: %v", err)
	}
}
