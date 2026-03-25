package main

import (
	"fmt"
	"os"

	"github.com/spf13/cobra"
	"github.com/GabrielFRails/assinatura/internal/environment"
	"github.com/GabrielFRails/assinatura/internal/runner"
	"github.com/GabrielFRails/assinatura/internal/storage"
)

var version = "dev"
var rootCmd = &cobra.Command{
	Use:   "assinatura",
	Short: "CLI para operações de assinatura digital",
	Long:  `Sistema Runner - CLI para criação e validação de assinaturas digitais via assinador.jar`,

	PersistentPreRunE: func(cmd *cobra.Command, args []string) error {
		return startup()
	},
}

var statusCmd = &cobra.Command{
	Use:   "status",
	Short: "Exibe o status do ambiente",
	RunE: func(cmd *cobra.Command, args []string) error {
		return runStatus()
	},
}

var versionCmd = &cobra.Command{
	Use:   "version",
	Short: "Exibe a versão do CLI",
	PersistentPreRunE: func(cmd *cobra.Command, args []string) error { return nil },
	Run: func(cmd *cobra.Command, args []string) {
		fmt.Printf("assinatura %s\n", version)
	},
}

func init() {
	rootCmd.AddCommand(statusCmd)
	rootCmd.AddCommand(versionCmd)
}

func main() {
	defer storage.Close()

	if err := rootCmd.Execute(); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}

func startup() error {
	result, err := runner.Startup()
	if err != nil {
		return fmt.Errorf("falha na inicialização: %w", err)
	}

	for _, w := range result.Warnings {
		fmt.Fprintf(os.Stderr, "aviso: %s\n", w)
	}

	return nil
}

func runStatus() error {
	dir, err := storage.HomeDir()
	if err != nil {
		return err
	}
	fmt.Printf("Diretório de trabalho : %s\n", dir)
	fmt.Printf("Sistema operacional   : %s\n", environment.CurrentOS())
	fmt.Printf("Arquitetura           : %s\n", environment.CurrentArch())

	javaInfo, err := environment.DetectJava()
	if err != nil {
		fmt.Printf("Java                  : não encontrado\n")
		return nil
	}

	fmt.Printf("Java                  : %s\n", javaInfo.Version)
	fmt.Printf("Java path             : %s\n", javaInfo.Path)

	if environment.IsVersionCompatible(javaInfo.Version, runner.MinJavaMajor) {
        fmt.Printf("Java compatível       : sim (mínimo: %d)\n", runner.MinJavaMajor)
    } else {
        fmt.Printf("Java compatível       : não (mínimo: %d, encontrado: %s)\n", runner.MinJavaMajor, javaInfo.Version)
    }

	return nil
}