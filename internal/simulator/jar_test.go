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

func TestEnsureJarReturnsExistingLocalJar(t *testing.T) {
	dir := t.TempDir()
	jarPath := filepath.Join(dir, simulatorJarName)
	withExecutablePathStub(t, filepath.Join(dir, "simulador"))

	if err := os.WriteFile(jarPath, []byte("existing"), 0o644); err != nil {
		t.Fatalf("failed to create local jar: %v", err)
	}

	result, err := EnsureJar("")
	if err != nil {
		t.Fatalf("EnsureJar failed: %v", err)
	}
	if result.Path != jarPath {
		t.Fatalf("expected path %q, got %q", jarPath, result.Path)
	}
	if result.Downloaded {
		t.Fatal("expected existing jar not to be marked as downloaded")
	}
}

func TestEnsureJarDownloadsFromSourceWhenMissing(t *testing.T) {
	dir := t.TempDir()
	jarPath := filepath.Join(dir, simulatorJarName)
	withExecutablePathStub(t, filepath.Join(dir, "simulador"))

	source := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte("downloaded"))
	}))
	defer source.Close()

	result, err := EnsureJar(source.URL + "/simulador.jar")
	if err != nil {
		t.Fatalf("EnsureJar failed: %v", err)
	}
	if result.Path != jarPath {
		t.Fatalf("expected path %q, got %q", jarPath, result.Path)
	}
	if !result.Downloaded {
		t.Fatal("expected missing jar to be downloaded")
	}

	got, err := os.ReadFile(jarPath)
	if err != nil {
		t.Fatalf("expected downloaded jar to exist: %v", err)
	}
	if string(got) != "downloaded" {
		t.Fatalf("unexpected downloaded content: %q", got)
	}
}

func TestEnsureJarReportsLatestReleaseLookupFailure(t *testing.T) {
	dir := t.TempDir()
	withExecutablePathStub(t, filepath.Join(dir, "simulador"))
	withLatestReleaseURLStub(t, "://invalid")

	if _, err := EnsureJar(""); err == nil {
		t.Fatal("expected missing jar without source to fail")
	}
}

func TestEnsureJarDownloadsFromLatestGitHubReleaseWhenSourceIsMissing(t *testing.T) {
	dir := t.TempDir()
	jarPath := filepath.Join(dir, simulatorJarName)
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
					{"name": "simulador.jar", "browser_download_url": "` + serverURL + `/simulador.jar"}
				]
			}`))
		case "/simulador.jar":
			w.WriteHeader(http.StatusOK)
			_, _ = w.Write([]byte("from latest release"))
		default:
			http.NotFound(w, r)
		}
	}))
	serverURL = server.URL
	defer server.Close()
	withLatestReleaseURLStub(t, server.URL+"/repos/example/project/releases/latest")

	result, err := EnsureJar("")
	if err != nil {
		t.Fatalf("EnsureJar failed: %v", err)
	}
	if !result.Downloaded {
		t.Fatal("expected jar to be downloaded")
	}
	if result.Version != "v1.2.3" {
		t.Fatalf("expected version v1.2.3, got %q", result.Version)
	}
	if result.SourceURL != server.URL+"/simulador.jar" {
		t.Fatalf("unexpected source URL: %q", result.SourceURL)
	}

	got, err := os.ReadFile(jarPath)
	if err != nil {
		t.Fatalf("expected downloaded jar to exist: %v", err)
	}
	if string(got) != "from latest release" {
		t.Fatalf("unexpected downloaded content: %q", got)
	}
}

func TestEnsureJarReportsMissingAssetInLatestRelease(t *testing.T) {
	dir := t.TempDir()
	withExecutablePathStub(t, filepath.Join(dir, "simulador"))

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"tag_name":"v1.2.3","assets":[]}`))
	}))
	defer server.Close()
	withLatestReleaseURLStub(t, server.URL)

	if _, err := EnsureJar(""); err == nil {
		t.Fatal("expected missing release asset to fail")
	}
}

func TestEnsureJarReportsInvalidLatestReleaseJSON(t *testing.T) {
	dir := t.TempDir()
	withExecutablePathStub(t, filepath.Join(dir, "simulador"))

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte(`{`))
	}))
	defer server.Close()
	withLatestReleaseURLStub(t, server.URL)

	if _, err := EnsureJar(""); err == nil {
		t.Fatal("expected invalid release JSON to fail")
	}
}
