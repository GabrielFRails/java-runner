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

const simulatorJarName = "simulador.jar"
const defaultLatestReleaseURL = "https://api.github.com/repos/GabrielFRails/java-runner/releases/latest" // deixar mocado mesmo já que o repo é meu hehe

type JarResult struct {
	Path       string
	Downloaded bool
	SourceURL  string
	Version    string
}

var executablePathFn = os.Executable
var jarHTTPClient = &http.Client{Timeout: 2 * time.Minute}
var latestReleaseURL = defaultLatestReleaseURL

type githubRelease struct {
	TagName string `json:"tag_name"`
	Assets  []struct {
		Name               string `json:"name"`
		BrowserDownloadURL string `json:"browser_download_url"`
	} `json:"assets"`
}

func LocateJar() (string, error) {
	execPath, err := executablePathFn()
	if err != nil {
		return "", fmt.Errorf("não foi possível localizar o executável simulador: %w", err)
	}

	return filepath.Join(filepath.Dir(execPath), simulatorJarName), nil
}

func EnsureJar(sourceURL string) (*JarResult, error) {
	jarPath, err := LocateJar()
	if err != nil {
		return nil, err
	}

	if _, err := os.Stat(jarPath); err == nil {
		return &JarResult{Path: jarPath}, nil
	} else if !os.IsNotExist(err) {
		return nil, fmt.Errorf("não foi possível verificar %s: %w", simulatorJarName, err)
	}

	if sourceURL == "" {
		assetURL, version, err := latestSimulatorJarAsset()
		if err != nil {
			return nil, err
		}
		sourceURL = assetURL

		if err := downloadJar(sourceURL, jarPath); err != nil {
			return nil, err
		}

		return &JarResult{Path: jarPath, Downloaded: true, SourceURL: sourceURL, Version: version}, nil
	}

	if err := downloadJar(sourceURL, jarPath); err != nil {
		return nil, err
	}

	return &JarResult{Path: jarPath, Downloaded: true, SourceURL: sourceURL}, nil
}

func downloadJar(sourceURL string, dest string) error {
	resp, err := getURL(sourceURL)
	if err != nil {
		return fmt.Errorf("erro ao baixar %s: %w", simulatorJarName, err)
	}
	defer resp.Body.Close()

	if resp.StatusCode < http.StatusOK || resp.StatusCode >= http.StatusMultipleChoices {
		return fmt.Errorf("download de %s retornou status %d", simulatorJarName, resp.StatusCode)
	}

	if err := os.MkdirAll(filepath.Dir(dest), 0o755); err != nil {
		return fmt.Errorf("não foi possível criar diretório para %s: %w", simulatorJarName, err)
	}

	tmp, err := os.CreateTemp(filepath.Dir(dest), simulatorJarName+".*.tmp")
	if err != nil {
		return fmt.Errorf("não foi possível criar arquivo temporário para %s: %w", simulatorJarName, err)
	}
	tmpPath := tmp.Name()
	defer os.Remove(tmpPath)

	if _, err := io.Copy(tmp, resp.Body); err != nil {
		tmp.Close()
		return fmt.Errorf("não foi possível salvar download de %s: %w", simulatorJarName, err)
	}
	if err := tmp.Close(); err != nil {
		return fmt.Errorf("não foi possível fechar download de %s: %w", simulatorJarName, err)
	}

	if err := os.Rename(tmpPath, dest); err != nil {
		return fmt.Errorf("não foi possível instalar %s: %w", simulatorJarName, err)
	}

	return nil
}

func latestSimulatorJarAsset() (string, string, error) {
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

	for _, asset := range release.Assets {
		if asset.Name == simulatorJarName && asset.BrowserDownloadURL != "" {
			return asset.BrowserDownloadURL, release.TagName, nil
		}
	}

	if release.TagName == "" {
		return "", "", fmt.Errorf("asset %s não encontrado na release mais recente", simulatorJarName)
	}
	return "", "", fmt.Errorf("asset %s não encontrado na release %s", simulatorJarName, release.TagName)
}

func getURL(rawURL string) (*http.Response, error) {
	req, err := http.NewRequest(http.MethodGet, rawURL, nil)
	if err != nil {
		return nil, err
	}
	req.Header.Set("Accept", "application/vnd.github+json")
	req.Header.Set("User-Agent", "hubsaude-simulador-cli")
	return jarHTTPClient.Do(req)
}
