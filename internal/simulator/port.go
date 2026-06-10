package simulator

import (
	"fmt"
	"net"
)

func IsPortAvailable(port int) (bool, error) {
	if port <= 0 || port > 65535 {
		return false, fmt.Errorf("porta inválida: %d", port)
	}

	listener, err := net.Listen("tcp", fmt.Sprintf("127.0.0.1:%d", port))
	if err != nil {
		return false, nil
	}
	defer listener.Close()

	return true, nil
}

func EnsurePortAvailable(port int) error {
	available, err := IsPortAvailable(port)
	if err != nil {
		return err
	}
	if !available {
		return fmt.Errorf("porta %d indisponível para iniciar o simulador", port)
	}
	return nil
}
