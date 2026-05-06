package environment

import (
	"os"
	"path/filepath"
	"runtime"
	"testing"
)

func withEnvironmentStubs(t *testing.T) {
	t.Helper()

	restoreHome := homeDirFn
	restoreLookPath := lookPathFn
	restoreVersion := getJavaVersionFn
	t.Cleanup(func() {
		homeDirFn = restoreHome
		lookPathFn = restoreLookPath
		getJavaVersionFn = restoreVersion
	})
}

func TestManagedJDKExecutableReturnsLatestCandidate(t *testing.T) {
	withEnvironmentStubs(t)

	tmpHome := t.TempDir()
	homeDirFn = func() (string, error) { return tmpHome, nil }

	first := filepath.Join(tmpHome, ".hubsaude", "jdk", "jdk-21.0.1")
	second := filepath.Join(tmpHome, ".hubsaude", "jdk", "jdk-21.0.2")

	if err := os.MkdirAll(filepath.Dir(javaHomeExecutable(first)), 0o755); err != nil {
		t.Fatalf("failed to create first managed JDK dir: %v", err)
	}
	if err := os.MkdirAll(filepath.Dir(javaHomeExecutable(second)), 0o755); err != nil {
		t.Fatalf("failed to create second managed JDK dir: %v", err)
	}

	if err := os.WriteFile(javaHomeExecutable(first), []byte(""), 0o755); err != nil {
		t.Fatalf("failed to create first fake java: %v", err)
	}
	if err := os.WriteFile(javaHomeExecutable(second), []byte(""), 0o755); err != nil {
		t.Fatalf("failed to create second fake java: %v", err)
	}

	got, err := managedJDKExecutable()
	if err != nil {
		t.Fatalf("expected managed JDK candidate, got error: %v", err)
	}

	want := javaHomeExecutable(second)
	if got != want {
		t.Fatalf("expected latest candidate %q, got %q", want, got)
	}
}

func TestManagedJDKExecutableReturnsNotExistWhenMissing(t *testing.T) {
	withEnvironmentStubs(t)

	tmpHome := t.TempDir()
	homeDirFn = func() (string, error) { return tmpHome, nil }

	_, err := managedJDKExecutable()
	if !os.IsNotExist(err) {
		t.Fatalf("expected os.ErrNotExist when no managed JDK exists, got %v", err)
	}
}

func TestAdoptiumOSMappings(t *testing.T) {
	t.Parallel()

	cases := map[string]string{
		"darwin":  "mac",
		"linux":   "linux",
		"windows": "windows",
	}

	for input, want := range cases {
		got, err := adoptiumOS(input)
		if err != nil {
			t.Fatalf("unexpected error for %s: %v", input, err)
		}
		if got != want {
			t.Fatalf("expected %s -> %s, got %s", input, want, got)
		}
	}
}

func TestAdoptiumArchMappings(t *testing.T) {
	t.Parallel()

	cases := map[string]string{
		"amd64": "x64",
		"arm64": "aarch64",
	}

	for input, want := range cases {
		got, err := adoptiumArch(input)
		if err != nil {
			t.Fatalf("unexpected error for %s: %v", input, err)
		}
		if got != want {
			t.Fatalf("expected %s -> %s, got %s", input, want, got)
		}
	}
}

func TestArchiveExtensionMatchesPlatform(t *testing.T) {
	t.Parallel()

	if got := archiveExtension("windows"); got != "zip" {
		t.Fatalf("expected windows archive extension zip, got %s", got)
	}
	if got := archiveExtension("darwin"); got != "tar.gz" {
		t.Fatalf("expected darwin archive extension tar.gz, got %s", got)
	}
}

