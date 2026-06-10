package simulator

import (
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"testing"
)

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

	result, err := EnsureArtifact("")
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

	result, err := EnsureArtifact(source.URL + "/simulador")
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

func TestEnsureArtifactReportsLatestReleaseLookupFailure(t *testing.T) {
	dir := t.TempDir()
	withExecutablePathStub(t, filepath.Join(dir, "simulador"))
	withLatestReleaseURLStub(t, "://invalid")

	if _, err := EnsureArtifact(""); err == nil {
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
					{"name": "simulador-v1.2.3-darwin-arm64", "browser_download_url": "` + serverURL + `/simulador-v1.2.3-darwin-arm64"}
				]
			}`))
		case "/simulador-v1.2.3-darwin-arm64":
			w.WriteHeader(http.StatusOK)
			_, _ = w.Write([]byte("from latest release"))
		default:
			http.NotFound(w, r)
		}
	}))
	serverURL = server.URL
	defer server.Close()
	withLatestReleaseURLStub(t, server.URL+"/repos/example/project/releases/latest")

	result, err := EnsureArtifact("")
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

	if _, err := EnsureArtifact(""); err == nil {
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

	if _, err := EnsureArtifact(""); err == nil {
		t.Fatal("expected invalid release JSON to fail")
	}
}
