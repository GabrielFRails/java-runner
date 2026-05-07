package server

import "testing"

func TestStartRejectsInvalidPort(t *testing.T) {
	t.Parallel()

	if _, err := Start("java", "assinador.jar", 0); err == nil {
		t.Fatal("expected error for invalid port")
	}
}

func TestIsHealthyRejectsInvalidPort(t *testing.T) {
	t.Parallel()

	if IsHealthy(0) {
		t.Fatal("expected invalid port to be unhealthy")
	}
}
