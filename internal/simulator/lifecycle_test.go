package simulator

import (
	"net"
	"testing"
)

func TestLifecycleStatusWithoutProcess(t *testing.T) {
	setupStorage(t)

	process, err := GetProcess()
	if err != nil {
		t.Fatalf("GetProcess failed: %v", err)
	}
	if process != nil {
		t.Fatalf("expected no simulator process, got %+v", process)
	}
}

func TestLifecycleRegistersSimulatorProcess(t *testing.T) {
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
		t.Fatalf("unexpected simulator process: %+v", process)
	}
}

func TestLifecycleStopWithoutProcessDoesNotTerminateAnything(t *testing.T) {
	setupStorage(t)

	terminateCalled := false
	withTerminateProcessStub(t, func(pid int) error {
		terminateCalled = true
		return nil
	})

	result, err := Stop(8081)
	if err != nil {
		t.Fatalf("Stop failed: %v", err)
	}
	if result.Stopped {
		t.Fatal("expected missing process not to be stopped")
	}
	if terminateCalled {
		t.Fatal("expected terminate process not to be called")
	}
}

func TestLifecycleValidatesPortAvailability(t *testing.T) {
	freePort := reserveAndReleasePort(t)
	if err := EnsurePortAvailable(freePort); err != nil {
		t.Fatalf("expected free port %d to be available: %v", freePort, err)
	}

	listener, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatalf("failed to reserve test port: %v", err)
	}
	defer listener.Close()

	occupiedPort := listener.Addr().(*net.TCPAddr).Port
	if err := EnsurePortAvailable(occupiedPort); err == nil {
		t.Fatalf("expected occupied port %d to be rejected", occupiedPort)
	}
}
