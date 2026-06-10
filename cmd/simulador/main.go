package main

import (
	"fmt"
	"io"
	"os"

	"github.com/kyriosdata/assinatura/internal/simulator"
	"github.com/kyriosdata/assinatura/internal/storage"
	"github.com/spf13/cobra"
)

const defaultSimulatorPort = 8081

var version = "dev"

type simulatorOptions struct {
	port     int
	source   string
	checksum string
}

func newRootCommand() *cobra.Command {
	options := &simulatorOptions{
		port: defaultSimulatorPort,
	}

	rootCmd := &cobra.Command{
		Use:   "simulador",
		Short: "CLI para gerenciar o Simulador do HubSaúde",
		Long:  "Sistema Runner - CLI para iniciar, parar e consultar o Simulador do HubSaúde.",
	}

	startCmd := &cobra.Command{
		Use:   "start",
		Short: "Inicia o simulador",
		Long:  "Inicia o binário do simulador como processo gerenciado pelo CLI.",
		RunE: func(cmd *cobra.Command, args []string) error {
			return runStart(options, cmd.OutOrStdout())
		},
	}
	startCmd.Flags().IntVar(&options.port, "port", defaultSimulatorPort, "Porta HTTP do Simulador")
	startCmd.Flags().StringVar(&options.source, "source", "", "URL alternativa para baixar o artefato do simulador")
	startCmd.Flags().StringVar(&options.checksum, "checksum", "", "Checksum SHA-256 esperado do artefato do simulador")

	stopCmd := &cobra.Command{
		Use:   "stop",
		Short: "Interrompe o Simulador",
		Long:  "Interrompe uma instância do Simulador gerenciada pelo CLI.",
		RunE: func(cmd *cobra.Command, args []string) error {
			return runStop(options, cmd.OutOrStdout())
		},
	}
	stopCmd.Flags().IntVar(&options.port, "port", defaultSimulatorPort, "Porta HTTP do Simulador")

	statusCmd := &cobra.Command{
		Use:   "status",
		Short: "Exibe o status do Simulador",
		RunE: func(cmd *cobra.Command, args []string) error {
			return runStatus(cmd.OutOrStdout())
		},
	}

	versionCmd := &cobra.Command{
		Use:   "version",
		Short: "Exibe a versão do CLI",
		Run: func(cmd *cobra.Command, args []string) {
			fmt.Fprintf(cmd.OutOrStdout(), "simulador %s\n", version)
		},
	}

	rootCmd.AddCommand(startCmd, stopCmd, statusCmd, versionCmd)
	return rootCmd
}

func runStart(options *simulatorOptions, out io.Writer) error {
	if err := simulator.EnsurePortAvailable(options.port); err != nil {
		return err
	}

	artifactResult, err := simulator.EnsureArtifact(options.source, options.checksum)
	if err != nil {
		return err
	}

	fmt.Fprintf(out, "simulador start definido (porta: %d)\n", options.port)
	fmt.Fprintln(out, "Porta    : disponível")
	if options.source != "" {
		fmt.Fprintf(out, "source   : %s\n", options.source)
	}
	if artifactResult.Downloaded {
		fmt.Fprintf(out, "Artefato : baixado em %s\n", artifactResult.Path)
		if artifactResult.Version != "" {
			fmt.Fprintf(out, "Release  : %s\n", artifactResult.Version)
		}
		if artifactResult.Checksum != "" {
			fmt.Fprintf(out, "SHA-256  : %s\n", artifactResult.Checksum)
		}
	} else {
		fmt.Fprintf(out, "Artefato : %s\n", artifactResult.Path)
	}
	fmt.Fprintln(out, "implementação do ciclo de vida pendente")
	return nil
}

func runStop(options *simulatorOptions, out io.Writer) error {
	if err := storage.EnsureHomeDir(); err != nil {
		return err
	}
	if err := storage.InitDatabase(); err != nil {
		return err
	}

	result, err := simulator.Stop(options.port)
	if err != nil {
		return err
	}
	if !result.Stopped {
		fmt.Fprintf(out, "nenhuma instância gerenciada do simulador encontrada na porta %d\n", options.port)
		return nil
	}

	fmt.Fprintln(out, "simulador interrompido")
	fmt.Fprintf(out, "PID      : %d\n", result.PID)
	fmt.Fprintf(out, "Porta    : %d\n", result.Port)
	return nil
}

func runStatus(out io.Writer) error {
	if err := storage.EnsureHomeDir(); err != nil {
		return err
	}
	if err := storage.InitDatabase(); err != nil {
		return err
	}

	homeDir, err := storage.HomeDir()
	if err != nil {
		return err
	}
	process, err := simulator.GetProcess()
	if err != nil {
		return err
	}

	fmt.Fprintf(out, "Diretório de trabalho : %s\n", homeDir)
	if process == nil {
		fmt.Fprintln(out, "Simulador             : não registrado")
		return nil
	}

	switch process.Status {
	case "running":
		fmt.Fprintln(out, "Simulador             : em execução")
	case "stopped":
		fmt.Fprintln(out, "Simulador             : parado")
	default:
		fmt.Fprintf(out, "Simulador             : %s\n", process.Status)
	}
	fmt.Fprintf(out, "PID                   : %d\n", process.PID)
	fmt.Fprintf(out, "Porta                 : %d\n", process.Port)
	return nil
}

func main() {
	defer storage.Close()

	if err := newRootCommand().Execute(); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}
