package tunnel

import (
	"net"
	"strconv"
	"strings"
	"sync"
	"time"

	"sshtui/internal/config"
)

var (
	cacheMu      sync.RWMutex
	latencyMap   = make(map[string]latencyEntry)
	cacheExpiry  = 5 * time.Second
	portCheckMap = make(map[string]portCheckEntry)
	portCheckTTL = 1 * time.Second
)

type latencyEntry struct {
	rtt       time.Duration
	err       error
	checkedAt time.Time
}

type portCheckEntry struct {
	open      bool
	checkedAt time.Time
}

// CheckLocalPort checks if a local address:port is actively listening.
func CheckLocalPort(bind string, port int) bool {
	if bind == "" {
		bind = "127.0.0.1"
	}
	addr := net.JoinHostPort(bind, strconv.Itoa(port))

	cacheMu.RLock()
	entry, exists := portCheckMap[addr]
	cacheMu.RUnlock()

	if exists && time.Since(entry.checkedAt) < portCheckTTL {
		return entry.open
	}

	conn, err := net.DialTimeout("tcp", addr, 200*time.Millisecond)
	open := (err == nil)
	if err == nil {
		_ = conn.Close()
	}

	cacheMu.Lock()
	portCheckMap[addr] = portCheckEntry{open: open, checkedAt: time.Now()}
	cacheMu.Unlock()

	return open
}

// ProbeHost measures round-trip connection time to the SSH host.
func ProbeHost(c config.Config, host string) (time.Duration, error) {
	cacheMu.RLock()
	entry, exists := latencyMap[host]
	cacheMu.RUnlock()

	if exists && time.Since(entry.checkedAt) < cacheExpiry {
		return entry.rtt, entry.err
	}

	// Resolve destination and port from config
	targetAddr := host
	port := 22
	if h, ok := c.Hosts[host]; ok {
		targetAddr = h.Address
		if h.Port != 0 {
			port = h.Port
		}
	} else if strings.Contains(host, "@") {
		parts := strings.SplitN(host, "@", 2)
		targetAddr = parts[1]
	}

	if h, p, err := net.SplitHostPort(targetAddr); err == nil {
		targetAddr = h
		if parsedPort, err := strconv.Atoi(p); err == nil && parsedPort != 0 {
			port = parsedPort
		}
	}

	addr := net.JoinHostPort(targetAddr, strconv.Itoa(port))
	start := time.Now()
	conn, err := net.DialTimeout("tcp", addr, 2*time.Second)
	rtt := time.Since(start)

	if err == nil {
		_ = conn.Close()
	}

	cacheMu.Lock()
	latencyMap[host] = latencyEntry{rtt: rtt, err: err, checkedAt: time.Now()}
	cacheMu.Unlock()

	return rtt, err
}
