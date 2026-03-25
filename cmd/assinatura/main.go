package main

import (
    "fmt"
    "os"

    "github.com/GabrielFRails/assinatura/internal/environment"
    "github.com/GabrielFRails/assinatura/internal/storage"
)

func main() {
    // 1. garante ~/.hubsaude
    if err := storage.EnsureHomeDir(); err != nil {
        fmt.Fprintf(os.Stderr, "erro ao criar diretório: %v\n", err)
        os.Exit(1)
    }

    // 2. detecta Java
    javaInfo, err := environment.DetectJava()
    if err != nil {
        fmt.Println("Java não encontrado:", err)
    } else {
        fmt.Printf("Java encontrado: versão %s em %s\n", javaInfo.Version, javaInfo.Path)
    }
}