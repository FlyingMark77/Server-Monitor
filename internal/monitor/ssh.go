package monitor

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strconv"
	"strings"
	"sync"
	"time"
)

type Sampler interface {
	Query(context.Context) ([]GPU, error)
	Close()
}
type Factory func(Server) Sampler

type SSHClient struct {
	dir    string
	binary string
}

func NewSSHClient() (*SSHClient, error) {
	binary := "/usr/bin/ssh"
	if runtime.GOOS != "darwin" {
		var err error
		binary, err = exec.LookPath("ssh")
		if err != nil {
			return nil, fmt.Errorf("未找到系统 OpenSSH 客户端")
		}
	}
	// Short private path keeps Unix domain sockets under macOS's path limit.
	dir, err := os.MkdirTemp("", "sm-")
	if err != nil {
		return nil, err
	}
	return &SSHClient{dir: dir, binary: binary}, nil
}
func (c *SSHClient) NewSession(s Server) Sampler {
	return &sshSession{client: c, server: s, path: filepath.Join(c.dir, NewID()[:16])}
}
func (c *SSHClient) Close() { os.RemoveAll(c.dir) }

type sshSession struct {
	client *SSHClient
	server Server
	path   string
	once   sync.Once
}

func (s *sshSession) args() []string {
	args := []string{"-T", "-o", "BatchMode=yes", "-o", "StrictHostKeyChecking=yes", "-o", "ConnectTimeout=5", "-o", "ConnectionAttempts=1", "-o", "ServerAliveInterval=3", "-o", "ServerAliveCountMax=2", "-o", "ClearAllForwardings=yes", "-o", "PermitLocalCommand=no", "-o", "RequestTTY=no", "-o", "RemoteCommand=none"}
	if runtime.GOOS == "darwin" {
		args = append(args, "-o", "ControlMaster=auto", "-o", "ControlPersist=30", "-o", "ControlPath="+s.path)
	} else {
		args = append(args, "-o", "ControlMaster=no")
	}
	if s.server.Port > 0 {
		args = append(args, "-p", strconv.Itoa(s.server.Port))
	}
	return args
}

// limitWriter prevents an unexpected remote response from growing memory without bound.
type limitWriter struct {
	b   bytes.Buffer
	max int
}

func (w *limitWriter) Write(p []byte) (int, error) {
	if w.b.Len()+len(p) > w.max {
		return 0, fmt.Errorf("远端输出超过大小限制")
	}
	return w.b.Write(p)
}
func (s *sshSession) Query(ctx context.Context) ([]GPU, error) {
	if err := ValidateTarget(s.server.Target); err != nil {
		return nil, err
	}
	args := append(s.args(), "--", s.server.Target, GPUCommand)
	cmd := exec.CommandContext(ctx, s.client.binary, args...)
	prepareCommand(cmd)
	stdout := &limitWriter{max: 1024 * 1024}
	stderr := &limitWriter{max: 16384}
	cmd.Stdout = stdout
	cmd.Stderr = stderr
	err := cmd.Run()
	if ctx.Err() != nil {
		return nil, ctx.Err()
	}
	if err != nil {
		msg := strings.TrimSpace(stderr.b.String() + " " + stdout.b.String())
		if strings.Contains(strings.ToLower(msg), "no devices were found") {
			return []GPU{}, nil
		}
		if len(msg) > 2000 {
			msg = msg[:2000]
		}
		return nil, fmt.Errorf("ssh: %s (%w)", msg, err)
	}
	return ParseGPUs(stdout.b.String())
}
func (s *sshSession) Close() {
	s.once.Do(func() {
		if runtime.GOOS != "darwin" {
			return
		}
		ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
		defer cancel()
		args := []string{"-o", "BatchMode=yes", "-o", "ControlPath=" + s.path, "-O", "exit", "--", s.server.Target}
		cmd := exec.CommandContext(ctx, s.client.binary, args...)
		prepareCommand(cmd)
		_ = cmd.Run()
		_ = os.Remove(s.path)
	})
}
func ErrorInfo(err error) (string, string) {
	if errors.Is(err, context.DeadlineExceeded) {
		return "timeout", "查询超时，请检查网络或 VPN"
	}
	msg := strings.ToLower(err.Error())
	switch {
	case strings.Contains(msg, "permission denied"):
		return "auth_failed", "SSH 认证失败，请确认本机可免密连接此别名"
	case strings.Contains(msg, "host key verification failed"), strings.Contains(msg, "remote host identification has changed"):
		return "host_key", "主机密钥未受信任或已变化，请在终端核对后连接"
	case strings.Contains(msg, "could not resolve hostname"):
		return "dns", "无法解析 SSH 别名或主机名"
	case strings.Contains(msg, "connection refused"):
		return "refused", "服务器拒绝连接，请检查 SSH 服务与端口"
	case strings.Contains(msg, "timed out"):
		return "timeout", "连接超时，请检查网络或 VPN"
	case strings.Contains(msg, "nvidia-smi") && strings.Contains(msg, "not found"):
		return "nvidia_smi_missing", "远端找不到 nvidia-smi"
	case strings.Contains(msg, "failed to initialize nvml"), strings.Contains(msg, "nvidia-smi has failed"):
		return "driver", "远端 NVIDIA 驱动不可用"
	default:
		return "query_failed", err.Error()
	}
}
