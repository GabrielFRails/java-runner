package simulator

import (
	"net"
	"strconv"
	"testing"
)

func TestIsPortAvailableRejectsInvalidPort(t *testing.T) {
	t.Parallel()

	if _, err := IsPortAvailable(0); err == nil {
		t.Fatal("expected error for invalid port")
	}
}

func TestIsPortAvailableReturnsTrueForFreePort(t *testing.T) {
	t.Parallel()

	port := reserveAndReleasePort(t)

	available, err := IsPortAvailable(port)
	if err != nil {
		t.Fatalf("IsPortAvailable failed: %v", err)
	}
	if !available {
		t.Fatalf("expected port %d to be available", port)
	}
}

func TestIsPortAvailableReturnsFalseForOccupiedPort(t *testing.T) {
	listener, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatalf("failed to reserve test port: %v", err)
	}
	defer listener.Close()

	port := listener.Addr().(*net.TCPAddr).Port
	available, err := IsPortAvailable(port)
	if err != nil {
		t.Fatalf("IsPortAvailable failed: %v", err)
	}
	if available {
		t.Fatalf("expected port %d to be unavailable", port)
	}
}

func TestEnsurePortAvailableReturnsErrorForOccupiedPort(t *testing.T) {
	listener, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatalf("failed to reserve test port: %v", err)
	}
	defer listener.Close()

	port := listener.Addr().(*net.TCPAddr).Port
	if err := EnsurePortAvailable(port); err == nil {
		t.Fatalf("expected occupied port %d to fail", port)
	}
}

func reserveAndReleasePort(t *testing.T) int {
	t.Helper()

	listener, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatalf("failed to reserve test port: %v", err)
	}
	defer listener.Close()

	_, portText, err := net.SplitHostPort(listener.Addr().String())
	if err != nil {
		t.Fatalf("failed to split listener address: %v", err)
	}
	port, err := strconv.Atoi(portText)
	if err != nil {
		t.Fatalf("failed to parse listener port: %v", err)
	}
	return port
}
