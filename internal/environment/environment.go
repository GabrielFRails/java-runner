package environment

import (
	"archive/tar"
	"archive/zip"
	"compress/gzip"
	"fmt"
	"io"
	"net/http"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"runtime"
	"sort"
	"strings"
)

type JavaInfo struct {
	Path    string
	Version string
	Source  string
}

var homeDirFn = os.UserHomeDir
var httpClient = http.DefaultClient

const (
	managedJDKVendor  = "temurin"
	managedJDKVersion = 21
)

func javaExecutable() string {
	if runtime.GOOS == "windows" {
		return "java.exe"
	}
	return "java"
}

func javaHomeExecutable(javaHome string) string {
	if runtime.GOOS == "darwin" {
		candidate := javaHome + "/Contents/Home/bin/" + javaExecutable()
		if _, err := os.Stat(candidate); err == nil {
			return candidate
		}
	}
	return javaHome + "/bin/" + javaExecutable()
}

func managedJDKExecutable() (string, error) {
	managedRoot, err := managedJDKRoot()
	if err != nil {
		return "", err
	}
	entries, err := os.ReadDir(managedRoot)
	if err != nil {
		if os.IsNotExist(err) {
			return "", os.ErrNotExist
		}
		return "", fmt.Errorf("não foi possível listar JDKs gerenciados: %w", err)
	}

	var candidates []string
	for _, entry := range entries {
		if !entry.IsDir() {
			continue
		}

		candidate := javaHomeExecutable(filepath.Join(managedRoot, entry.Name()))
		if _, err := os.Stat(candidate); err == nil {
			candidates = append(candidates, candidate)
		}
	}

	if len(candidates) == 0 {
		return "", os.ErrNotExist
	}

	sort.Strings(candidates)
	return candidates[len(candidates)-1], nil
}

func managedJDKRoot() (string, error) {
	home, err := homeDirFn()
	if err != nil {
		return "", fmt.Errorf("não foi possível determinar o diretório home: %w", err)
	}
	return filepath.Join(home, ".hubsaude", "jdk"), nil
}

func managedJDKInstallDir() (string, error) {
	root, err := managedJDKRoot()
	if err != nil {
		return "", err
	}
	return filepath.Join(root, fmt.Sprintf(
		"%s-%d-%s-%s",
		managedJDKVendor,
		managedJDKVersion,
		runtime.GOOS,
		runtime.GOARCH,
	)), nil
}

func managedJDKArchivePath() (string, error) {
	root, err := managedJDKRoot()
	if err != nil {
		return "", err
	}
	return filepath.Join(root, "downloads", fmt.Sprintf(
		"%s-%d-%s-%s.%s",
		managedJDKVendor,
		managedJDKVersion,
		runtime.GOOS,
		runtime.GOARCH,
		archiveExtension(runtime.GOOS),
	)), nil
}

func adoptiumOS(goos string) (string, error) {
	switch goos {
	case "darwin":
		return "mac", nil
	case "linux":
		return "linux", nil
	case "windows":
		return "windows", nil
	default:
		return "", fmt.Errorf("plataforma não suportada para provisionamento: %s", goos)
	}
}

func adoptiumArch(goarch string) (string, error) {
	switch goarch {
	case "amd64":
		return "x64", nil
	case "arm64":
		return "aarch64", nil
	default:
		return "", fmt.Errorf("arquitetura não suportada para provisionamento: %s", goarch)
	}
}

func archiveExtension(goos string) string {
	if goos == "windows" {
		return "zip"
	}
	return "tar.gz"
}

func latestJDKDownloadURL() (string, error) {
	osName, err := adoptiumOS(runtime.GOOS)
	if err != nil {
		return "", err
	}
	archName, err := adoptiumArch(runtime.GOARCH)
	if err != nil {
		return "", err
	}

	return fmt.Sprintf(
		"https://api.adoptium.net/v3/binary/latest/%d/ga/%s/%s/jdk/hotspot/normal/eclipse",
		managedJDKVersion,
		osName,
		archName,
	), nil
}

