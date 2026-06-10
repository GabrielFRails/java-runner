package simulator

import "fmt"

type StopResult struct {
	PID     int
	Port    int
	Stopped bool
}

var terminateProcessFn = terminateProcess

func Stop(port int) (*StopResult, error) {
	if port <= 0 || port > 65535 {
		return nil, fmt.Errorf("porta inválida: %d", port)
	}

	process, err := GetProcess()
	if err != nil {
		return nil, err
	}
	if process == nil || process.Port != port || process.Status != "running" {
		return &StopResult{Port: port}, nil
	}

	result := &StopResult{
		PID:  process.PID,
		Port: process.Port,
	}

	if process.PID > 0 {
		if err := terminateProcessFn(process.PID); err != nil {
			return nil, fmt.Errorf("erro ao encerrar simulador.jar com PID %d: %w", process.PID, err)
		}
	}

	if err := MarkStopped(*process); err != nil {
		return nil, err
	}

	result.Stopped = true
	return result, nil
}
