package clearbox

import (
	"fmt"
	"math"
	"net"
	"net/http"
	"net/netip"
	"net/url"
	"strings"
	"sync"
	"time"

	"golang.org/x/time/rate"
)

type visitor struct {
	limiter *rate.Limiter
	seen    time.Time
}

type traffic struct {
	mu      sync.Mutex
	global  *rate.Limiter
	writes  *rate.Limiter
	clients map[netip.Addr]*visitor
	proxies []netip.Prefix
	perIP   rate.Limit
}

func configure(cfg *Config) (*traffic, error) {
	if cfg.PublicURL != "" {
		u, err := url.Parse(cfg.PublicURL)
		if err != nil || u.Hostname() == "" || u.User != nil || u.RawQuery != "" || u.Fragment != "" || (u.Path != "" && u.Path != "/") ||
			(u.Scheme != "https" && !(u.Scheme == "http" && localHost(u.Host))) {
			return nil, fmt.Errorf("public-url must be an HTTPS origin (loopback HTTP is allowed for tests)")
		}
		cfg.PublicURL = (&url.URL{Scheme: u.Scheme, Host: u.Host}).String()
	}
	if cfg.RequestsPerSecond == 0 {
		cfg.RequestsPerSecond = 10
	}
	if cfg.RequestsPerSecond < 1 || cfg.RequestsPerSecond > 1000 || math.IsNaN(cfg.RequestsPerSecond) {
		return nil, fmt.Errorf("rate-limit must be between 1 and 1000 requests per second per IP")
	}
	if cfg.MaxOperations == 0 {
		cfg.MaxOperations = 2000
	}
	if cfg.MaxOperations < 1 {
		return nil, fmt.Errorf("max-operations must be positive")
	}
	t := &traffic{global: rate.NewLimiter(50, 100), writes: rate.NewLimiter(2, 4), clients: make(map[netip.Addr]*visitor), perIP: rate.Limit(cfg.RequestsPerSecond)}
	if cfg.SealDelay > 0 {
		// Public writes use the sealing interval as their admission pace too;
		// merely delaying receipts would still allow the database to fill fast.
		t.writes = rate.NewLimiter(rate.Every(cfg.SealDelay), 1)
	}
	for _, raw := range cfg.TrustedProxies {
		prefix, err := netip.ParsePrefix(strings.TrimSpace(raw))
		if err != nil {
			return nil, fmt.Errorf("invalid trusted proxy CIDR %q: %w", raw, err)
		}
		t.proxies = append(t.proxies, prefix)
	}
	return t, nil
}

func (t *traffic) trusted(ip netip.Addr) bool {
	for _, prefix := range t.proxies {
		if prefix.Contains(ip) {
			return true
		}
	}
	return false
}

func (t *traffic) clientIP(r *http.Request) netip.Addr {
	host, _, _ := net.SplitHostPort(r.RemoteAddr)
	peer, _ := netip.ParseAddr(host)
	peer = peer.Unmap()
	ip := peer
	// Walk from the nearest hop; untrusted peers cannot select their own bucket.
	hops := strings.Split(r.Header.Get("X-Forwarded-For"), ",")
	for i := len(hops) - 1; i >= 0 && t.trusted(ip); i-- {
		next, err := netip.ParseAddr(strings.TrimSpace(hops[i]))
		if err != nil {
			return peer
		}
		ip = next.Unmap()
	}
	return ip
}

func (t *traffic) allow(r *http.Request) bool {
	t.mu.Lock()
	defer t.mu.Unlock()
	ip := t.clientIP(r)
	v := t.clients[ip]
	if v == nil {
		if len(t.clients) >= 2048 {
			for key, old := range t.clients {
				if time.Since(old.seen) > 10*time.Minute {
					delete(t.clients, key)
				}
			}
		}
		if len(t.clients) >= 2048 {
			return false
		}
		v = &visitor{limiter: rate.NewLimiter(t.perIP, int(2*t.perIP))}
		t.clients[ip] = v
	}
	v.seen = time.Now()
	return v.limiter.Allow() && t.global.Allow()
}

func limited(w http.ResponseWriter, message string) {
	if w.Header().Get("Retry-After") == "" {
		w.Header().Set("Retry-After", "1")
	}
	fail(w, 429, -32008, message, "")
}
