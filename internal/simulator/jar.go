package simulator

import (
	"bufio"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"hash"
	"io"
	"net/http"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"time"
)

const defaultLatestReleaseURL = "https://api.github.com/repos/GabrielFRails/java-runner/releases/latest" // deixar mocado mesmo já que o repo é meu hehe
const simulatorReleaseAssetOS = "darwin"                                                                 // [TODO] detectar com runtime.GOOS.
const simulatorReleaseAssetArch = "arm64"                                                                // [TODO] detectar com runtime.GOARCH.
const localSimulatorArtifactName = "simulador-managed"

type ArtifactResult struct {
	Path           string
	Downloaded     bool
	SourceURL      string
	Version        string
	Checksum       string
	CosignVerified bool
}

type CosignOptions struct {
	SignatureURL          string
	CertificateURL        string
	IdentityRegexp        string
	CertificateOIDCIssuer string
}

var executablePathFn = os.Executable
var artifactHTTPClient = &http.Client{Timeout: 2 * time.Minute}
var latestReleaseURL = defaultLatestReleaseURL
var verifyCosignBlobFn = runCosignVerifyBlob

type githubRelease struct {
	TagName string `json:"tag_name"`
	Assets  []struct {
		Name               string `json:"name"`
		BrowserDownloadURL string `json:"browser_download_url"`
		Digest             string `json:"digest"`
	} `json:"assets"`
}

func LocateArtifact() (string, error) {
	execPath, err := executablePathFn()
	if err != nil {
		return "", fmt.Errorf("não foi possível localizar o executável simulador: %w", err)
	}

	return filepath.Join(filepath.Dir(execPath), localSimulatorArtifactName), nil
}

func EnsureArtifact(sourceURL string, expectedChecksum string, cosign CosignOptions) (*ArtifactResult, error) {
	artifactPath, err := LocateArtifact()
	if err != nil {
		return nil, err
	}

	expectedChecksum, err = normalizeSHA256(expectedChecksum)
	if err != nil {
		return nil, err
	}

	if _, err := os.Stat(artifactPath); err == nil {
		if expectedChecksum != "" {
			if err := verifyFileSHA256(artifactPath, expectedChecksum); err != nil {
				return nil, err
			}
		}
		cosignVerified := false
		if hasCosignMetadata(cosign) {
			if err := verifyCosignSignature(artifactPath, cosign); err != nil {
				return nil, err
			}
			cosignVerified = true
		}
		return &ArtifactResult{Path: artifactPath, Checksum: expectedChecksum, CosignVerified: cosignVerified}, nil
	} else if !os.IsNotExist(err) {
		return nil, fmt.Errorf("não foi possível verificar artefato local do simulador: %w", err)
	}

	if sourceURL == "" {
		assetURL, version, releaseChecksum, releaseCosign, err := latestSimulatorReleaseAsset(expectedChecksum == "")
		if err != nil {
			return nil, err
		}
		sourceURL = assetURL
		if expectedChecksum == "" {
			expectedChecksum = releaseChecksum
		}
		if cosign.SignatureURL == "" {
			cosign.SignatureURL = releaseCosign.SignatureURL
		}
		if cosign.CertificateURL == "" {
			cosign.CertificateURL = releaseCosign.CertificateURL
		}
		if expectedChecksum == "" {
			return nil, fmt.Errorf("checksum SHA-256 esperado não encontrado para artefato do simulador na release mais recente")
		}

		if err := downloadArtifact(sourceURL, artifactPath, expectedChecksum, cosign); err != nil {
			return nil, err
		}

		return &ArtifactResult{
			Path: artifactPath, Downloaded: true, SourceURL: sourceURL, Version: version, Checksum: expectedChecksum, CosignVerified: true,
		}, nil
	}

	if expectedChecksum == "" {
		return nil, fmt.Errorf("checksum SHA-256 esperado é obrigatório para baixar artefato do simulador via --source")
	}

	if err := downloadArtifact(sourceURL, artifactPath, expectedChecksum, cosign); err != nil {
		return nil, err
	}

	return &ArtifactResult{Path: artifactPath, Downloaded: true, SourceURL: sourceURL, Checksum: expectedChecksum, CosignVerified: true}, nil
}

