package network

import (
	"fmt"
	"math/big"
	"net"
)

// GetLocalIP returns the non loopback local IP of the host
func GetLocalIP() string {
	addrs, err := net.InterfaceAddrs()
	if err != nil {
		return ""
	}
	for _, address := range addrs {
		// check the address type and if it is not a loopback the display it
		if ipnet, ok := address.(*net.IPNet); ok && !ipnet.IP.IsLoopback() {
			if ipnet.IP.To4() != nil {
				return ipnet.IP.String()
			}
		}
	}
	return ""
}

// Adapted from https://github.com/netdata/go.d.plugin/blob/master/pkg/iprange/range.go
type V4Range struct {
	start  net.IP
	end    net.IP
	prefix int32
}

func NewV4Range(start, end string, prefix int32) *V4Range {
	return &V4Range{
		start:  ToIpV4(start),
		end:    ToIpV4(end),
		prefix: prefix,
	}
}

// Size reports the number of IP addresses in the range.
func (r V4Range) Size() *big.Int {
	if r.end == nil || r.start == nil {
		return nil
	}
	return big.NewInt(v4ToInt(r.end) - v4ToInt(r.start) + 1)
}

func (r V4Range) Validate(minimumIps int) error {
	cidr := fmt.Sprintf("%s/%d", r.start.String(), r.prefix)
	_, ipNet, err := net.ParseCIDR(cidr)
	if err != nil {
		return err
	}
	if !ipNet.Contains(r.end) {
		return fmt.Errorf("CIDR, %s, does not contain end IP: %s", cidr, r.end)
	}
	size := r.Size()
	if size.Cmp(big.NewInt(int64(minimumIps))) == -1 {
		return fmt.Errorf("IP range contains only %s IP(s). Minimum size: %d", size.String(), minimumIps)
	}
	return nil
}

func v4ToInt(ip net.IP) int64 {
	ip = ip.To4()
	return int64(ip[0])<<24 | int64(ip[1])<<16 | int64(ip[2])<<8 | int64(ip[3])
}

func ToIpV4(ip string) net.IP {
	parsedIP := net.ParseIP(ip)
	if parsedIP == nil {
		return nil
	}
	return parsedIP.To4()
}

func IncIp(ip net.IP) {
	for j := len(ip) - 1; j >= 0; j-- {
		ip[j]++
		if ip[j] > 0 {
			break
		}
	}
}
