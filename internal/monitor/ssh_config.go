package monitor

import (
	"bufio"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"
)

type SSHChoice struct {
	Name   string `json:"name"`
	Target string `json:"target"`
}

// Discovery only: never reinterpret HostName, IdentityFile, Match or ProxyJump.
// The selected original alias is passed to OpenSSH for full config resolution.
func DiscoverSSHConfig(root string) ([]SSHChoice, error) {
	aliases := map[string]bool{}
	seen := map[string]bool{}
	base := filepath.Dir(root)
	var read func(string, int) error
	read = func(path string, depth int) error {
		if depth > 32 {
			return fmt.Errorf("SSH Include 嵌套过深")
		}
		abs, err := filepath.Abs(path)
		if err != nil {
			return err
		}
		if seen[abs] {
			return nil
		}
		seen[abs] = true
		f, err := os.Open(abs)
		if err != nil {
			return err
		}
		defer f.Close()
		scan := bufio.NewScanner(f)
		scan.Buffer(make([]byte, 4096), 1024*1024)
		for scan.Scan() {
			tok := sshTokens(scan.Text())
			if len(tok) < 2 {
				continue
			}
			switch strings.ToLower(tok[0]) {
			case "host":
				for _, a := range tok[1:] {
					if !strings.ContainsAny(a, "*?!") && ValidateTarget(a) == nil {
						aliases[a] = true
					}
				}
			case "include":
				for _, pattern := range tok[1:] {
					if strings.HasPrefix(pattern, "~/") {
						home, _ := os.UserHomeDir()
						pattern = filepath.Join(home, pattern[2:])
					}
					if !filepath.IsAbs(pattern) {
						pattern = filepath.Join(base, pattern)
					}
					paths, err := filepath.Glob(pattern)
					if err != nil {
						return err
					}
					for _, p := range paths {
						if err = read(p, depth+1); err != nil {
							return err
						}
					}
				}
			}
		}
		return scan.Err()
	}
	if _, err := os.Stat(root); os.IsNotExist(err) {
		return []SSHChoice{}, nil
	}
	if err := read(root, 0); err != nil {
		return nil, err
	}
	names := make([]string, 0, len(aliases))
	for a := range aliases {
		names = append(names, a)
	}
	sort.Strings(names)
	out := make([]SSHChoice, 0, len(names))
	for _, a := range names {
		out = append(out, SSHChoice{Name: a, Target: a})
	}
	return out, nil
}
func sshTokens(line string) []string {
	var out []string
	var b strings.Builder
	var quote rune
	escaped := false
	flush := func() {
		if b.Len() > 0 {
			out = append(out, b.String())
			b.Reset()
		}
	}
	for _, r := range line {
		if escaped {
			b.WriteRune(r)
			escaped = false
			continue
		}
		if r == '\\' {
			escaped = true
			continue
		}
		if quote != 0 {
			if r == quote {
				quote = 0
			} else {
				b.WriteRune(r)
			}
			continue
		}
		if r == '"' || r == '\'' {
			quote = r
			continue
		}
		if r == '#' {
			break
		}
		if r == ' ' || r == '\t' || (r == '=' && len(out) < 2) {
			flush()
		} else {
			b.WriteRune(r)
		}
	}
	flush()
	return out
}
