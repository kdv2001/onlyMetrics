package network

import (
	"fmt"
	"net"
)

// GetLocalIP возвращает локальный адрес
func GetLocalIP() (net.IP, error) {
	iFaces, err := net.Interfaces()
	if err != nil {
		return nil, fmt.Errorf("error get net interfaces: %w", err)
	}

	for _, iface := range iFaces {
		// Пропускаем интерфейсы, которые не активны или являются петлевыми
		if iface.Flags&net.FlagUp == 0 || iface.Flags&net.FlagLoopback != 0 {
			continue
		}

		addrs, err := iface.Addrs()
		if err != nil {
			return nil, fmt.Errorf("error get interfaces address %s: %w", iface.Name, err)
		}

		for _, addr := range addrs {
			if ipnet, ok := addr.(*net.IPNet); ok && !ipnet.IP.IsLoopback() && ipnet.IP.To4() != nil {
				return ipnet.IP.To4(), nil
			}
		}
	}

	return nil, fmt.Errorf("not found local ip")
}
