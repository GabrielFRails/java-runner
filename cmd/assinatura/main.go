package main

import (
	"fmt"
	"os"
	"time"

	"github.com/spf13/cobra"
	"github.com/kyriosdata/assinatura/internal/environment"
	"github.com/kyriosdata/assinatura/internal/jar"
	"github.com/kyriosdata/assinatura/internal/runner"
	"github.com/kyriosdata/assinatura/internal/storage"
)

var version = "dev"

var signFlags jar.SignFlags
var validateFlags jar.ValidateFlags

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
	PersistentPreRunE: func(cmd *cobra.Command, args []string) error {
		return nil
	},
	Run: func(cmd *cobra.Command, args []string) {
		fmt.Printf("assinatura %s\n", version)
	},
}

var signCmd = &cobra.Command{
	Use:   "sign",
	Short: "Cria uma assinatura digital via assinador.jar",
	Long: `Invoca o assinador.jar para criar uma assinatura digital simulada.
O assinador.jar deve estar no mesmo diretório do executável assinatura.`,
	Example: `  assinatura sign \
	--bundle bundle.json \
	--provenance provenance.json \
	--strategy iat \
	--policy "https://fhir.saude.go.gov.br/r4/seguranca/ImplementationGuide/br.go.ses.seguranca|0.1.2" \
	--cert chain.json \
	--crypto-type PEM \
	--crypto-pem key.pem \
	--config config.json`,
	RunE: func(cmd *cobra.Command, args []string) error {
		return runSign()
	},
}

var validateCmd = &cobra.Command{
	Use:   "validate",
	Short: "Valida uma assinatura digital via assinador.jar",
	Long: `Invoca o assinador.jar para validar uma assinatura digital.
O assinador.jar deve estar no mesmo diretório do executável assinatura.`,
	Example: `  assinatura validate \
	--signature signature.json \
	--policy "https://fhir.saude.go.gov.br/r4/seguranca/ImplementationGuide/br.go.ses.seguranca|0.1.2" \
	--config config.json`,
	RunE: func(cmd *cobra.Command, args []string) error {
		return runValidate()
	},
}

func init() {
	// flags do comando sign
	signCmd.Flags().StringVar(&signFlags.Bundle, "bundle", "", "Arquivo JSON com o Bundle FHIR R4 (obrigatório)")
	signCmd.Flags().StringVar(&signFlags.Provenance, "provenance", "", "Arquivo JSON com o Provenance FHIR R4 (obrigatório)")
	signCmd.Flags().Int64Var(&signFlags.Timestamp, "timestamp", time.Now().Unix(), "Timestamp de referência Unix UTC")
	signCmd.Flags().StringVar(&signFlags.Strategy, "strategy", "iat", "Estratégia de timestamp: iat ou tsa")
	signCmd.Flags().StringVar(&signFlags.PolicyUri, "policy", "", "URI da política de assinatura (obrigatório)")
	signCmd.Flags().StringVar(&signFlags.CertChain, "cert", "", "Arquivo JSON com array de certificados base64 DER (obrigatório)")
	signCmd.Flags().StringVar(&signFlags.CryptoType, "crypto-type", "PEM", "Tipo do material criptográfico: PEM, PKCS12, SMARTCARD, TOKEN")
	signCmd.Flags().StringVar(&signFlags.CryptoPem, "crypto-pem", "", "Arquivo com chave privada PEM")
	signCmd.Flags().StringVar(&signFlags.CryptoPassword, "crypto-password", "", "Senha do material criptográfico")
	signCmd.Flags().StringVar(&signFlags.CryptoPkcs12, "crypto-pkcs12", "", "Arquivo PKCS12 em base64")
	signCmd.Flags().StringVar(&signFlags.CryptoAlias, "crypto-alias", "", "Alias da chave no PKCS12")
	signCmd.Flags().StringVar(&signFlags.Config, "config", "", "Arquivo JSON com configurações operacionais (obrigatório)")

	// flags do comando validate
	validateCmd.Flags().StringVar(&validateFlags.Signature, "signature", "", "Arquivo com o Signature.data em base64 (obrigatório)")
	validateCmd.Flags().Int64Var(&validateFlags.Timestamp, "timestamp", time.Now().Unix(), "Timestamp de referência Unix UTC")
	validateCmd.Flags().StringVar(&validateFlags.PolicyUri, "policy", "", "URI da política de assinatura (obrigatório)")
	validateCmd.Flags().StringVar(&validateFlags.Config, "config", "", "Arquivo JSON com configurações operacionais (obrigatório)")

	rootCmd.AddCommand(statusCmd)
	rootCmd.AddCommand(versionCmd)
	rootCmd.AddCommand(signCmd)
	rootCmd.AddCommand(validateCmd)
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

func runSign() error {
	javaInfo, err := environment.DetectJava()
	if err != nil {
		return fmt.Errorf("Java não encontrado: %w\nInstale o Java ou aguarde o provisionamento automático (US-04)", err)
	}

	jarPath, err := jar.Locate()
	if err != nil {
		return err
	}

	result, err := jar.InvokeSign(javaInfo.Path, jarPath, signFlags)
	if err != nil {
		return err
	}

	fmt.Println(result.Output)

	if result.ExitCode != 0 {
		os.Exit(result.ExitCode)
	}
	return nil
}

func runValidate() error {
	javaInfo, err := environment.DetectJava()
	if err != nil {
		return fmt.Errorf("Java não encontrado: %w\nInstale o Java ou aguarde o provisionamento automático (US-04)", err)
	}

	jarPath, err := jar.Locate()
	if err != nil {
		return err
	}

	result, err := jar.InvokeValidate(javaInfo.Path, jarPath, validateFlags)
	if err != nil {
		return err
	}

	fmt.Println(result.Output)

	if result.ExitCode != 0 {
		os.Exit(result.ExitCode)
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

	jarPath, err := jar.Locate()
	if err != nil {
		fmt.Printf("assinador.jar         : não encontrado\n")
		return nil
	}
	fmt.Printf("assinador.jar         : %s\n", jarPath)

	return nil
}