func EnsureManagedJDK() (*JavaInfo, error) {
	if path, err := managedJDKExecutable(); err == nil {
		version, versionErr := getJavaVersion(path)
		if versionErr == nil {
			return &JavaInfo{Path: path, Version: version, Source: "downloaded"}, nil
		}
	}

	installDir, err := managedJDKInstallDir()
	if err != nil {
		return nil, err
	}
	archivePath, err := managedJDKArchivePath()
	if err != nil {
		return nil, err
	}
	if err := os.MkdirAll(filepath.Dir(archivePath), 0o755); err != nil {
		return nil, fmt.Errorf("não foi possível criar diretório de downloads do JDK: %w", err)
	}

	if _, err := os.Stat(archivePath); os.IsNotExist(err) {
		url, urlErr := latestJDKDownloadURL()
		if urlErr != nil {
			return nil, urlErr
		}
		if err := downloadFile(url, archivePath); err != nil {
			return nil, err
		}
	}

	if _, err := os.Stat(installDir); os.IsNotExist(err) {
		if err := extractManagedJDKArchive(archivePath, installDir); err != nil {
			return nil, err
		}
	}

	javaPath := javaHomeExecutable(installDir)
	version, err := getJavaVersion(javaPath)
	if err != nil {
		return nil, fmt.Errorf("JDK provisionado, mas inválido: %w", err)
	}

	return &JavaInfo{
		Path:    javaPath,
		Version: version,
		Source:  "downloaded",
	}, nil
}

