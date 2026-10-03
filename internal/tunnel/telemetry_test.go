package tunnel

import (
	"net"
	"strconv"
	"testing"
	"time"

	"sshtui/internal/config"
)

func TestCheckLocalPort(t *testing.T) {
	l, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatalf("failed to listen: %v", err)
	}
	defer l.Close()

	_, portStr, _ := net.SplitHostPort(l.Addr().String())
	port, _ := strconv.Atoi(portStr)

	if !CheckLocalPort("127.0.0.1", port) {
		t.Errorf("expected port %d to be listening", port)
	}

	// Test default bind address (empty string defaults to 127.0.0.1)
	if !CheckLocalPort("", port) {
		t.Errorf("expected empty bind to default to 127.0.0.1 and be listening")
	}

	// Verify caching: closing listener shouldn't immediately change status within 1s TTL
	_ = l.Close()
	if !CheckLocalPort("127.0.0.1", port) {
		t.Errorf("expected cached listening status for port %d within TTL", port)
	}
	lUnused, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatalf("failed to find free port: %v", err)
	}
	unusedAddr := lUnused.Addr().String()
	lUnused.Close()
	_, unusedPortStr, _ := net.SplitHostPort(unusedAddr)
	unusedPort, _ := strconv.Atoi(unusedPortStr)

	if CheckLocalPort("127.0.0.1", unusedPort) {
		t.Errorf("expected unused port %d to not be listening", unusedPort)
	}
}

func TestProbeHost(t *testing.T) {
	l, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatalf("failed to listen: %v", err)
	}
	defer l.Close()

	_, portStr, _ := net.SplitHostPort(l.Addr().String())
	port, _ := strconv.Atoi(portStr)

	cfg := config.Config{
		Hosts: map[string]config.HostCfg{
			"testhost": {Address: "127.0.0.1", Port: port},
		},
	}

	rtt, err := ProbeHost(cfg, "testhost")
	if err != nil {
		t.Fatalf("unexpected error probing host: %v", err)
	}
	if rtt <= 0 {
		t.Errorf("expected positive rtt, got %v", rtt)
	}

	// Verify caching: subsequent probe within expiry should return cached result
	rttCached, errCached := ProbeHost(cfg, "testhost")
	if errCached != nil {
		t.Fatalf("unexpected error probing cached host: %v", errCached)
	}
	if rttCached != rtt {
		t.Errorf("expected cached rtt %v, got %v", rtt, rttCached)
	}

	// Test host with user@host:port syntax when not present in cfg.Hosts
	lUser, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatalf("failed to listen: %v", err)
	}
	defer lUser.Close()
	_, userPortStr, _ := net.SplitHostPort(lUser.Addr().String())

	cfgEmpty := config.Config{}
	userHost := "user@127.0.0.1:" + userPortStr
	rttUser, errUser := ProbeHost(cfgEmpty, userHost)
	if errUser != nil {
		t.Fatalf("unexpected error probing %s: %v", userHost, errUser)
	}
	if rttUser <= 0 {
		t.Errorf("expected positive rtt for %s, got %v", userHost, rttUser)
	}

	// Unreachable host should return an error
	lUnused, err := net.Listen("tcp", "127.0.0.1:0")
	if err == nil {
		unusedAddr := lUnused.Addr().String()
		lUnused.Close()
		_, pStr, _ := net.SplitHostPort(unusedAddr)
		p, _ := strconv.Atoi(pStr)
		cfgUnused := config.Config{
			Hosts: map[string]config.HostCfg{
				"closedhost": {Address: "127.0.0.1", Port: p},
			},
		}
		_, errProbe := ProbeHost(cfgUnused, "closedhost")
		if errProbe == nil {
			t.Errorf("expected probe error on closed port, got nil")
		}
	}
}

func TestRuntimeSnapshotAndTelemetry(t *testing.T) {
	// Stopped runtime
	r := &Runtime{}
	snap := r.Snapshot()
	if snap.Status != Stopped {
		t.Errorf("expected status Stopped, got %v", snap.Status)
	}
	if snap.LastError != "" {
		t.Errorf("expected empty LastError, got %q", snap.LastError)
	}
	if snap.RetryCount != 0 {
		t.Errorf("expected 0 RetryCount, got %d", snap.RetryCount)
	}
	if snap.RetryIn != 0 {
		t.Errorf("expected 0 RetryIn, got %v", snap.RetryIn)
	}
	if snap.Uptime != 0 {
		t.Errorf("expected 0 Uptime, got %v", snap.Uptime)
	}

	// Runtime with reconnecting state, retryCount, lastError, and future retry time
	futureRetry := time.Now().Add(5 * time.Second)
	rRecon := &Runtime{
		status:      Reconnecting,
		lastError:   "connection refused",
		retryCount:  3,
		nextRetryAt: futureRetry,
	}
	snapRecon := rRecon.Snapshot()
	if snapRecon.Status != Reconnecting {
		t.Errorf("expected status Reconnecting, got %v", snapRecon.Status)
	}
	if snapRecon.LastError != "connection refused" {
		t.Errorf("expected LastError 'connection refused', got %q", snapRecon.LastError)
	}
	if snapRecon.RetryCount != 3 {
		t.Errorf("expected RetryCount 3, got %d", snapRecon.RetryCount)
	}
	if snapRecon.RetryIn <= 0 || snapRecon.RetryIn > 5*time.Second {
		t.Errorf("expected RetryIn between 0 and 5s, got %v", snapRecon.RetryIn)
	}

	// Runtime in Running status computes Uptime
	rRunning := &Runtime{
		status: Running,
		since:  time.Now().Add(-10 * time.Second),
	}
	snapRunning := rRunning.Snapshot()
	if snapRunning.Status != Running {
		t.Errorf("expected status Running, got %v", snapRunning.Status)
	}
	if snapRunning.Uptime < 9*time.Second {
		t.Errorf("expected Uptime >= 9s, got %v", snapRunning.Uptime)
	}
}
