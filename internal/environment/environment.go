package environment

import (
    "fmt"
    "os"
    "os/exec"
    "regexp"
    "strings"
)

type JavaInfo struct {
    Path    string
    Version string
}

func DetectJava() (*JavaInfo, error) {
    javaHome := os.Getenv("JAVA_HOME")
    if javaHome != "" {
        path := javaHome + "/bin/java"
        version, err := getJavaVersion(path)
        if err == nil {
            return &JavaInfo{Path: path, Version: version}, nil
        }
    }

    path, err := exec.LookPath("java")
    if err != nil {
        return nil, fmt.Errorf("java não encontrado no PATH nem em JAVA_HOME")
    }

    version, err := getJavaVersion(path)
    if err != nil {
        return nil, err
    }

    return &JavaInfo{Path: path, Version: version}, nil
}

func getJavaVersion(javaPath string) (string, error) {
    cmd := exec.Command(javaPath, "-version")
    output, err := cmd.CombinedOutput()
    if err != nil {
        return "", fmt.Errorf("erro ao executar java: %v", err)
    }

    re := regexp.MustCompile(`version "([^"]+)"`)
    match := re.FindStringSubmatch(string(output))
    if len(match) < 2 {
        return "", fmt.Errorf("não foi possível extrair versão do output: %s", output)
    }

    return strings.TrimSpace(match[1]), nil
}