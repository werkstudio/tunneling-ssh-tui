package detect

import "testing"

func TestParseSS(t *testing.T) {
	out := `LISTEN 0      4096         0.0.0.0:8891       0.0.0.0:*
LISTEN 0      511          0.0.0.0:20128      0.0.0.0:*    users:(("next-server (v1",pid=1826801,fd=21))
LISTEN 0      4096       127.0.0.1:5432       0.0.0.0:*
LISTEN 0      4096            [::]:8891          [::]:*
LISTEN 0      128             [::]:22            [::]:*
`
	got := Parse(out)
	if len(got) != 4 {
		t.Fatalf("want 4 unique ports, got %d: %+v", len(got), got)
	}
	want := []int{22, 5432, 8891, 20128}
	for i, p := range got {
		if p.Port != want[i] {
			t.Errorf("idx %d: want port %d got %d", i, want[i], p.Port)
		}
	}
	if got[3].Process != "next-server (v1" {
		t.Errorf("process = %q", got[3].Process)
	}
	if !got[1].Loopback() || got[1].Target() != "localhost" {
		t.Errorf("5432 should be loopback: %+v", got[1])
	}
}

func TestParseNetstat(t *testing.T) {
	out := `Active Internet connections (only servers)
Proto Recv-Q Send-Q Local Address           Foreign Address         State       PID/Program name
tcp        0      0 0.0.0.0:22              0.0.0.0:*               LISTEN      812/sshd
tcp6       0      0 :::3000                 :::*                    LISTEN      1203/node
tcp        0      0 10.0.0.5:9000           0.0.0.0:*               LISTEN      -
`
	got := Parse(out)
	if len(got) != 3 {
		t.Fatalf("got %+v", got)
	}
	if got[0].Process != "sshd" || got[1].Port != 3000 || got[1].Process != "node" {
		t.Errorf("unexpected: %+v", got)
	}
	if got[2].Target() != "10.0.0.5" {
		t.Errorf("target = %q", got[2].Target())
	}
}

func TestParseLsof(t *testing.T) {
	out := `COMMAND   PID USER   FD   TYPE DEVICE SIZE/OFF NODE NAME
node     4242 arry   23u  IPv4 0x1234      0t0  TCP *:3000 (LISTEN)
postgres  991 arry    7u  IPv4 0x5678      0t0  TCP 127.0.0.1:5432 (LISTEN)
`
	got := Parse(out)
	if len(got) != 2 || got[0].Process != "node" || got[1].Port != 5432 {
		t.Fatalf("got %+v", got)
	}
}

func TestParseGarbage(t *testing.T) {
	if got := Parse("bash: ss: command not found\n\n"); len(got) != 0 {
		t.Fatalf("want none, got %+v", got)
	}
}
