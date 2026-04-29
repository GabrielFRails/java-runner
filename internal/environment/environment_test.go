package environment

import (
	"os"
	"path/filepath"
	"runtime"
	"testing"
)

func TestManagedJDKExecutableReturnsLatestCandidate(t *testing.T) {
	t.Parallel()

	tmpHome := t.TempDir()
	restore := homeDirFn
	homeDirFn = func() (string, error) { return tmpHome, nil }
	t.Cleanup(func() { homeDirFn = restore })

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
	t.Parallel()

	tmpHome := t.TempDir()
	restore := homeDirFn
	homeDirFn = func() (string, error) { return tmpHome, nil }
	t.Cleanup(func() { homeDirFn = restore })

	_, err := managedJDKExecutable()
	if !os.IsNotExist(err) {
		t.Fatalf("expected os.ErrNotExist when no managed JDK exists, got %v", err)
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
