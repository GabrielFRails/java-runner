package simulator

import "github.com/kyriosdata/assinatura/internal/storage"

const ProcessName = "simulador"

func SaveProcess(pid int, port int, status string) error {
	return storage.SaveProcess(storage.Process{
		Name:   ProcessName,
		PID:    pid,
		Port:   port,
		Status: status,
	})
}

func GetProcess() (*storage.Process, error) {
	return storage.GetProcess(ProcessName)
}

func MarkStopped(process storage.Process) error {
	return SaveProcess(process.PID, process.Port, "stopped")
}
