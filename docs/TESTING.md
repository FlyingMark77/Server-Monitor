# 构建与测试

## 环境

macOS 上安装 Xcode Command Line Tools、Go 1.25.x、Node.js 22+ 和 npm。构建工具不需要安装到被监控服务器。

```sh
go install github.com/wailsapp/wails/v2/cmd/wails@v2.11.0
export PATH="$(go env GOPATH)/bin:$PATH"
cd frontend
npm ci
npm run build
cd ..
go test -race ./...
go vet ./...
./scripts/package-macos.sh
```

Go 1.25 工具链用于保持 macOS 12 目标；升级工具链前检查其最低系统版本。实际 Mach-O 的 `minos` 与 Info.plist 均应为 12.0。macOS 新 SDK 构建需要链接 UniformTypeIdentifiers，已在平台代码中声明。

输出位于 `dist/`：Apple Silicon `.app`、压缩 `.dmg` 和 SHA-256 文件。打包脚本先在临时 APFS 目录清理元数据并签名，再生成 DMG；也清理外置磁盘上的 AppleDouble 旁路文件。生成的程序是本地 ad-hoc 签名，未进行 Apple 公证。对外公开分发前应另行配置 Developer ID 签名及公证。

## 自动化测试

后端测试覆盖 CSV／缺失值／异常数据、SSH 别名与 Include、OpenSSH `-G` 实际配置解析、配置保存与重复项、单台阻塞不影响其他台、删除后的迟到结果、重试恢复、5 并发上限、过期计数，以及查询超时后的进程组取消。

前端测试使用虚构数据，不接触真实服务器。先启动预览：

```sh
cd frontend
npm run dev -- --host 127.0.0.1 --port 34115 --strictPort
```

另一个终端运行：

```sh
cd frontend
npx playwright install chromium
npm run test:ui
npm run format:check
```

也可设置 `CHROME_BIN` 指向已有 Chrome。测试涵盖主题、刷新间隔、SSH 勾选、手动添加、编辑、暂停、删除、迷你窗口与浏览器错误；截图保存在被 Git 忽略的 `.local/qa/`。

## 真实服务器与稳定性

使用与桌面应用完全相同的采集与调度核心：

```sh
go run ./cmd/monitor-check -hosts gpu-lab-a,gpu-lab-b -duration 1h
```

每分钟输出匿名化的采样次数、错误事件数与 GPU 数。退出时清理专属 SSH 连接。`-hosts` 只填写已获准查询的本机 SSH 别名；原始运行结果保留在本地，不提交真实连接信息。

可显式启用真实连接恢复测试，默认测试套件会跳过它：

```sh
SERVER_MONITOR_TEST_HOSTS=gpu-lab-a,gpu-lab-b go test -race -v ./internal/monitor -run TestLiveSSHReconnect -count=1
```

它只关闭测试自己创建的 SSH 复用连接，随后重新读取 GPU，核对 UUID 和设备数，并检查套接字清理。

桌面验收另需检查：从 Finder 启动、菜单栏摘要、配置重启恢复、窗口隐藏后继续采样、退出后的连接清理、主机故障和睡眠恢复。自动化中的数据过期与重试恢复测试不能替代真实的硬件睡眠验证。

具体已完成与未完成的项目记录在 `VERIFICATION-0.1.3.md`，首版连续采样记录保留在 `VERIFICATION.md`，不把构建成功当成全部平台验证通过。
