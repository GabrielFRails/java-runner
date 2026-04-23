package jar

import (
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"testing"
)

func TestBuildSignArgsMapsFlags(t *testing.T) {
	t.Parallel()

	args := buildSignArgs("/tmp/assinador.jar", SignFlags{
		Bundle:         "bundle.json",
		Provenance:     "provenance.json",
		Timestamp:      123,
		Strategy:       "iat",
		PolicyUri:      "https://example.com/policy|0.1.2",
		CertChain:      "chain.json",
		CryptoType:     "PEM",
		CryptoPem:      "key.pem",
		CryptoPassword: "secret",
		CryptoPkcs12:   "cert.p12",
		CryptoAlias:    "alias",
		Config:         "config.json",
	})

	got := strings.Join(args, " ")

	for _, expected := range []string{
		"-jar /tmp/assinador.jar sign",
		"--bundle bundle.json",
		"--provenance provenance.json",
		"--timestamp 123",
		"--strategy iat",
		"--policy https://example.com/policy|0.1.2",
		"--cert chain.json",
		"--crypto-type PEM",
		"--crypto-pem key.pem",
		"--crypto-password secret",
		"--crypto-pkcs12 cert.p12",
		"--crypto-alias alias",
		"--config config.json",
	} {
		if !strings.Contains(got, expected) {
			t.Fatalf("expected args to contain %q, got %q", expected, got)
		}
	}
}

func TestBuildValidateArgsMapsFlags(t *testing.T) {
	t.Parallel()

	args := buildValidateArgs("/tmp/assinador.jar", ValidateFlags{
		Signature: "signature.txt",
		Timestamp: 456,
		PolicyUri: "https://example.com/policy|0.1.2",
		Config:    "config.json",
	})

	got := strings.Join(args, " ")

	for _, expected := range []string{
		"-jar /tmp/assinador.jar validate",
		"--signature signature.txt",
		"--timestamp 456",
		"--policy https://example.com/policy|0.1.2",
		"--config config.json",
	} {
		if !strings.Contains(got, expected) {
			t.Fatalf("expected args to contain %q, got %q", expected, got)
		}
	}
}

func TestInvokeSignCapturesStdoutAndExitCode(t *testing.T) {
	t.Parallel()

	javaPath := createFakeJava(t, "#!/bin/sh\nprintf 'SIGN_OK'\n")

	result, err := InvokeSign(javaPath, "/tmp/assinador.jar", SignFlags{
		Bundle:     "bundle.json",
		Provenance: "provenance.json",
		PolicyUri:  "https://example.com/policy|0.1.2",
		CertChain:  "chain.json",
		CryptoType: "PEM",
		CryptoPem:  "key.pem",
		Config:     "config.json",
	})
	if err != nil {
		t.Fatalf("expected no error invoking sign, got %v", err)
	}

	if result.ExitCode != 0 {
		t.Fatalf("expected exit code 0, got %d", result.ExitCode)
	}
	if result.Output != "SIGN_OK" {
		t.Fatalf("expected output SIGN_OK, got %q", result.Output)
	}
}

func TestInvokeValidateUsesStderrWhenProcessFails(t *testing.T) {
	t.Parallel()

	javaPath := createFakeJava(t, "#!/bin/sh\nprintf 'VALIDATION_ERROR' 1>&2\nexit 7\n")

	result, err := InvokeValidate(javaPath, "/tmp/assinador.jar", ValidateFlags{
		Signature: "signature.txt",
		PolicyUri: "https://example.com/policy|0.1.2",
		Config:    "config.json",
	})
	if err != nil {
		t.Fatalf("expected no error for process exit, got %v", err)
	}

	if result.ExitCode != 7 {
		t.Fatalf("expected exit code 7, got %d", result.ExitCode)
	}
	if result.Output != "VALIDATION_ERROR" {
		t.Fatalf("expected stderr output fallback, got %q", result.Output)
	}
}

func createFakeJava(t *testing.T, script string) string {
	t.Helper()

	dir := t.TempDir()
	name := "java"
	if runtime.GOOS == "windows" {
		name = "java.bat"
		script = "@echo off\r\n" + script
	}

	path := filepath.Join(dir, name)
	if err := os.WriteFile(path, []byte(script), 0o755); err != nil {
		t.Fatalf("failed to create fake java executable: %v", err)
	}

	return path
}
