package commands

import (
	"fmt"
	"net"
	"net/http"
	"syscall"
	"time"
)

// wgetSocketControl prevents basic SSRF attacks by only allowing certain kinds
// of connections.
func wgetSocketControl(network string, address string, conn syscall.RawConn) error {
	switch network {
	case "tcp", "tcp4", "tcp6", "udp", "udp4", "udp6":
		// Accept types used for HTTP1/2/3.
	default:
		// This is likely something like the attacker trying to open a UNIX socket,
		// bail.
		return fmt.Errorf("unknown network type: %v", network)
	}

	host, _, err := net.SplitHostPort(address)
	if err != nil {
		return fmt.Errorf("bad network address: %v", address)
	}

	ipAddress := net.ParseIP(host)
	if ipAddress == nil {
		return fmt.Errorf("bad network address: %v", address)
	}

	if ipAddress.IsLoopback() || ipAddress.IsPrivate() {
		// Prevent loopback or fetches to private networks.
		return fmt.Errorf("couldn't resolve: %s", address)
	}

	return nil
}

var wgetDialer = &net.Dialer{
	Timeout: 5 * time.Second,
	Control: wgetSocketControl,
}

var wgetTransport = &http.Transport{
	Proxy:                 http.ProxyFromEnvironment,
	ForceAttemptHTTP2:     true,
	MaxIdleConns:          10,
	IdleConnTimeout:       90 * time.Second,
	TLSHandshakeTimeout:   5 * time.Second,
	ExpectContinueTimeout: 1 * time.Second,
}

var wgetHTTPClient = &http.Client{
	Transport: wgetTransport,
}
