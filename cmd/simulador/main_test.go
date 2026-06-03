package main

import (
	"bytes"
	"strings"
	"testing"
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
	t.Parallel()

	cmd := newRootCommand()
	output := &bytes.Buffer{}
	cmd.SetOut(output)
	cmd.SetErr(output)
	cmd.SetArgs([]string{"start", "--port", "18081", "--source", "https://example.com/simulador.jar"})

	if err := cmd.Execute(); err != nil {
		t.Fatalf("expected start command to accept flags, got error: %v", err)
	}
}
