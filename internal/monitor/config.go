package monitor

import (
	"crypto/rand"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"math"
	"os"
	"path/filepath"
	"strings"
	"unicode"
)

func NewID() string {
	var b [16]byte
	if _, err := rand.Read(b[:]); err != nil {
		panic(err)
	}
	return hex.EncodeToString(b[:])
}
func ValidateTarget(s string) error {
	if s == "" || len(s) > 255 || strings.HasPrefix(s, "-") || strings.ContainsAny(s, "\x00\n\r\t /\\\"'`$;&|<>()") {
		return fmt.Errorf("请输入有效的 SSH 别名或 user@host；端口请单独填写")
	}
	for _, r := range s {
		if unicode.IsSpace(r) || unicode.IsControl(r) {
			return fmt.Errorf("SSH 目标不能含空白或控制字符")
		}
	}
	return nil
}
func NormalizeConfig(c Config) (Config, error) {
	if math.IsNaN(c.IdleMemoryGB) || math.IsInf(c.IdleMemoryGB, 0) || c.IdleMemoryGB < 0 || c.IdleMemoryGB > 1024 {
		return c, fmt.Errorf("显存阈值应为 0–1024 GB")
	}
	if math.IsNaN(c.IdleUtilPercent) || math.IsInf(c.IdleUtilPercent, 0) || c.IdleUtilPercent < 0 || c.IdleUtilPercent > 100 {
		return c, fmt.Errorf("利用率阈值应为 0–100%%")
	}
	if c.Version != 1 {
		return c, fmt.Errorf("不支持的配置版本：%d", c.Version)
	}
	switch c.Interval {
	case 1, 2, 5, 10:
	default:
		return c, fmt.Errorf("刷新间隔必须为 1、2、5 或 10 秒")
	}
	switch c.Theme {
	case "system", "light", "dark":
	default:
		return c, fmt.Errorf("无效主题")
	}
	c.Servers = append([]Server{}, c.Servers...)
	seenID := map[string]bool{}
	seenTarget := map[string]bool{}
	for i, s := range c.Servers {
		s.Target = strings.TrimSpace(s.Target)
		s.Name = strings.TrimSpace(s.Name)
		if err := ValidateTarget(s.Target); err != nil {
			return c, err
		}
		if s.Port < 0 || s.Port > 65535 {
			return c, fmt.Errorf("端口应为 1–65535，或留空使用 SSH 配置")
		}
		if s.Name == "" {
			s.Name = s.Target
		}
		if len([]rune(s.Name)) > 80 {
			return c, fmt.Errorf("显示名称不能超过 80 个字符")
		}
		if strings.IndexFunc(s.Name, unicode.IsControl) >= 0 {
			return c, fmt.Errorf("显示名称不能含控制字符")
		}
		if s.ID == "" {
			s.ID = NewID()
		}
		if seenID[s.ID] {
			return c, fmt.Errorf("重复的服务器 ID")
		}
		seenID[s.ID] = true
		key := fmt.Sprintf("%s:%d", s.Target, s.Port)
		if seenTarget[key] {
			return c, fmt.Errorf("该 SSH 目标与端口已经添加")
		}
		seenTarget[key] = true
		c.Servers[i] = s
	}
	return c, nil
}
func ConfigPath() (string, error) {
	base, err := os.UserConfigDir()
	if err != nil {
		return "", err
	}
	return filepath.Join(base, "ServerMonitor", "config.json"), nil
}
func LoadConfig(path string) (Config, error) {
	b, err := os.ReadFile(path)
	if os.IsNotExist(err) {
		return DefaultConfig(), nil
	}
	if err != nil {
		return Config{}, err
	}
	c := DefaultConfig()
	if err = json.Unmarshal(b, &c); err != nil {
		return c, fmt.Errorf("配置文件无法解析，原文件已保留：%w", err)
	}
	return NormalizeConfig(c)
}
func SaveConfig(path string, c Config) error {
	b, err := json.MarshalIndent(c, "", "  ")
	if err != nil {
		return err
	}
	if err = os.MkdirAll(filepath.Dir(path), 0700); err != nil {
		return err
	}
	f, err := os.CreateTemp(filepath.Dir(path), ".config-*")
	if err != nil {
		return err
	}
	defer os.Remove(f.Name())
	if _, err = f.Write(append(b, '\n')); err != nil {
		f.Close()
		return err
	}
	if err = f.Sync(); err != nil {
		f.Close()
		return err
	}
	if err = f.Close(); err != nil {
		return err
	}
	return os.Rename(f.Name(), path)
}
