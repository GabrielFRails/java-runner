package simulator

import (
	"errors"
	"testing"
)

func withTerminateProcessStub(t *testing.T, fn func(pid int) error) {
	t.Helper()

	restore := terminateProcessFn
	terminateProcessFn = fn
	t.Cleanup(func() {
		terminateProcessFn = restore
	})
}

func TestStopRejectsInvalidPort(t *testing.T) {
	t.Parallel()

	if _, err := Stop(0); err == nil {
		t.Fatal("expected error for invalid port")
	}
}

func TestStopReturnsNotStoppedWhenSimulatorProcessIsMissing(t *testing.T) {
	setupStorage(t)

	result, err := Stop(8081)
	if err != nil {
		t.Fatalf("Stop failed: %v", err)
	}
	if result.Stopped {
		t.Fatal("expected no process to be stopped")
	}
}

func TestStopReturnsNotStoppedWhenPortDoesNotMatch(t *testing.T) {
	setupStorage(t)

	if err := SaveProcess(4321, 8081, "running"); err != nil {
		t.Fatalf("SaveProcess failed: %v", err)
	}

	result, err := Stop(18081)
	if err != nil {
		t.Fatalf("Stop failed: %v", err)
	}
	if result.Stopped {
		t.Fatal("expected process on another port not to be stopped")
	}
}

func TestStopTerminatesAndMarksSimulatorStopped(t *testing.T) {
	setupStorage(t)

	var terminatedPID int
	withTerminateProcessStub(t, func(pid int) error {
		terminatedPID = pid
		return nil
	})

	if err := SaveProcess(4321, 8081, "running"); err != nil {
		t.Fatalf("SaveProcess failed: %v", err)
	}

	result, err := Stop(8081)
	if err != nil {
		t.Fatalf("Stop failed: %v", err)
	}
	if !result.Stopped {
		t.Fatal("expected process to be stopped")
	}
	if terminatedPID != 4321 {
		t.Fatalf("expected PID 4321 to be terminated, got %d", terminatedPID)
	}

	process, err := GetProcess()
	if err != nil {
		t.Fatalf("GetProcess failed: %v", err)
	}
	if process == nil || process.Status != "stopped" {
		t.Fatalf("expected stopped process, got %+v", process)
	}
}

func TestStopReturnsTerminateError(t *testing.T) {
	setupStorage(t)

	withTerminateProcessStub(t, func(pid int) error {
		return errors.New("boom")
	})

	if err := SaveProcess(4321, 8081, "running"); err != nil {
		t.Fatalf("SaveProcess failed: %v", err)
	}

	if _, err := Stop(8081); err == nil {
		t.Fatal("expected terminate error")
	}
}
