package main

import (
	"testing"

	"github.com/kyriosdata/assinatura/internal/jar"
)

func TestValidateSignFlagsRequiresCoreFlags(t *testing.T) {
	t.Parallel()

	err := validateSignFlags(jar.SignFlags{})
	if err == nil {
		t.Fatal("expected error for missing sign flags")
	}
}

func TestValidateSignFlagsAcceptsValidPemFlow(t *testing.T) {
	t.Parallel()

	err := validateSignFlags(jar.SignFlags{
		Bundle:     "bundle.json",
		Provenance: "provenance.json",
		PolicyUri:  "https://example.com/policy|0.1.2",
		CertChain:  "chain.json",
		Config:     "config.json",
		CryptoType: "PEM",
		CryptoPem:  "key.pem",
	})
	if err != nil {
		t.Fatalf("expected valid PEM flags, got error: %v", err)
	}
}

func TestValidateSignFlagsRequiresPkcs12Fields(t *testing.T) {
	t.Parallel()

	err := validateSignFlags(jar.SignFlags{
		Bundle:       "bundle.json",
		Provenance:   "provenance.json",
		PolicyUri:    "https://example.com/policy|0.1.2",
		CertChain:    "chain.json",
		Config:       "config.json",
		CryptoType:   "PKCS12",
		CryptoPkcs12: "cert.p12",
	})
	if err == nil {
		t.Fatal("expected error for incomplete PKCS12 flags")
	}
}

func TestValidateValidateFlagsRequiresCoreFlags(t *testing.T) {
	t.Parallel()

	err := validateValidateFlags(jar.ValidateFlags{})
	if err == nil {
		t.Fatal("expected error for missing validate flags")
	}
}

func TestValidateValidateFlagsAcceptsValidFlow(t *testing.T) {
	t.Parallel()

	err := validateValidateFlags(jar.ValidateFlags{
		Signature: "signature.txt",
		PolicyUri: "https://example.com/policy|0.1.2",
		Config:    "config.json",
	})
	if err != nil {
		t.Fatalf("expected valid validate flags, got error: %v", err)
	}
}
