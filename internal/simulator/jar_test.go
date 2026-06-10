package simulator

import (
	"crypto/sha256"
	"encoding/hex"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"testing"
)

func sha256Hex(content string) string {
	sum := sha256.Sum256([]byte(content))
	return hex.EncodeToString(sum[:])
}

func withExecutablePathStub(t *testing.T, path string) {
	t.Helper()

	restore := executablePathFn
	executablePathFn = func() (string, error) {
		return path, nil
	}
	t.Cleanup(func() {
		executablePathFn = restore
	})
}

func withLatestReleaseURLStub(t *testing.T, url string) {
	t.Helper()

	restore := latestReleaseURL
	latestReleaseURL = url
	t.Cleanup(func() {
		latestReleaseURL = restore
	})
}

func TestEnsureArtifactReturnsExistingLocalArtifact(t *testing.T) {
	dir := t.TempDir()
	artifactPath := filepath.Join(dir, localSimulatorArtifactName)
	withExecutablePathStub(t, filepath.Join(dir, "simulador"))

	if err := os.WriteFile(artifactPath, []byte("existing"), 0o644); err != nil {
		t.Fatalf("failed to create local artifact: %v", err)
	}

	result, err := EnsureArtifact("", "")
	if err != nil {
		t.Fatalf("EnsureArtifact failed: %v", err)
	}
	if result.Path != artifactPath {
		t.Fatalf("expected path %q, got %q", artifactPath, result.Path)
	}
	if result.Downloaded {
		t.Fatal("expected existing artifact not to be marked as downloaded")
	}
}

func TestEnsureArtifactDownloadsFromSourceWhenMissing(t *testing.T) {
	dir := t.TempDir()
	artifactPath := filepath.Join(dir, localSimulatorArtifactName)
	withExecutablePathStub(t, filepath.Join(dir, "simulador"))

	source := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte("downloaded"))
	}))
	defer source.Close()

	result, err := EnsureArtifact(source.URL+"/simulador", sha256Hex("downloaded"))
	if err != nil {
		t.Fatalf("EnsureArtifact failed: %v", err)
	}
	if result.Path != artifactPath {
		t.Fatalf("expected path %q, got %q", artifactPath, result.Path)
	}
	if !result.Downloaded {
		t.Fatal("expected missing artifact to be downloaded")
	}

	got, err := os.ReadFile(artifactPath)
	if err != nil {
		t.Fatalf("expected downloaded artifact to exist: %v", err)
	}
	if string(got) != "downloaded" {
		t.Fatalf("unexpected downloaded content: %q", got)
	}
}

func TestEnsureArtifactRejectsSourceDownloadWhenChecksumDoesNotMatch(t *testing.T) {
	dir := t.TempDir()
	artifactPath := filepath.Join(dir, localSimulatorArtifactName)
	withExecutablePathStub(t, filepath.Join(dir, "simulador"))

	source := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte("downloaded"))
	}))
	defer source.Close()

	if _, err := EnsureArtifact(source.URL+"/simulador", sha256Hex("other content")); err == nil {
		t.Fatal("expected checksum mismatch to fail")
	}
	if _, err := os.Stat(artifactPath); !os.IsNotExist(err) {
		t.Fatalf("expected artifact not to be installed after checksum mismatch, got err=%v", err)
	}
}

func TestEnsureArtifactRequiresChecksumForSourceDownload(t *testing.T) {
	dir := t.TempDir()
	withExecutablePathStub(t, filepath.Join(dir, "simulador"))

	if _, err := EnsureArtifact("https://example.test/simulador", ""); err == nil {
		t.Fatal("expected source download without checksum to fail")
	}
}

func TestEnsureArtifactReportsLatestReleaseLookupFailure(t *testing.T) {
	dir := t.TempDir()
	withExecutablePathStub(t, filepath.Join(dir, "simulador"))
	withLatestReleaseURLStub(t, "://invalid")

	if _, err := EnsureArtifact("", ""); err == nil {
		t.Fatal("expected missing artifact without source to fail")
	}
}

