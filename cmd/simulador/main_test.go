package main

import (
	"bytes"
	"net"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"testing"

	"github.com/kyriosdata/assinatura/internal/simulator"
	"github.com/kyriosdata/assinatura/internal/storage"
)

func TestRootCommandDefinesLifecycleCommands(t *testing.T) {
	t.Parallel()

	cmd := newRootCommand()

	for _, name := range []string{"start", "stop", "status", "version"} {
		if found, _, err := cmd.Find([]string{name}); err != nil || found == nil || found.Name() != name {
			t.Fatalf("expected command %q to be defined, got command=%v err=%v", name, found, err)
		}
	}
}

func TestVersionCommandPrintsCurrentVersion(t *testing.T) {
	t.Parallel()

	cmd := newRootCommand()
	output := &bytes.Buffer{}
	cmd.SetOut(output)
	cmd.SetArgs([]string{"version"})

	if err := cmd.Execute(); err != nil {
		t.Fatalf("expected version command to run, got error: %v", err)
	}

	got := strings.TrimSpace(output.String())
	if got != "simulador dev" {
		t.Fatalf("expected version output %q, got %q", "simulador dev", got)
	}
}

func TestStartCommandAcceptsPortAndSourceFlags(t *testing.T) {
	execPath, err := os.Executable()
	if err != nil {
		t.Fatalf("failed to locate test executable: %v", err)
	}
	execDir := filepath.Dir(execPath)
	jarPath := filepath.Join(execDir, "simulador.jar")
	if err := os.Remove(jarPath); err != nil && !os.IsNotExist(err) {
		t.Fatalf("failed to remove stale test jar: %v", err)
	}
	t.Cleanup(func() {
		_ = os.Remove(jarPath)
	})

	source := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte("fake simulator jar"))
	}))
	defer source.Close()

	cmd := newRootCommand()
	output := &bytes.Buffer{}
	cmd.SetOut(output)
	cmd.SetErr(output)
	cmd.SetArgs([]string{"start", "--port", "18081", "--source", source.URL + "/simulador.jar"})

	if err := cmd.Execute(); err != nil {
		t.Fatalf("expected start command to accept flags, got error: %v", err)
	}

	got, err := os.ReadFile(filepath.Join(execDir, "simulador.jar"))
	if err != nil {
		t.Fatalf("expected simulador.jar to be downloaded: %v", err)
	}
	if string(got) != "fake simulator jar" {
		t.Fatalf("unexpected downloaded jar content: %q", got)
	}
}

func TestStartCommandRejectsOccupiedPort(t *testing.T) {
	listener, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatalf("failed to reserve test port: %v", err)
	}
	defer listener.Close()

	port := listener.Addr().(*net.TCPAddr).Port
	cmd := newRootCommand()
	output := &bytes.Buffer{}
	cmd.SetOut(output)
	cmd.SetErr(output)
	cmd.SetArgs([]string{"start", "--port", strconv.Itoa(port)})

	if err := cmd.Execute(); err == nil {
		t.Fatal("expected start command to reject occupied port")
	}
}

func TestStopCommandReportsMissingSimulatorProcess(t *testing.T) {
	t.Setenv("HOME", t.TempDir())
	t.Cleanup(storage.Close)

	cmd := newRootCommand()
	output := &bytes.Buffer{}
	cmd.SetOut(output)
	cmd.SetErr(output)
	cmd.SetArgs([]string{"stop", "--port", "18081"})

	if err := cmd.Execute(); err != nil {
		t.Fatalf("expected stop command to run, got error: %v", err)
	}

	got := output.String()
	if !strings.Contains(got, "nenhuma instância gerenciada do simulador.jar encontrada na porta 18081") {
		t.Fatalf("expected missing simulator stop message, got:\n%s", got)
	}
}

func TestStatusCommandReportsMissingSimulatorProcess(t *testing.T) {
	t.Setenv("HOME", t.TempDir())
	t.Cleanup(storage.Close)

	cmd := newRootCommand()
	output := &bytes.Buffer{}
	cmd.SetOut(output)
	cmd.SetErr(output)
	cmd.SetArgs([]string{"status"})

	if err := cmd.Execute(); err != nil {
		t.Fatalf("expected status command to run, got error: %v", err)
	}

	got := output.String()
	if !strings.Contains(got, "Simulador             : não registrado") {
		t.Fatalf("expected missing simulator status, got:\n%s", got)
	}
}

func TestStatusCommandReportsStoredSimulatorProcess(t *testing.T) {
	t.Setenv("HOME", t.TempDir())
	if err := storage.EnsureHomeDir(); err != nil {
		t.Fatalf("EnsureHomeDir failed: %v", err)
	}
	if err := storage.InitDatabase(); err != nil {
		t.Fatalf("InitDatabase failed: %v", err)
	}
	if err := simulator.SaveProcess(4321, 18081, "running"); err != nil {
		t.Fatalf("SaveProcess failed: %v", err)
	}
	storage.Close()
	t.Cleanup(storage.Close)

	cmd := newRootCommand()
	output := &bytes.Buffer{}
	cmd.SetOut(output)
	cmd.SetErr(output)
	cmd.SetArgs([]string{"status"})

	if err := cmd.Execute(); err != nil {
		t.Fatalf("expected status command to run, got error: %v", err)
	}

	got := output.String()
	for _, want := range []string{
		"Simulador             : em execução",
		"PID                   : 4321",
		"Porta                 : 18081",
	} {
		if !strings.Contains(got, want) {
			t.Fatalf("expected status output to contain %q, got:\n%s", want, got)
		}
	}
}