func downloadArtifact(sourceURL string, dest string, expectedChecksum string, cosign CosignOptions) error {
	resp, err := getURL(sourceURL)
	if err != nil {
		return fmt.Errorf("erro ao baixar artefato do simulador: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode < http.StatusOK || resp.StatusCode >= http.StatusMultipleChoices {
		return fmt.Errorf("download do artefato do simulador retornou status %d", resp.StatusCode)
	}

	if err := os.MkdirAll(filepath.Dir(dest), 0o755); err != nil {
		return fmt.Errorf("não foi possível criar diretório para artefato do simulador: %w", err)
	}

	tmp, err := os.CreateTemp(filepath.Dir(dest), localSimulatorArtifactName+".*.tmp")
	if err != nil {
		return fmt.Errorf("não foi possível criar arquivo temporário para artefato do simulador: %w", err)
	}
	tmpPath := tmp.Name()
	defer os.Remove(tmpPath)

	hasher := sha256.New()
	if _, err := io.Copy(io.MultiWriter(tmp, hasher), resp.Body); err != nil {
		tmp.Close()
		return fmt.Errorf("não foi possível salvar download do artefato do simulador: %w", err)
	}
	if err := tmp.Close(); err != nil {
		return fmt.Errorf("não foi possível fechar download do artefato do simulador: %w", err)
	}

	if err := verifySHA256(hasher, expectedChecksum); err != nil {
		return err
	}
	if err := verifyCosignSignature(tmpPath, cosign); err != nil {
		return err
	}

	if err := os.Rename(tmpPath, dest); err != nil {
		return fmt.Errorf("não foi possível instalar artefato do simulador: %w", err)
	}
	if err := os.Chmod(dest, 0o755); err != nil {
		return fmt.Errorf("não foi possível tornar artefato do simulador executável: %w", err)
	}

	return nil
}

func latestSimulatorReleaseAsset(requireChecksum bool) (string, string, string, CosignOptions, error) {
	resp, err := getURL(latestReleaseURL)
	if err != nil {
		return "", "", "", CosignOptions{}, fmt.Errorf("erro ao consultar GitHub Releases: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode == http.StatusNotFound {
		return "", "", "", CosignOptions{}, fmt.Errorf("release mais recente não encontrada no GitHub")
	}
	if resp.StatusCode < http.StatusOK || resp.StatusCode >= http.StatusMultipleChoices {
		return "", "", "", CosignOptions{}, fmt.Errorf("consulta ao GitHub Releases retornou status %d", resp.StatusCode)
	}

	var release githubRelease
	if err := json.NewDecoder(resp.Body).Decode(&release); err != nil {
		return "", "", "", CosignOptions{}, fmt.Errorf("resposta inválida do GitHub Releases: %w", err)
	}

	assetName := simulatorReleaseAssetName(release.TagName)
	var assetURL, digest, checksumURL string
	releaseCosign := CosignOptions{}
	for _, asset := range release.Assets {
		if asset.Name == assetName && asset.BrowserDownloadURL != "" {
			assetURL = asset.BrowserDownloadURL
			digest = asset.Digest
		}
		if asset.Name == assetName+".sig" && asset.BrowserDownloadURL != "" {
			releaseCosign.SignatureURL = asset.BrowserDownloadURL
		}
		if asset.Name == assetName+".pem" && asset.BrowserDownloadURL != "" {
			releaseCosign.CertificateURL = asset.BrowserDownloadURL
		}
		if isChecksumAssetName(asset.Name) && asset.BrowserDownloadURL != "" {
			checksumURL = asset.BrowserDownloadURL
		}
	}

	if assetURL != "" {
		if !requireChecksum {
			return assetURL, release.TagName, "", releaseCosign, nil
		}
		checksum, err := checksumFromRelease(assetName, digest, checksumURL)
		if err != nil {
			return "", "", "", CosignOptions{}, err
		}
		return assetURL, release.TagName, checksum, releaseCosign, nil
	}

	if release.TagName == "" {
		return "", "", "", CosignOptions{}, fmt.Errorf("asset do simulador não encontrado na release mais recente")
	}
	return "", "", "", CosignOptions{}, fmt.Errorf("asset %s não encontrado na release %s", assetName, release.TagName)
}

func simulatorReleaseAssetName(tagName string) string {
	return fmt.Sprintf(
		"simulador-%s-%s-%s",
		tagName,
		simulatorReleaseAssetOS,
		simulatorReleaseAssetArch,
	)
}

func getURL(rawURL string) (*http.Response, error) {
	req, err := http.NewRequest(http.MethodGet, rawURL, nil)
	if err != nil {
		return nil, err
	}
	req.Header.Set("Accept", "application/vnd.github+json")
	req.Header.Set("User-Agent", "hubsaude-simulador-cli")
	return artifactHTTPClient.Do(req)
}

func checksumFromRelease(assetName string, digest string, checksumURL string) (string, error) {
	if checksum, err := normalizeSHA256(digest); err != nil {
		return "", err
	} else if checksum != "" {
		return checksum, nil
	}

	if checksumURL == "" {
		return "", fmt.Errorf("checksum SHA-256 não encontrado para %s na release mais recente", assetName)
	}

	checksum, err := downloadChecksum(checksumURL, assetName)
	if err != nil {
		return "", err
	}
	return checksum, nil
}

func downloadChecksum(sourceURL string, artifactName string) (string, error) {
	resp, err := getURL(sourceURL)
	if err != nil {
		return "", fmt.Errorf("erro ao baixar checksum do artefato do simulador: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode < http.StatusOK || resp.StatusCode >= http.StatusMultipleChoices {
		return "", fmt.Errorf("download do checksum do simulador retornou status %d", resp.StatusCode)
	}

	scanner := bufio.NewScanner(resp.Body)
	for scanner.Scan() {
		checksum, name, ok := parseChecksumLine(scanner.Text())
		if !ok {
			continue
		}
		if name == "" || name == artifactName || filepath.Base(name) == artifactName {
			return checksum, nil
		}
	}
	if err := scanner.Err(); err != nil {
		return "", fmt.Errorf("não foi possível ler checksum do simulador: %w", err)
	}

	return "", fmt.Errorf("checksum SHA-256 de %s não encontrado no arquivo de checksums", artifactName)
}

func parseChecksumLine(line string) (string, string, bool) {
	fields := strings.Fields(line)
	if len(fields) == 0 {
		return "", "", false
	}

	checksum, err := normalizeSHA256(fields[0])
	if err != nil || checksum == "" {
		return "", "", false
	}

	if len(fields) == 1 {
		return checksum, "", true
	}
	name := strings.TrimPrefix(fields[1], "*")
	return checksum, name, true
}

func isChecksumAssetName(name string) bool {
	lower := strings.ToLower(name)
	return lower == "checksums.txt" ||
		lower == "sha256sums.txt" ||
		strings.HasSuffix(lower, ".sha256") ||
		strings.HasSuffix(lower, ".sha256sum") ||
		strings.HasSuffix(lower, ".sha256sums")
}

func verifyFileSHA256(path string, expectedChecksum string) error {
	file, err := os.Open(path)
	if err != nil {
		return fmt.Errorf("não foi possível abrir artefato do simulador para checksum: %w", err)
	}
	defer file.Close()

	hasher := sha256.New()
	if _, err := io.Copy(hasher, file); err != nil {
		return fmt.Errorf("não foi possível calcular checksum do artefato do simulador: %w", err)
	}
	return verifySHA256(hasher, expectedChecksum)
}

func verifySHA256(hasher hash.Hash, expectedChecksum string) error {
	actual := hex.EncodeToString(hasher.Sum(nil))
	if actual != expectedChecksum {
		return fmt.Errorf("checksum SHA-256 do artefato do simulador não confere: esperado %s, obtido %s", expectedChecksum, actual)
	}
	return nil
}

func verifyCosignSignature(artifactPath string, options CosignOptions) error {
	if options.SignatureURL == "" {
		return fmt.Errorf("assinatura Cosign do artefato do simulador não encontrada")
	}
	if options.CertificateURL == "" {
		return fmt.Errorf("certificado Cosign do artefato do simulador não encontrado")
	}
	if options.IdentityRegexp == "" {
		return fmt.Errorf("identidade esperada do certificado Cosign não informada")
	}
	if options.CertificateOIDCIssuer == "" {
		return fmt.Errorf("emissor OIDC esperado do certificado Cosign não informado")
	}

	signaturePath, err := downloadSidecar(options.SignatureURL, "simulador-cosign-*.sig")
	if err != nil {
		return err
	}
	defer os.Remove(signaturePath)

	certificatePath, err := downloadSidecar(options.CertificateURL, "simulador-cosign-*.pem")
	if err != nil {
		return err
	}
	defer os.Remove(certificatePath)

	if err := verifyCosignBlobFn(artifactPath, signaturePath, certificatePath, options); err != nil {
		return err
	}
	return nil
}

func runCosignVerifyBlob(artifactPath string, signaturePath string, certificatePath string, options CosignOptions) error {
	cmd := exec.Command(
		"cosign",
		"verify-blob",
		"--certificate", certificatePath,
		"--signature", signaturePath,
		"--certificate-identity-regexp", options.IdentityRegexp,
		"--certificate-oidc-issuer", options.CertificateOIDCIssuer,
		artifactPath,
	)
	output, err := cmd.CombinedOutput()
	if err != nil {
		detail := strings.TrimSpace(string(output))
		if detail == "" {
			detail = err.Error()
		}
		return fmt.Errorf("assinatura Cosign do artefato do simulador não confere: %s", detail)
	}
	return nil
}

func hasCosignMetadata(options CosignOptions) bool {
	return options.SignatureURL != "" || options.CertificateURL != ""
}

func downloadSidecar(sourceURL string, pattern string) (string, error) {
	resp, err := getURL(sourceURL)
	if err != nil {
		return "", fmt.Errorf("erro ao baixar metadado Cosign do simulador: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode < http.StatusOK || resp.StatusCode >= http.StatusMultipleChoices {
		return "", fmt.Errorf("download do metadado Cosign do simulador retornou status %d", resp.StatusCode)
	}

	tmp, err := os.CreateTemp("", pattern)
	if err != nil {
		return "", fmt.Errorf("não foi possível criar arquivo temporário para metadado Cosign: %w", err)
	}
	tmpPath := tmp.Name()

	if _, err := io.Copy(tmp, resp.Body); err != nil {
		tmp.Close()
		os.Remove(tmpPath)
		return "", fmt.Errorf("não foi possível salvar metadado Cosign do simulador: %w", err)
	}
	if err := tmp.Close(); err != nil {
		os.Remove(tmpPath)
		return "", fmt.Errorf("não foi possível fechar metadado Cosign do simulador: %w", err)
	}

	return tmpPath, nil
}

func normalizeSHA256(checksum string) (string, error) {
	checksum = strings.TrimSpace(strings.ToLower(checksum))
	checksum = strings.TrimPrefix(checksum, "sha256:")
	if checksum == "" {
		return "", nil
	}
	if len(checksum) != sha256.Size*2 {
		return "", fmt.Errorf("checksum SHA-256 inválido: esperado 64 caracteres hexadecimais")
	}
	if _, err := hex.DecodeString(checksum); err != nil {
		return "", fmt.Errorf("checksum SHA-256 inválido: %w", err)
	}
	return checksum, nil
}
