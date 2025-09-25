package util

import (
	"net"
	"strings"
)

// ExtractHost returns the host portion of a network address string.
func ExtractHost(addr string) string {
	if addr == "" {
		return ""
	}
	host, _, err := net.SplitHostPort(addr)
	if err == nil {
		return host
	}
	// If SplitHostPort fails, attempt to trim potential protocol prefixes.
	addr = strings.TrimSpace(addr)
	if strings.HasPrefix(addr, "[") {
		if idx := strings.LastIndex(addr, "]"); idx > 0 {
			return addr[1:idx]
		}
	}
	if idx := strings.Index(addr, ":"); idx > 0 {
		return addr[:idx]
	}
	return addr
}
