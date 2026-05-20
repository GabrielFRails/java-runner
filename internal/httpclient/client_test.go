package httpclient

import (
	"encoding/json"
	"os"
	"path/filepath"
	"testing"

	"github.com/kyriosdata/assinatura/internal/jar"
)

func TestBuildSignRequestFromFlags(t *testing.T) {
	dir := t.TempDir()

	payload, err := buildSignRequest(jar.SignFlags{
		Bundle:     writeFile(t, dir, "bundle.json", `{"resourceType":"Bundle"}`),
		Provenance: writeFile(t, dir, "provenance.json", `{"resourceType":"Provenance"}`),
		Timestamp:  123,
		Strategy:   "iat",
		PolicyUri:  "policy|0.1.2",
		CertChain:  writeFile(t, dir, "chain.json", `["MAMA","MAMA"]`),
		CryptoType: "PEM",
		CryptoPem:  writeFile(t, dir, "key.pem", "PRIVATE"),
		Config:     writeFile(t, dir, "config.json", `{"maxRetries":3}`),
	})
	if err != nil {
		t.Fatalf("buildSignRequest failed: %v", err)
	}

	if payload.Bundle != `{"resourceType":"Bundle"}` {
		t.Fatalf("unexpected bundle: %s", payload.Bundle)
	}
	if payload.CryptoMaterial.PrivateKeyPEM != "PRIVATE" {
		t.Fatalf("unexpected privateKeyPem: %s", payload.CryptoMaterial.PrivateKeyPEM)
	}

	var chain []string
	if err := json.Unmarshal(payload.CertificateChain, &chain); err != nil {
		t.Fatalf("certificateChain is invalid JSON: %v", err)
	}
	if len(chain) != 2 {
		t.Fatalf("unexpected certificateChain length: %d", len(chain))
	}
}

func TestBuildValidateRequestFromFlags(t *testing.T) {
	dir := t.TempDir()

	payload, err := buildValidateRequest(jar.ValidateFlags{
		Signature: writeFile(t, dir, "signature.txt", "U0lNVUxBVEVEX1NJR05BVFVSRQ=="),
		Timestamp: 123,
		PolicyUri: "policy|0.1.2",
		Config:    writeFile(t, dir, "config.json", `{"maxRetries":3}`),
	})
	if err != nil {
		t.Fatalf("buildValidateRequest failed: %v", err)
	}

	if payload.SignatureData != "U0lNVUxBVEVEX1NJR05BVFVSRQ==" {
		t.Fatalf("unexpected signatureData: %s", payload.SignatureData)
	}
	if payload.PolicyURI != "policy|0.1.2" {
		t.Fatalf("unexpected policyUri: %s", payload.PolicyURI)
	}
}

func writeFile(t *testing.T, dir, name, content string) string {
	t.Helper()

	path := filepath.Join(dir, name)
	if err := os.WriteFile(path, []byte(content), 0o644); err != nil {
		t.Fatalf("WriteFile failed: %v", err)
	}
	return path
}
