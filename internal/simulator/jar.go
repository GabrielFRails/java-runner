package simulator

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"time"
)

const defaultLatestReleaseURL = "https://api.github.com/repos/GabrielFRails/java-runner/releases/latest" // deixar mocado mesmo já que o repo é meu hehe
const simulatorReleaseAssetOS = "darwin"                                                                 // [TODO] detectar com runtime.GOOS.
const simulatorReleaseAssetArch = "arm64"                                                                // [TODO] detectar com runtime.GOARCH.
const localSimulatorArtifactName = "simulador-managed"

type ArtifactResult struct {
	Path       string
	Downloaded bool
	SourceURL  string
	Version    string
}

var executablePathFn = os.Executable
var artifactHTTPClient = &http.Client{Timeout: 2 * time.Minute}
var latestReleaseURL = defaultLatestReleaseURL

type githubRelease struct {
	TagName string `json:"tag_name"`
	Assets  []struct {
		Name               string `json:"name"`
		BrowserDownloadURL string `json:"browser_download_url"`
	} `json:"assets"`
}

func LocateArtifact() (string, error) {
	execPath, err := executablePathFn()
	if err != nil {
		return "", fmt.Errorf("não foi possível localizar o executável simulador: %w", err)
	}

	return filepath.Join(filepath.Dir(execPath), localSimulatorArtifactName), nil
}

func EnsureArtifact(sourceURL string) (*ArtifactResult, error) {
	artifactPath, err := LocateArtifact()
	if err != nil {
		return nil, err
	}

	if _, err := os.Stat(artifactPath); err == nil {
		return &ArtifactResult{Path: artifactPath}, nil
	} else if !os.IsNotExist(err) {
		return nil, fmt.Errorf("não foi possível verificar artefato local do simulador: %w", err)
	}

	if sourceURL == "" {
		assetURL, version, err := latestSimulatorReleaseAsset()
		if err != nil {
			return nil, err
		}
		sourceURL = assetURL

		if err := downloadArtifact(sourceURL, artifactPath); err != nil {
			return nil, err
		}

		return &ArtifactResult{Path: artifactPath, Downloaded: true, SourceURL: sourceURL, Version: version}, nil
	}

	if err := downloadArtifact(sourceURL, artifactPath); err != nil {
		return nil, err
	}

	return &ArtifactResult{Path: artifactPath, Downloaded: true, SourceURL: sourceURL}, nil
}

func downloadArtifact(sourceURL string, dest string) error {
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

	if _, err := io.Copy(tmp, resp.Body); err != nil {
		tmp.Close()
		return fmt.Errorf("não foi possível salvar download do artefato do simulador: %w", err)
	}
	if err := tmp.Close(); err != nil {
		return fmt.Errorf("não foi possível fechar download do artefato do simulador: %w", err)
	}

	if err := os.Rename(tmpPath, dest); err != nil {
		return fmt.Errorf("não foi possível instalar artefato do simulador: %w", err)
	}
	if err := os.Chmod(dest, 0o755); err != nil {
		return fmt.Errorf("não foi possível tornar artefato do simulador executável: %w", err)
	}

	return nil
}

func latestSimulatorReleaseAsset() (string, string, error) {
	resp, err := getURL(latestReleaseURL)
	if err != nil {
		return "", "", fmt.Errorf("erro ao consultar GitHub Releases: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode == http.StatusNotFound {
		return "", "", fmt.Errorf("release mais recente não encontrada no GitHub")
	}
	if resp.StatusCode < http.StatusOK || resp.StatusCode >= http.StatusMultipleChoices {
		return "", "", fmt.Errorf("consulta ao GitHub Releases retornou status %d", resp.StatusCode)
	}

	var release githubRelease
	if err := json.NewDecoder(resp.Body).Decode(&release); err != nil {
		return "", "", fmt.Errorf("resposta inválida do GitHub Releases: %w", err)
	}

	assetName := simulatorReleaseAssetName(release.TagName)
	for _, asset := range release.Assets {
		if asset.Name == assetName && asset.BrowserDownloadURL != "" {
			return asset.BrowserDownloadURL, release.TagName, nil
		}
	}

	if release.TagName == "" {
		return "", "", fmt.Errorf("asset do simulador não encontrado na release mais recente")
	}
	return "", "", fmt.Errorf("asset %s não encontrado na release %s", assetName, release.TagName)
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
