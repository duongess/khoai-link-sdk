package utils

import (
	"fmt"
	"net"
	"net/url"
	"strings"
	"time"
)

// GetOutboundIP tim IP mang LAN thuc te cua Node dang ket noi ra Gateway
func GetOutboundIP(gatewayAddr string) (string, error) {
	// 1. Neu co gatewayAddr, thu mo ket noi UDP gia lap (khong gui goi tin thuc)
	// He dieu hanh se tu chon interface card mang hop ly nhat
	if gatewayAddr != "" {
		host := extractHost(gatewayAddr)
		conn, err := net.DialTimeout("udp", host+":80", 1*time.Second)
		if err == nil {
			defer conn.Close()
			localAddr := conn.LocalAddr().(*net.UDPAddr)
			return localAddr.IP.String(), nil
		}
	}

	// 2. Fallback: Quet cac network interfaces hop le
	interfaces, err := net.Interfaces()
	if err != nil {
		return "", fmt.Errorf("failed to get network interfaces: %w", err)
	}

	for _, iface := range interfaces {
		// Bo qua interface da tat hoac loopback
		if iface.Flags&net.FlagUp == 0 || iface.Flags&net.FlagLoopback != 0 {
			continue
		}

		// Bo qua card mang ao Docker, VMware, veth
		name := strings.ToLower(iface.Name)
		if strings.HasPrefix(name, "docker") || strings.HasPrefix(name, "veth") || strings.HasPrefix(name, "br-") {
			continue
		}

		addrs, err := iface.Addrs()
		if err != nil {
			continue
		}

		for _, addr := range addrs {
			var ip net.IP
			switch v := addr.(type) {
			case *net.IPNet:
				ip = v.IP
			case *net.IPAddr:
				ip = v.IP
			}

			// Chi lay IPv4 va khong phai loopback
			if ip == nil || ip.IsLoopback() {
				continue
			}
			ip = ip.To4()
			if ip == nil {
				continue
			}

			return ip.String(), nil
		}
	}

	return "", fmt.Errorf("no valid outbound LAN IP found")
}

// FormatEndpoint chuan hoa IP + Port thanh endpoint URL day du: http://192.168.1.10:3000
func FormatEndpoint(ip string, port int) string {
	ip = strings.TrimSpace(ip)
	if !strings.HasPrefix(ip, "http://") && !strings.HasPrefix(ip, "https://") {
		ip = "http://" + ip
	}

	// Neu ip chua co port thi them vao
	u, err := url.Parse(ip)
	if err == nil && u.Port() == "" && port > 0 {
		return fmt.Sprintf("%s:%d", strings.TrimRight(ip, "/"), port)
	}

	return strings.TrimRight(ip, "/")
}

// IsPortAvailable kiem tra xem port co dang bi chiem dung hay khong
func IsPortAvailable(port int) bool {
	ln, err := net.Listen("tcp", fmt.Sprintf(":%d", port))
	if err != nil {
		return false
	}
	_ = ln.Close()
	return true
}

// extractHost tach lay host/ip tu URL hoac chuoi host:port
func extractHost(rawAddr string) string {
	if strings.Contains(rawAddr, "://") {
		u, err := url.Parse(rawAddr)
		if err == nil {
			return u.Hostname()
		}
	}
	host, _, err := net.SplitHostPort(rawAddr)
	if err == nil {
		return host
	}
	return rawAddr
}
