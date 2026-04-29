package environment

import (
	"fmt"
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
	home, err := homeDirFn()
	if err != nil {
		return "", fmt.Errorf("não foi possível determinar o diretório home: %w", err)
	}

	managedRoot := filepath.Join(home, ".hubsaude", "jdk")
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
	if err != nil {
		return nil, fmt.Errorf("java não encontrado no PATH nem em JAVA_HOME")
	}

	version, err := getJavaVersion(path)
	if err != nil {
		return nil, err
	}

	return &JavaInfo{
		Path:    path,
		Version: version,
		Source:  "system",
	}, nil
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
