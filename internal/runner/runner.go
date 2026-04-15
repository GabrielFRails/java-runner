package runner

import (
	"fmt"

	"github.com/kyriosdata/assinatura/internal/environment"
	"github.com/kyriosdata/assinatura/internal/storage"
)

const MinJavaMajor = 21

type StartupResult struct {
	JavaInfo *environment.JavaInfo
	Warnings []string
}

// Startup executa a sequência de inicialização do CLI:
// 1. garante ~/.hubsaude
// 2. inicializa banco de dados
// 3. detecta Java
// 4. registra Java no banco se encontrado
func Startup() (*StartupResult, error) {
	result := &StartupResult{}

	if err := storage.EnsureHomeDir(); err != nil {
		return nil, fmt.Errorf("erro ao criar diretório de trabalho: %w", err)
	}

	if err := storage.InitDatabase(); err != nil {
		return nil, fmt.Errorf("erro ao inicializar banco de dados: %w", err)
	}

	javaInfo, err := environment.DetectJava()
	if err != nil {
		result.Warnings = append(result.Warnings, fmt.Sprintf("Java não encontrado: %v", err))
		return result, nil
	}

	if !environment.IsVersionCompatible(javaInfo.Version, MinJavaMajor) {
		result.Warnings = append(result.Warnings,
			fmt.Sprintf("Java %s encontrado, mas versão mínima exigida é %d",
				javaInfo.Version, MinJavaMajor))
		return result, nil
	}

	if err := saveJavaRuntime(javaInfo); err != nil {
		result.Warnings = append(result.Warnings,
			fmt.Sprintf("não foi possível salvar runtime Java no banco: %v", err))
	}

	result.JavaInfo = javaInfo
	return result, nil
}

func saveJavaRuntime(info *environment.JavaInfo) error {
	_, err := storage.DB.Exec(`
		INSERT INTO java_runtime (path, version, source)
		VALUES (?, ?, ?)
		ON CONFLICT DO NOTHING
	`, info.Path, info.Version, info.Source)
	return err
}