func TestJavaHomeExecutableSupportsMacBundleLayout(t *testing.T) {
	t.Parallel()

	if runtime.GOOS != "darwin" {
		t.Skip("macOS-specific path behavior")
	}

	base := t.TempDir()
	candidate := filepath.Join(base, "Contents", "Home", "bin")
	if err := os.MkdirAll(candidate, 0o755); err != nil {
		t.Fatalf("failed to create mac bundle dir: %v", err)
	}
	javaPath := filepath.Join(candidate, javaExecutable())
	if err := os.WriteFile(javaPath, []byte(""), 0o755); err != nil {
		t.Fatalf("failed to create mac bundle java executable: %v", err)
	}

	got := javaHomeExecutable(base)
	if got != javaPath {
		t.Fatalf("expected mac bundle executable %q, got %q", javaPath, got)
	}
}

func TestDetectJavaPrefersManagedCompatibleJDK(t *testing.T) {
	withEnvironmentStubs(t)

	tmpHome := t.TempDir()
	homeDirFn = func() (string, error) { return tmpHome, nil }

	managedDir := filepath.Join(tmpHome, ".hubsaude", "jdk", "temurin-21-darwin-arm64")
	javaPath := javaHomeExecutable(managedDir)
	if err := os.MkdirAll(filepath.Dir(javaPath), 0o755); err != nil {
		t.Fatalf("failed to create managed java dir: %v", err)
	}
	if err := os.WriteFile(javaPath, []byte(""), 0o755); err != nil {
		t.Fatalf("failed to create managed java file: %v", err)
	}

	getJavaVersionFn = func(path string) (string, error) {
		if path == javaPath {
			return "21.0.7", nil
		}
		return "", os.ErrNotExist
	}
	lookPathFn = func(file string) (string, error) { return "", os.ErrNotExist }

	info, err := DetectJava()
	if err != nil {
		t.Fatalf("expected managed compatible JDK, got error: %v", err)
	}
	if info.Source != "downloaded" {
		t.Fatalf("expected source downloaded, got %s", info.Source)
	}
	if info.Path != javaPath {
		t.Fatalf("expected managed java path %q, got %q", javaPath, info.Path)
	}
}

func TestDetectJavaIgnoresIncompatibleSystemAndUsesManagedCompatibleJDK(t *testing.T) {
	withEnvironmentStubs(t)

	tmpHome := t.TempDir()
	homeDirFn = func() (string, error) { return tmpHome, nil }

	systemJavaPath := filepath.Join(tmpHome, "bin", javaExecutable())
	managedInstallDir, err := managedJDKInstallDir()
	if err != nil {
		t.Fatalf("failed to compute managed install dir: %v", err)
	}
	managedJavaPath := javaHomeExecutable(managedInstallDir)

	if err := os.MkdirAll(filepath.Dir(systemJavaPath), 0o755); err != nil {
		t.Fatalf("failed to create system java dir: %v", err)
	}
	if err := os.WriteFile(systemJavaPath, []byte(""), 0o755); err != nil {
		t.Fatalf("failed to create system java file: %v", err)
	}
	if err := os.MkdirAll(filepath.Dir(managedJavaPath), 0o755); err != nil {
		t.Fatalf("failed to create managed java dir: %v", err)
	}
	if err := os.WriteFile(managedJavaPath, []byte(""), 0o755); err != nil {
		t.Fatalf("failed to create managed java file: %v", err)
	}

	lookPathFn = func(file string) (string, error) { return systemJavaPath, nil }
	getJavaVersionFn = func(path string) (string, error) {
		switch path {
		case systemJavaPath:
			return "17.0.9", nil
		case managedJavaPath:
			return "21.0.7", nil
		default:
			return "", os.ErrNotExist
		}
	}

	info, err := DetectJava()
	if err != nil {
		t.Fatalf("expected fallback to managed JDK, got error: %v", err)
	}
	if info.Path != managedJavaPath {
		t.Fatalf("expected managed java path %q, got %q", managedJavaPath, info.Path)
	}
	if info.Source != "downloaded" {
		t.Fatalf("expected downloaded source, got %s", info.Source)
	}
}
