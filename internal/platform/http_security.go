package platform

import (
	"net"
	"net/http"
	"net/netip"
	"strings"
)

// clientIP accepts a proxy-supplied address only after explicit opt-in and only from the loopback proxy.
// Nginx must overwrite X-Real-IP with $remote_addr before the deployment enables this option.
func (a *App) clientIP(r *http.Request) string {
	peer, _, err := net.SplitHostPort(r.RemoteAddr)
	if err != nil {
		peer = r.RemoteAddr
	}
	address, err := netip.ParseAddr(peer)
	if err != nil {
		return peer
	}
	address = address.Unmap()
	if a.Config.TrustLoopbackProxy && address.IsLoopback() {
		forwarded, err := netip.ParseAddr(strings.TrimSpace(r.Header.Get("X-Real-IP")))
		if err == nil && forwarded.Zone() == "" && !forwarded.IsUnspecified() {
			return forwarded.Unmap().String()
		}
	}
	return address.String()
}

// privateStaticPath prevents accidental publication of dotfiles, credentials, backups and local database files.
func privateStaticPath(requestPath string) bool {
	for _, segment := range strings.Split(strings.ReplaceAll(requestPath, "\\", "/"), "/") {
		name := strings.ToLower(segment)
		if strings.HasPrefix(name, ".") && name != ".well-known" {
			return true
		}
		if name == "development-state.json" || strings.Contains(name, ".env.") {
			return true
		}
		for _, suffix := range []string{".env", ".pem", ".key", ".p12", ".pfx", ".bak", ".sql", ".sqlite", ".sqlite3", ".db"} {
			if strings.HasSuffix(name, suffix) {
				return true
			}
		}
	}
	return false
}
