package storage

import (
    "os"
    "path/filepath"
)

func HomeDir() (string, error) {
    home, err := os.UserHomeDir()
    if err != nil {
        return "", err
    }
    return filepath.Join(home, ".hubsaude"), nil
}

func EnsureHomeDir() error {
    dir, err := HomeDir()
    if err != nil {
        return err
    }
    // MkdirAll não falha se o diretório já existe
    return os.MkdirAll(dir, 0755)
}