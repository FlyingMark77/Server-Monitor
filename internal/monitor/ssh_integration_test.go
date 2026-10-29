//go:build !windows

package monitor

import (
	"context"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
	"time"
)

func script(t *testing.T, body string) string {
	t.Helper()
	p := filepath.Join(t.TempDir(), "ssh-test")
	if err := os.WriteFile(p, []byte("#!/bin/sh\n"+body), 0700); err != nil {
		t.Fatal(err)
	}
	return p
}
func TestSSHSeparatesStderrAndParsesMissingMetrics(t *testing.T) {
	binary := script(t, "echo 'connection diagnostic' >&2\nprintf 'GPU-test, 0, Test GPU, 30, 100, 200, N/A\\n'\n")
	c := &SSHClient{binary: binary, dir: t.TempDir()}
	s := c.NewSession(Server{Target: "gpu-a"})
	g, err := s.Query(context.Background())
	if err != nil || len(g) != 1 || g[0].Temp != nil {
		t.Fatalf("%v %+v", err, g)
	}
}
func TestSSHNoDevicesAndDriverFailure(t *testing.T) {
	for _, tt := range []struct {
		message string
		want    string
	}{{"No devices were found", ""}, {"NVIDIA-SMI has failed because it couldn't communicate with the NVIDIA driver.", "driver"}, {"nvidia-smi: command not found", "nvidia_smi_missing"}} {
		// Single quotes in the message are passed through a quoted heredoc.
		binary := script(t, "cat >&2 <<'END'\n"+tt.message+"\nEND\nexit 1\n")
		c := &SSHClient{binary: binary, dir: t.TempDir()}
		g, err := c.NewSession(Server{Target: "gpu-a"}).Query(context.Background())
		if tt.want == "" {
			if err != nil || len(g) != 0 {
				t.Fatalf("%v %v", g, err)
			}
		} else {
			if err == nil {
				t.Fatal("expected failure")
			}
			code, _ := ErrorInfo(err)
			if code != tt.want {
				t.Fatalf("%s != %s", code, tt.want)
			}
		}
	}
}
func TestSSHDeadlineKillsOwnedProcessGroup(t *testing.T) {
	binary := script(t, "sleep 30 &\nwait\n")
	c := &SSHClient{binary: binary, dir: t.TempDir()}
	ctx, cancel := context.WithTimeout(context.Background(), 100*time.Millisecond)
	defer cancel()
	start := time.Now()
	_, err := c.NewSession(Server{Target: "gpu-a"}).Query(ctx)
	if err != context.DeadlineExceeded {
		t.Fatalf("expected deadline, got %v", err)
	}
	if time.Since(start) > 2*time.Second {
		t.Fatal("query did not cancel promptly")
	}
}
func TestOpenSSHEffectiveConfig(t *testing.T) {
	ssh, err := exec.LookPath("ssh")
	if err != nil {
		t.Skip("OpenSSH unavailable")
	}
	d := t.TempDir()
	include := filepath.Join(d, "hosts")
	root := filepath.Join(d, "config")
	os.WriteFile(include, []byte("Host lab-gpu\n HostName 192.0.2.10\n User sample\n Port 2222\n IdentityFile /tmp/example-key\n ProxyJump gateway\n"), 0600)
	os.WriteFile(root, []byte("Include \""+include+"\"\nHost *\n ServerAliveInterval 5\n"), 0600)
	choices, err := DiscoverSSHConfig(root)
	if err != nil || len(choices) != 1 || choices[0].Target != "lab-gpu" {
		t.Fatalf("%+v %v", choices, err)
	}
	out, err := exec.Command(ssh, "-G", "-F", root, choices[0].Target).Output()
	if err != nil {
		t.Fatal(err)
	}
	for _, want := range []string{"hostname 192.0.2.10", "user sample", "port 2222", "identityfile /tmp/example-key", "proxyjump gateway"} {
		if !strings.Contains(string(out), want) {
			t.Errorf("missing effective option %s", want)
		}
	}
}
