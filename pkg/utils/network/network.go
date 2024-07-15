// Package network provides network utility functions.
package network

import (
	"errors"
	"net"
	"os/exec"
	"strings"

	log "github.com/validator-labs/validatorctl/pkg/logging"
)

// GetDefaultHostAddress returns the default host IPv4 address.
func GetDefaultHostAddress() string {
	devName, err := DiscoverDefaultGatewayDevice()
	if err != nil {
		log.Error("failed to discover default gateway device name: %v", err)
		return ""
	}
	return GetInterfaceIPV4Address(devName)
}

// DiscoverDefaultGatewayDevice discovers the interface name of the default gateway device.
func DiscoverDefaultGatewayDevice() (string, error) {
	cmd := exec.Command("ip", "route", "show", "default")
	output, err := cmd.Output()
	if err != nil {
		log.Error("failed to execute command '%s': %v", cmd.String(), err)
		return "", err
	}
	fields := strings.Fields(string(output))
	for i, field := range fields {
		if field == "dev" && i+1 < len(fields) {
			deviceName := fields[i+1]
			return deviceName, nil
		}
	}
	return "", errors.New("no 'dev' in 'ip route show default' output")
}

// GetInterfaceIPV4Address returns the IPv4 address of the given interface.
func GetInterfaceIPV4Address(ifName string) string {
	nic, err := net.InterfaceByName(ifName)
	if err != nil {
		return ""
	}
	addresses, err := nic.Addrs()
	if err != nil {
		return ""
	}
	for _, addr := range addresses {
		if ipnet, ok := addr.(*net.IPNet); ok && !ipnet.IP.IsLoopback() {
			if ipnet.IP.To4() != nil {
				return ipnet.IP.String()
			}
		}
	}
	return ""
}