func downloadFile(url, dest string) error {
	resp, err := httpClient.Get(url)
	if err != nil {
		return fmt.Errorf("falha ao baixar JDK: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("download do JDK retornou status %d", resp.StatusCode)
	}

	file, err := os.Create(dest)
	if err != nil {
		return fmt.Errorf("não foi possível criar arquivo de download do JDK: %w", err)
	}
	defer file.Close()

	if _, err := io.Copy(file, resp.Body); err != nil {
		return fmt.Errorf("não foi possível salvar download do JDK: %w", err)
	}
	return nil
}

func extractManagedJDKArchive(archivePath, installDir string) error {
	tmpDir := installDir + ".tmp"
	_ = os.RemoveAll(tmpDir)
	_ = os.RemoveAll(installDir)

	if err := os.MkdirAll(tmpDir, 0o755); err != nil {
		return fmt.Errorf("não foi possível criar diretório temporário do JDK: %w", err)
	}

	var err error
	switch archiveExtension(runtime.GOOS) {
	case "zip":
		err = extractZip(archivePath, tmpDir)
	default:
		err = extractTarGz(archivePath, tmpDir)
	}
	if err != nil {
		return err
	}

	extractedRoot, err := normalizeExtractedRoot(tmpDir)
	if err != nil {
		return err
	}

	if err := os.Rename(extractedRoot, installDir); err != nil {
		return fmt.Errorf("não foi possível finalizar extração do JDK: %w", err)
	}
	if extractedRoot != tmpDir {
		_ = os.RemoveAll(tmpDir)
	}

	return nil
}

func normalizeExtractedRoot(tmpDir string) (string, error) {
	entries, err := os.ReadDir(tmpDir)
	if err != nil {
		return "", fmt.Errorf("não foi possível inspecionar diretório extraído do JDK: %w", err)
	}
	if len(entries) == 1 && entries[0].IsDir() {
		return filepath.Join(tmpDir, entries[0].Name()), nil
	}
	return tmpDir, nil
}

func extractZip(archivePath, dest string) error {
	reader, err := zip.OpenReader(archivePath)
	if err != nil {
		return fmt.Errorf("não foi possível abrir arquivo zip do JDK: %w", err)
	}
	defer reader.Close()

	for _, file := range reader.File {
		target := filepath.Join(dest, file.Name)
		if !strings.HasPrefix(target, filepath.Clean(dest)+string(os.PathSeparator)) {
			return fmt.Errorf("entrada zip inválida: %s", file.Name)
		}
		if file.FileInfo().IsDir() {
			if err := os.MkdirAll(target, 0o755); err != nil {
				return err
			}
			continue
		}
		if err := os.MkdirAll(filepath.Dir(target), 0o755); err != nil {
			return err
		}
		src, err := file.Open()
		if err != nil {
			return err
		}
		dst, err := os.OpenFile(target, os.O_CREATE|os.O_WRONLY|os.O_TRUNC, file.Mode())
		if err != nil {
			src.Close()
			return err
		}
		if _, err := io.Copy(dst, src); err != nil {
			src.Close()
			dst.Close()
			return err
		}
		src.Close()
		dst.Close()
	}
	return nil
}

func extractTarGz(archivePath, dest string) error {
	file, err := os.Open(archivePath)
	if err != nil {
		return fmt.Errorf("não foi possível abrir arquivo tar.gz do JDK: %w", err)
	}
	defer file.Close()

	gzipReader, err := gzip.NewReader(file)
	if err != nil {
		return fmt.Errorf("não foi possível abrir gzip do JDK: %w", err)
	}
	defer gzipReader.Close()

	tarReader := tar.NewReader(gzipReader)
	for {
		header, err := tarReader.Next()
		if err == io.EOF {
			break
		}
		if err != nil {
			return fmt.Errorf("erro ao ler tar do JDK: %w", err)
		}
		target := filepath.Join(dest, header.Name)
		if !strings.HasPrefix(target, filepath.Clean(dest)+string(os.PathSeparator)) {
			return fmt.Errorf("entrada tar inválida: %s", header.Name)
		}
		switch header.Typeflag {
		case tar.TypeDir:
			if err := os.MkdirAll(target, 0o755); err != nil {
				return err
			}
		case tar.TypeReg:
			if err := os.MkdirAll(filepath.Dir(target), 0o755); err != nil {
				return err
			}
			dst, err := os.OpenFile(target, os.O_CREATE|os.O_WRONLY|os.O_TRUNC, os.FileMode(header.Mode))
			if err != nil {
				return err
			}
			if _, err := io.Copy(dst, tarReader); err != nil {
				dst.Close()
				return err
			}
			dst.Close()
		}
	}
	return nil
}

func DetectJava() (*JavaInfo, error) {
	if path, err := managedJDKExecutable(); err == nil {
		version, versionErr := getJavaVersion(path)
		if versionErr == nil {
			return &JavaInfo{
				Path:    path,
				Version: version,
				Source:  "downloaded",
			}, nil
		}
	}

	javaHome := os.Getenv("JAVA_HOME")
	if javaHome != "" {
		path := javaHomeExecutable(javaHome)
		version, err := getJavaVersion(path)
		if err == nil {
			return &JavaInfo{
				Path:    path,
				Version: version,
				Source:  "system",
			}, nil
		}
	}

	path, err := exec.LookPath(javaExecutable())
	if err == nil {
		version, versionErr := getJavaVersion(path)
		if versionErr == nil {
			return &JavaInfo{
				Path:    path,
				Version: version,
				Source:  "system",
			}, nil
		}
	}

	info, provisionErr := EnsureManagedJDK()
	if provisionErr == nil {
		return info, nil
	}

	return nil, fmt.Errorf("java não encontrado no PATH nem em JAVA_HOME e não foi possível provisionar automaticamente: %w", provisionErr)
}

func getJavaVersion(javaPath string) (string, error) {
	cmd := exec.Command(javaPath, "-version")
	output, err := cmd.CombinedOutput()
	if err != nil {
		return "", fmt.Errorf("erro ao executar java: %w", err)
	}

	re := regexp.MustCompile(`version "([^"]+)"`)
	match := re.FindStringSubmatch(string(output))
	if len(match) < 2 {
		return "", fmt.Errorf("não foi possível extrair versão do output: %s", output)
	}

	return strings.TrimSpace(match[1]), nil
}

func IsVersionCompatible(version string, minMajor int) bool {
	major := parseMajorVersion(version)
	return major >= minMajor
}

func parseMajorVersion(version string) int {
	if strings.HasPrefix(version, "1.") {
		var legacy int
		fmt.Sscanf(version, "1.%d", &legacy)
		return legacy
	}

	var major int
	fmt.Sscanf(version, "%d", &major)
	return major
}

func CurrentOS() string {
	switch runtime.GOOS {
	case "windows":
		return "Windows"
	case "darwin":
		return "macOS"
	default:
		return "Linux"
	}
}

func CurrentArch() string {
	switch runtime.GOARCH {
	case "amd64":
		return "amd64"
	case "arm64":
		return "arm64"
	default:
		return runtime.GOARCH
	}
}
