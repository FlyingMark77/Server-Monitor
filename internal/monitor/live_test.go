package monitor

import (
	"context"
	"os"
	"os/exec"
	"runtime"
	"strconv"
	"strings"
	"testing"
	"time"
)

// Explicit opt-in only: the default test suite never connects to user servers.
func TestLiveSSHReconnect(t *testing.T) {
	hosts := os.Getenv("SERVER_MONITOR_TEST_HOSTS")
	if hosts == "" {
		t.Skip("set SERVER_MONITOR_TEST_HOSTS to explicitly authorize live read-only queries")
	}
	if runtime.GOOS != "darwin" {
		t.Skip("macOS multiplexing test")
	}
	for i, host := range strings.Split(hosts, ",") {
		t.Run(strconv.Itoa(i+1), func(t *testing.T) {
			client, err := NewSSHClient()
			if err != nil {
				t.Fatal(err)
			}
			defer client.Close()
			session := client.NewSession(Server{Target: strings.TrimSpace(host)}).(*sshSession)
			defer session.Close()
			query := func() []GPU {
				ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
				defer cancel()
				g, err := session.Query(ctx)
				if err != nil {
					t.Fatal(err)
				}
				return g
			}
			before := query()
			if len(before) == 0 {
				t.Fatal("expected at least one GPU for live verification")
			}
			// Close ONLY this test's private master, then exercise automatic reconnection.
			ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
			err = exec.CommandContext(ctx, client.binary, "-S", session.path, "-O", "exit", "--", session.server.Target).Run()
			cancel()
			if err != nil {
				t.Fatal(err)
			}
			after := query()
			if len(before) != len(after) {
				t.Fatalf("GPU count changed: %d -> %d", len(before), len(after))
			}
			for j := range before {
				if before[j].UUID != after[j].UUID {
					t.Fatal("device identity changed")
				}
			}
			session.Close()
			if _, err = os.Stat(session.path); !os.IsNotExist(err) {
				t.Fatalf("control socket remains: %v", err)
			}
			t.Logf("%d GPUs: query, private connection restart, reconnect and cleanup passed", len(after))
		})
	}
}
