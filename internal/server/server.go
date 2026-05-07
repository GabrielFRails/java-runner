package server

import (
	"context"
	"fmt"
	"net/http"
	"os"
	"os/exec"
	"path/filepath"
	"time"

	"github.com/kyriosdata/assinatura/internal/storage"
)

const (
	AssinadorProcessName = "assinador"
	DefaultPort          = 8080
	startupTimeout       = 15 * time.Second
)

type StartResult struct {
	PID     int
	Port    int
	LogPath string
	Reused  bool
}

func Start(javaPath, jarPath string, port int) (*StartResult, error) {
	if port <= 0 || port > 65535 {
		return nil, fmt.Errorf("porta inválida: %d", port)
	}

	home, err := storage.HomeDir()
	if err != nil {
		return nil, err
	}
	if err := os.MkdirAll(home, 0o755); err != nil {
		return nil, fmt.Errorf("não foi possível criar diretório de trabalho: %w", err)
	}

	logPath := filepath.Join(home, "assinador-server.log")

	existing, err := storage.GetProcess(AssinadorProcessName)
	if err != nil {
		return nil, err
	}
	if existing != nil && existing.Status == "running" {
		if isHealthy(existing.Port) {
			return &StartResult{
				PID:     existing.PID,
				Port:    existing.Port,
				LogPath: logPath,
				Reused:  true,
			}, nil
		}

		if err := storage.SaveProcess(storage.Process{
			Name:   AssinadorProcessName,
			PID:    existing.PID,
			Port:   existing.Port,
			Status: "stopped",
		}); err != nil {
			return nil, err
		}
	}

	logFile, err := os.OpenFile(logPath, os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0o644)
	if err != nil {
		return nil, fmt.Errorf("não foi possível abrir log do servidor: %w", err)
	}
	defer logFile.Close()

	cmd := exec.Command(javaPath, "-jar", jarPath, fmt.Sprintf("--server.port=%d", port))
	cmd.Stdout = logFile
	cmd.Stderr = logFile
	cmd.SysProcAttr = backgroundProcessAttributes()

	if err := cmd.Start(); err != nil {
		return nil, fmt.Errorf("erro ao iniciar assinador.jar: %w", err)
	}

	result := &StartResult{
		PID:     cmd.Process.Pid,
		Port:    port,
		LogPath: logPath,
	}

	if err := waitForHealth(port, startupTimeout); err != nil {
		_ = cmd.Process.Kill()
		_ = cmd.Wait()
		return nil, err
	}

	if err := storage.SaveProcess(storage.Process{
		Name:   AssinadorProcessName,
		PID:    result.PID,
		Port:   result.Port,
		Status: "running",
	}); err != nil {
		_ = cmd.Process.Kill()
		_ = cmd.Wait()
		return nil, err
	}

	if err := cmd.Process.Release(); err != nil {
		_ = cmd.Process.Kill()
		return nil, fmt.Errorf("erro ao liberar processo em background: %w", err)
	}

	return result, nil
}

func IsHealthy(port int) bool {
	return isHealthy(port)
}

func isHealthy(port int) bool {
	if port <= 0 || port > 65535 {
		return false
	}

	ctx, cancel := context.WithTimeout(context.Background(), 1*time.Second)
	defer cancel()

	url := fmt.Sprintf("http://127.0.0.1:%d/health", port)
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
	if err != nil {
		return false
	}

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return false
	}
	defer resp.Body.Close()

	return resp.StatusCode == http.StatusOK
}

func waitForHealth(port int, timeout time.Duration) error {
	ctx, cancel := context.WithTimeout(context.Background(), timeout)
	defer cancel()

	ticker := time.NewTicker(300 * time.Millisecond)
	defer ticker.Stop()

	url := fmt.Sprintf("http://127.0.0.1:%d/health", port)
	client := &http.Client{Timeout: 1 * time.Second}

	for {
		req, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
		if err != nil {
			return err
		}

		resp, err := client.Do(req)
		if err == nil {
			_ = resp.Body.Close()
			if resp.StatusCode == http.StatusOK {
				return nil
			}
		}

		select {
		case <-ctx.Done():
			return fmt.Errorf("assinador.jar não respondeu em %s na porta %d", timeout, port)
		case <-ticker.C:
		}
	}
}