func TestEnsureArtifactDownloadsFromLatestGitHubReleaseWhenSourceIsMissing(t *testing.T) {
	dir := t.TempDir()
	artifactPath := filepath.Join(dir, localSimulatorArtifactName)
	withExecutablePathStub(t, filepath.Join(dir, "simulador"))

	var serverURL string
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case "/repos/example/project/releases/latest":
			w.Header().Set("Content-Type", "application/json")
			_, _ = w.Write([]byte(`{
				"tag_name": "v1.2.3",
				"assets": [
					{"name": "other.jar", "browser_download_url": "` + serverURL + `/other.jar"},
					{"name": "simulador-v1.2.3-darwin-arm64", "browser_download_url": "` + serverURL + `/simulador-v1.2.3-darwin-arm64"},
					{"name": "checksums.txt", "browser_download_url": "` + serverURL + `/checksums.txt"}
				]
			}`))
		case "/simulador-v1.2.3-darwin-arm64":
			w.WriteHeader(http.StatusOK)
			_, _ = w.Write([]byte("from latest release"))
		case "/checksums.txt":
			w.WriteHeader(http.StatusOK)
			_, _ = w.Write([]byte(sha256Hex("from latest release") + "  simulador-v1.2.3-darwin-arm64\n"))
		default:
			http.NotFound(w, r)
		}
	}))
	serverURL = server.URL
	defer server.Close()
	withLatestReleaseURLStub(t, server.URL+"/repos/example/project/releases/latest")

	result, err := EnsureArtifact("", "")
	if err != nil {
		t.Fatalf("EnsureArtifact failed: %v", err)
	}
	if !result.Downloaded {
		t.Fatal("expected artifact to be downloaded")
	}
	if result.Version != "v1.2.3" {
		t.Fatalf("expected version v1.2.3, got %q", result.Version)
	}
	if result.SourceURL != server.URL+"/simulador-v1.2.3-darwin-arm64" {
		t.Fatalf("unexpected source URL: %q", result.SourceURL)
	}
	if result.Checksum != sha256Hex("from latest release") {
		t.Fatalf("unexpected checksum: %q", result.Checksum)
	}

	got, err := os.ReadFile(artifactPath)
	if err != nil {
		t.Fatalf("expected downloaded artifact to exist: %v", err)
	}
	if string(got) != "from latest release" {
		t.Fatalf("unexpected downloaded content: %q", got)
	}
}

func TestEnsureArtifactReportsMissingAssetInLatestRelease(t *testing.T) {
	dir := t.TempDir()
	withExecutablePathStub(t, filepath.Join(dir, "simulador"))

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"tag_name":"v1.2.3","assets":[]}`))
	}))
	defer server.Close()
	withLatestReleaseURLStub(t, server.URL)

	if _, err := EnsureArtifact("", ""); err == nil {
		t.Fatal("expected missing release asset to fail")
	}
}

func TestEnsureArtifactReportsInvalidLatestReleaseJSON(t *testing.T) {
	dir := t.TempDir()
	withExecutablePathStub(t, filepath.Join(dir, "simulador"))

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte(`{`))
	}))
	defer server.Close()
	withLatestReleaseURLStub(t, server.URL)

	if _, err := EnsureArtifact("", ""); err == nil {
		t.Fatal("expected invalid release JSON to fail")
	}
}

func TestEnsureArtifactUsesReleaseAssetDigestAsChecksum(t *testing.T) {
	dir := t.TempDir()
	withExecutablePathStub(t, filepath.Join(dir, "simulador"))

	var serverURL string
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case "/release":
			w.Header().Set("Content-Type", "application/json")
			_, _ = w.Write([]byte(`{
				"tag_name": "v1.2.3",
				"assets": [
					{"name": "simulador-v1.2.3-darwin-arm64", "browser_download_url": "` + serverURL + `/simulador-v1.2.3-darwin-arm64", "digest": "sha256:` + sha256Hex("release digest") + `"}
				]
			}`))
		case "/simulador-v1.2.3-darwin-arm64":
			w.WriteHeader(http.StatusOK)
			_, _ = w.Write([]byte("release digest"))
		default:
			http.NotFound(w, r)
		}
	}))
	serverURL = server.URL
	defer server.Close()
	withLatestReleaseURLStub(t, server.URL+"/release")

	result, err := EnsureArtifact("", "")
	if err != nil {
		t.Fatalf("EnsureArtifact failed: %v", err)
	}
	if result.Checksum != sha256Hex("release digest") {
		t.Fatalf("unexpected checksum: %q", result.Checksum)
	}
}

func TestEnsureArtifactAcceptsProvidedChecksumForLatestRelease(t *testing.T) {
	dir := t.TempDir()
	withExecutablePathStub(t, filepath.Join(dir, "simulador"))

	var serverURL string
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case "/release":
			w.Header().Set("Content-Type", "application/json")
			_, _ = w.Write([]byte(`{
				"tag_name": "v1.2.3",
				"assets": [
					{"name": "simulador-v1.2.3-darwin-arm64", "browser_download_url": "` + serverURL + `/simulador-v1.2.3-darwin-arm64"}
				]
			}`))
		case "/simulador-v1.2.3-darwin-arm64":
			w.WriteHeader(http.StatusOK)
			_, _ = w.Write([]byte("provided checksum"))
		default:
			http.NotFound(w, r)
		}
	}))
	serverURL = server.URL
	defer server.Close()
	withLatestReleaseURLStub(t, server.URL+"/release")

	result, err := EnsureArtifact("", sha256Hex("provided checksum"))
	if err != nil {
		t.Fatalf("EnsureArtifact failed: %v", err)
	}
	if result.Checksum != sha256Hex("provided checksum") {
		t.Fatalf("unexpected checksum: %q", result.Checksum)
	}
}
