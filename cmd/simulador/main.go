package main

import (
	"fmt"
	"os"

	"github.com/spf13/cobra"
)

const defaultSimulatorPort = 8081

var version = "dev"

type simulatorOptions struct {
	port   int
	source string
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
		Short: "Inicia o simulador.jar",
		Long:  "Inicia o simulador.jar como processo gerenciado pelo CLI.",
		RunE: func(cmd *cobra.Command, args []string) error {
			return runStart(options)
		},
	}
	startCmd.Flags().IntVar(&options.port, "port", defaultSimulatorPort, "Porta HTTP do Simulador")
	startCmd.Flags().StringVar(&options.source, "source", "", "URL alternativa para baixar o simulador.jar")

	stopCmd := &cobra.Command{
		Use:   "stop",
		Short: "Interrompe o Simulador",
		Long:  "Interrompe uma instância do Simulador gerenciada pelo CLI.",
		RunE: func(cmd *cobra.Command, args []string) error {
			return runStop(options)
		},
	}
	stopCmd.Flags().IntVar(&options.port, "port", defaultSimulatorPort, "Porta HTTP do Simulador")

	statusCmd := &cobra.Command{
		Use:   "status",
		Short: "Exibe o status do Simulador",
		RunE: func(cmd *cobra.Command, args []string) error {
			return runStatus(options)
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

func runStart(options *simulatorOptions) error {
	fmt.Printf("simulador start definido (porta: %d)\n", options.port)
	if options.source != "" {
		fmt.Printf("source   : %s\n", options.source)
	}
	fmt.Println("implementação do ciclo de vida pendente")
	return nil
}

func runStop(options *simulatorOptions) error {
	fmt.Printf("simulador stop definido (porta: %d)\n", options.port)
	fmt.Println("implementação do ciclo de vida pendente")
	return nil
}

func runStatus(options *simulatorOptions) error {
	fmt.Println("simulador status definido")
	fmt.Println("implementação do ciclo de vida pendente")
	return nil
}

func main() {
	if err := newRootCommand().Execute(); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}
