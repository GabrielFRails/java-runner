package server

import (
	"testing"

	"github.com/kyriosdata/assinatura/internal/storage"
)

func TestStartRejectsInvalidPort(t *testing.T) {
	t.Parallel()

	if _, err := Start("java", "assinador.jar", 0, 0); err == nil {
		t.Fatal("expected error for invalid port")
	}
}

func TestStartRejectsInvalidTimeout(t *testing.T) {
	t.Parallel()

	if _, err := Start("java", "assinador.jar", 8080, -1); err == nil {
		t.Fatal("expected error for invalid timeout")
	}
}

func TestIsHealthyRejectsInvalidPort(t *testing.T) {
	t.Parallel()

	if IsHealthy(0) {
		t.Fatal("expected invalid port to be unhealthy")
	}
}

func TestStopRejectsInvalidPort(t *testing.T) {
	t.Parallel()

	if _, err := Stop(0); err == nil {
		t.Fatal("expected error for invalid port")
	}
}

func TestStopReturnsNotStoppedWhenProcessIsMissing(t *testing.T) {
	t.Setenv("HOME", t.TempDir())

	if err := storage.EnsureHomeDir(); err != nil {
		t.Fatalf("EnsureHomeDir failed: %v", err)
	}
	if err := storage.InitDatabase(); err != nil {
		t.Fatalf("InitDatabase failed: %v", err)
	}
	defer storage.Close()

	result, err := Stop(8080)
	if err != nil {
		t.Fatalf("Stop failed: %v", err)
	}
	if result.Stopped {
		t.Fatal("expected no process to be stopped")
	}
}
