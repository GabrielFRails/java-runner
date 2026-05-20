package httpclient

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"strings"
	"time"

	"github.com/kyriosdata/assinatura/internal/jar"
)

type Result struct {
	Output     string
	StatusCode int
}

type cryptoMaterial struct {
	Type          string `json:"type,omitempty"`
	PrivateKeyPEM string `json:"privateKeyPem,omitempty"`
	Password      string `json:"password,omitempty"`
	PKCS12Base64  string `json:"pkcs12Base64,omitempty"`
	Alias         string `json:"alias,omitempty"`
}

type signRequest struct {
	Bundle             string          `json:"bundle"`
	Provenance         string          `json:"provenance"`
	CryptoMaterial     cryptoMaterial  `json:"cryptoMaterial"`
	CertificateChain   json.RawMessage `json:"certificateChain"`
	ReferenceTimestamp int64           `json:"referenceTimestamp"`
	Strategy           string          `json:"strategy"`
	PolicyURI          string          `json:"policyUri"`
	Config             json.RawMessage `json:"config"`
}

type validateRequest struct {
	SignatureData      string          `json:"signatureData"`
	ReferenceTimestamp int64           `json:"referenceTimestamp"`
	PolicyURI          string          `json:"policyUri"`
	Config             json.RawMessage `json:"config"`
}

func Sign(port int, flags jar.SignFlags) (*Result, error) {
	payload, err := buildSignRequest(flags)
	if err != nil {
		return nil, err
	}

	return postJSON(port, "/sign", payload)
}

func Validate(port int, flags jar.ValidateFlags) (*Result, error) {
	payload, err := buildValidateRequest(flags)
	if err != nil {
		return nil, err
	}

	return postJSON(port, "/validate", payload)
}

func buildSignRequest(flags jar.SignFlags) (signRequest, error) {
	bundle, err := readTrimmedFile(flags.Bundle)
	if err != nil {
		return signRequest{}, err
	}
	provenance, err := readTrimmedFile(flags.Provenance)
	if err != nil {
		return signRequest{}, err
	}
	certChain, err := readRawJSONFile(flags.CertChain)
	if err != nil {
		return signRequest{}, err
	}
	config, err := readRawJSONFile(flags.Config)
	if err != nil {
		return signRequest{}, err
	}

	crypto, err := buildCryptoMaterial(flags)
	if err != nil {
		return signRequest{}, err
	}

	return signRequest{
		Bundle:             bundle,
		Provenance:         provenance,
		CryptoMaterial:     crypto,
		CertificateChain:   certChain,
		ReferenceTimestamp: flags.Timestamp,
		Strategy:           flags.Strategy,
		PolicyURI:          flags.PolicyUri,
		Config:             config,
	}, nil
}

func buildValidateRequest(flags jar.ValidateFlags) (validateRequest, error) {
	signature, err := readTrimmedFile(flags.Signature)
	if err != nil {
		return validateRequest{}, err
	}
	config, err := readRawJSONFile(flags.Config)
	if err != nil {
		return validateRequest{}, err
	}

	return validateRequest{
		SignatureData:      signature,
		ReferenceTimestamp: flags.Timestamp,
		PolicyURI:          flags.PolicyUri,
		Config:             config,
	}, nil
}

func buildCryptoMaterial(flags jar.SignFlags) (cryptoMaterial, error) {
	material := cryptoMaterial{
		Type:     flags.CryptoType,
		Password: flags.CryptoPassword,
		Alias:    flags.CryptoAlias,
	}
	if material.Type == "" {
		material.Type = "PEM"
	}

	if flags.CryptoPem != "" {
		value, err := readTrimmedFile(flags.CryptoPem)
		if err != nil {
			return material, err
		}
		material.PrivateKeyPEM = value
	}
	if flags.CryptoPkcs12 != "" {
		value, err := readTrimmedFile(flags.CryptoPkcs12)
		if err != nil {
			return material, err
		}
		material.PKCS12Base64 = value
	}

	return material, nil
}

func postJSON(port int, path string, payload any) (*Result, error) {
	body, err := json.Marshal(payload)
	if err != nil {
		return nil, fmt.Errorf("erro ao montar requisição HTTP: %w", err)
	}

	url := fmt.Sprintf("http://127.0.0.1:%d%s", port, path)
	client := &http.Client{Timeout: 30 * time.Second}
	resp, err := client.Post(url, "application/json", bytes.NewReader(body))
	if err != nil {
		return nil, fmt.Errorf("erro ao chamar assinador.jar via HTTP: %w", err)
	}
	defer resp.Body.Close()

	output, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("erro ao ler resposta HTTP: %w", err)
	}

	return &Result{
		Output:     strings.TrimSpace(string(output)),
		StatusCode: resp.StatusCode,
	}, nil
}

func readTrimmedFile(path string) (string, error) {
	content, err := os.ReadFile(path)
	if err != nil {
		return "", fmt.Errorf("não foi possível ler %s: %w", path, err)
	}
	return strings.TrimSpace(string(content)), nil
}

func readRawJSONFile(path string) (json.RawMessage, error) {
	content, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("não foi possível ler %s: %w", path, err)
	}
	trimmed := bytes.TrimSpace(content)
	if !json.Valid(trimmed) {
		return nil, fmt.Errorf("%s não contém JSON válido", path)
	}
	return json.RawMessage(trimmed), nil
}
