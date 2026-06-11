package main

import (
	"bytes"
	"crypto/sha256"
	"encoding/hex"
	"net"
	"strconv"
	"strings"
	"testing"

	"github.com/kyriosdata/assinatura/internal/simulator"
	"github.com/kyriosdata/assinatura/internal/storage"
)

func sha256Hex(content string) string {
	sum := sha256.Sum256([]byte(content))
	return hex.EncodeToString(sum[:])
}

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
	restore := ensureArtifactFn
	ensureArtifactFn = func(sourceURL string, expectedChecksum string, cosign simulator.CosignOptions) (*simulator.ArtifactResult, error) {
		if sourceURL != "https://example.test/simulador" {
			t.Fatalf("unexpected source URL: %q", sourceURL)
		}
		if expectedChecksum != sha256Hex("fake simulator artifact") {
			t.Fatalf("unexpected checksum: %q", expectedChecksum)
		}
		if cosign.SignatureURL != "https://example.test/simulador.sig" {
			t.Fatalf("unexpected signature URL: %q", cosign.SignatureURL)
		}
		if cosign.CertificateURL != "https://example.test/simulador.pem" {
			t.Fatalf("unexpected certificate URL: %q", cosign.CertificateURL)
		}
		if cosign.IdentityRegexp == "" || cosign.CertificateOIDCIssuer == "" {
			t.Fatal("expected default Cosign identity and issuer")
		}
		return &simulator.ArtifactResult{
			Path:           "/tmp/simulador-managed",
			Downloaded:     true,
			SourceURL:      sourceURL,
			Checksum:       expectedChecksum,
			CosignVerified: true,
		}, nil
	}
	t.Cleanup(func() {
		ensureArtifactFn = restore
	})

	cmd := newRootCommand()
	output := &bytes.Buffer{}
	cmd.SetOut(output)
	cmd.SetErr(output)
	cmd.SetArgs([]string{
		"start",
		"--port", "18081",
		"--source", "https://example.test/simulador",
		"--checksum", sha256Hex("fake simulator artifact"),
		"--cosign-signature", "https://example.test/simulador.sig",
		"--cosign-certificate", "https://example.test/simulador.pem",
	})

	if err := cmd.Execute(); err != nil {
		t.Fatalf("expected start command to accept flags, got error: %v", err)
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
	if !strings.Contains(got, "nenhuma instância gerenciada do simulador encontrada na porta 18081") {
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
