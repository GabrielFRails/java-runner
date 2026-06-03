package simulator

import (
	"testing"

	"github.com/kyriosdata/assinatura/internal/storage"
)

func setupStorage(t *testing.T) {
	t.Helper()

	t.Setenv("HOME", t.TempDir())
	if err := storage.EnsureHomeDir(); err != nil {
		t.Fatalf("EnsureHomeDir failed: %v", err)
	}
	if err := storage.InitDatabase(); err != nil {
		t.Fatalf("InitDatabase failed: %v", err)
	}
	t.Cleanup(storage.Close)
}

func TestSaveAndGetSimulatorProcess(t *testing.T) {
	setupStorage(t)

	if err := SaveProcess(4321, 8081, "running"); err != nil {
		t.Fatalf("SaveProcess failed: %v", err)
	}

	process, err := GetProcess()
	if err != nil {
		t.Fatalf("GetProcess failed: %v", err)
	}
	if process == nil {
		t.Fatal("expected simulator process")
	}
	if process.Name != ProcessName {
		t.Fatalf("expected process name %q, got %q", ProcessName, process.Name)
	}
	if process.PID != 4321 || process.Port != 8081 || process.Status != "running" {
		t.Fatalf("unexpected process: %+v", process)
	}
}

func TestMarkStoppedKeepsPidAndPort(t *testing.T) {
	setupStorage(t)

	process := storage.Process{
		Name:   ProcessName,
		PID:    4321,
		Port:   8081,
		Status: "running",
	}
	if err := MarkStopped(process); err != nil {
		t.Fatalf("MarkStopped failed: %v", err)
	}

	got, err := GetProcess()
	if err != nil {
		t.Fatalf("GetProcess failed: %v", err)
	}
	if got == nil {
		t.Fatal("expected simulator process")
	}
	if got.PID != process.PID || got.Port != process.Port || got.Status != "stopped" {
		t.Fatalf("unexpected stopped process: %+v", got)
	}
}
