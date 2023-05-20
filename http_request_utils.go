package mux

import (
	"net/http"
	"strings"
)

func GetHost(r *http.Request) string {
	var host string
	host = r.Host
	if strings.Contains(host, ":") {
		host = strings.Split(host, ":")[0]
	}
	return host
}

func GetIP(r *http.Request, proxied bool) string {
	var ip string
	if ip = r.Header.Get("X-Forwarded-For"); ip != "" && proxied {
		goto parse
	} else if ip = r.Header.Get("X-Real-IP"); ip != "" && proxied {
		goto parse
	} else {
		ip = r.RemoteAddr
		goto parse
	}
parse:
	// Parse the IP address.
	if i := strings.Index(ip, ":"); i != -1 {
		ip = ip[:i]
	}
	return ip
}
