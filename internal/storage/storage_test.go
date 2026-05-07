package storage

import (
	"testing"
)

func TestSaveAndGetProcess(t *testing.T) {
	t.Setenv("HOME", t.TempDir())

	if err := EnsureHomeDir(); err != nil {
		t.Fatalf("EnsureHomeDir failed: %v", err)
	}
	if err := InitDatabase(); err != nil {
		t.Fatalf("InitDatabase failed: %v", err)
	}
	defer Close()

	err := SaveProcess(Process{
		Name:   "assinador",
		PID:    1234,
		Port:   8080,
		Status: "running",
	})
	if err != nil {
		t.Fatalf("SaveProcess failed: %v", err)
	}

	process, err := GetProcess("assinador")
	if err != nil {
		t.Fatalf("GetProcess failed: %v", err)
	}
	if process == nil {
		t.Fatal("expected process")
	}
	if process.PID != 1234 || process.Port != 8080 || process.Status != "running" {
		t.Fatalf("unexpected process: %+v", process)
	}
}

func TestGetProcessReturnsNilWhenMissing(t *testing.T) {
	t.Setenv("HOME", t.TempDir())

	if err := EnsureHomeDir(); err != nil {
		t.Fatalf("EnsureHomeDir failed: %v", err)
	}
	if err := InitDatabase(); err != nil {
		t.Fatalf("InitDatabase failed: %v", err)
	}
	defer Close()

	process, err := GetProcess("assinador")
	if err != nil {
		t.Fatalf("GetProcess failed: %v", err)
	}
	if process != nil {
		t.Fatalf("expected nil process, got %+v", process)
	}
}
