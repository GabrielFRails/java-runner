package jar

import (
	"bytes"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
)

const assinadorJarName = "assinador.jar"

// Result contém o resultado da execução do jar
type Result struct {
	Output   string // stdout do processo
	ExitCode int
}

// Locate encontra o assinador.jar no mesmo diretório do executável atual.
// Retorna o path absoluto ou erro se não encontrado.
func Locate() (string, error) {
	execPath, err := os.Executable()
	if err != nil {
		return "", fmt.Errorf("não foi possível determinar o path do executável: %w", err)
	}

	// Resolve symlinks (importante para `go run` e instalações via symlink)
	execPath, err = filepath.EvalSymlinks(execPath)
	if err != nil {
		return "", fmt.Errorf("não foi possível resolver symlinks do executável: %w", err)
	}

	dir := filepath.Dir(execPath)
	jarPath := filepath.Join(dir, assinadorJarName)

	if _, err := os.Stat(jarPath); os.IsNotExist(err) {
		return "", fmt.Errorf(
			"%s não encontrado em %s\n"+
				"coloque o assinador.jar no mesmo diretório do executável assinatura",
			assinadorJarName, dir,
		)
	}

	return jarPath, nil
}

// InvokeSign executa o assinador.jar no modo sign com as flags fornecidas.
// javaPath é o path completo do executável java.
// jarPath é o path completo do assinador.jar.
func InvokeSign(javaPath, jarPath string, flags SignFlags) (*Result, error) {
	args := buildSignArgs(jarPath, flags)
	return invoke(javaPath, args)
}

// InvokeValidate executa o assinador.jar no modo validate com as flags fornecidas.
func InvokeValidate(javaPath, jarPath string, flags ValidateFlags) (*Result, error) {
	args := buildValidateArgs(jarPath, flags)
	return invoke(javaPath, args)
}

// invoke executa java com os argumentos fornecidos e captura stdout/stderr.
func invoke(javaPath string, args []string) (*Result, error) {
	cmd := exec.Command(javaPath, args...)

	var stdout, stderr bytes.Buffer
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr

	err := cmd.Run()

	// Captura exit code
	exitCode := 0
	if err != nil {
		if exitErr, ok := err.(*exec.ExitError); ok {
			exitCode = exitErr.ExitCode()
		} else {
			// Erro de execução (java não encontrado, permissão negada, etc.)
			return nil, fmt.Errorf("erro ao executar o assinador.jar: %w", err)
		}
	}

	// O jar escreve erros na stderr e resultado na stdout
	output := stdout.String()
	if output == "" && stderr.String() != "" {
		output = stderr.String()
	}

	return &Result{
		Output:   strings.TrimSpace(output),
		ExitCode: exitCode,
	}, nil
}

// buildSignArgs monta os argumentos para o comando sign
func buildSignArgs(jarPath string, f SignFlags) []string {
	args := []string{"-jar", jarPath, "sign"}

	if f.Bundle != "" {
		args = append(args, "--bundle", f.Bundle)
	}
	if f.Provenance != "" {
		args = append(args, "--provenance", f.Provenance)
	}
	if f.Timestamp != 0 {
		args = append(args, "--timestamp", fmt.Sprintf("%d", f.Timestamp))
	}
	if f.Strategy != "" {
		args = append(args, "--strategy", f.Strategy)
	}
	if f.PolicyUri != "" {
		args = append(args, "--policy", f.PolicyUri)
	}
	if f.CertChain != "" {
		args = append(args, "--cert", f.CertChain)
	}
	if f.CryptoType != "" {
		args = append(args, "--crypto-type", f.CryptoType)
	}
	if f.CryptoPem != "" {
		args = append(args, "--crypto-pem", f.CryptoPem)
	}
	if f.CryptoPassword != "" {
		args = append(args, "--crypto-password", f.CryptoPassword)
	}
	if f.CryptoPkcs12 != "" {
		args = append(args, "--crypto-pkcs12", f.CryptoPkcs12)
	}
	if f.CryptoAlias != "" {
		args = append(args, "--crypto-alias", f.CryptoAlias)
	}
	if f.Config != "" {
		args = append(args, "--config", f.Config)
	}

	return args
}

// buildValidateArgs monta os argumentos para o comando validate
func buildValidateArgs(jarPath string, f ValidateFlags) []string {
	args := []string{"-jar", jarPath, "validate"}

	if f.Signature != "" {
		args = append(args, "--signature", f.Signature)
	}
	if f.Timestamp != 0 {
		args = append(args, "--timestamp", fmt.Sprintf("%d", f.Timestamp))
	}
	if f.PolicyUri != "" {
		args = append(args, "--policy", f.PolicyUri)
	}
	if f.Config != "" {
		args = append(args, "--config", f.Config)
	}

	return args
}

// SignFlags agrupa as flags do comando sign
type SignFlags struct {
	Bundle         string
	Provenance     string
	Timestamp      int64
	Strategy       string
	PolicyUri      string
	CertChain      string
	CryptoType     string
	CryptoPem      string
	CryptoPassword string
	CryptoPkcs12   string
	CryptoAlias    string
	Config         string
}

// ValidateFlags agrupa as flags do comando validate
type ValidateFlags struct {
	Signature string
	Timestamp int64
	PolicyUri string
	Config    string
}