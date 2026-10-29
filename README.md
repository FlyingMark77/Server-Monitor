# Server Monitor

**在 macOS 菜单栏查看多台服务器的 GPU 状态。**

简体中文 · [English](README.en.md)

[下载安装包](https://github.com/FlyingMark77/Server-Monitor/releases/latest) · [更新记录](https://github.com/FlyingMark77/Server-Monitor/releases) · [问题反馈](https://github.com/FlyingMark77/Server-Monitor/issues)

![Server Monitor 示例界面](docs/preview.png)

## 功能

- **多服务器监控**：通过 SSH 实时获取 NVIDIA GPU 利用率、显存占用与温度。
- **菜单栏摘要**：按服务器编号显示空闲卡数，如 `#1 2空闲 · #2 3空闲`。
- **自定义空闲标准**：设置显存和利用率阈值，在详情中查看符合条件的 GPU 编号。
- **拖动排序**：拖动服务器卡片的编号手柄调整顺序，自动保存并同步菜单栏编号。
- **灵活连接**：导入本机 SSH 别名，支持密钥、SSH Agent、自定义端口与跳板机。
- **独立刷新**：每台服务器独立采样、重试，保留最近一次成功的数据。
- **简洁界面**：菜单栏弹窗与总览窗口，支持浅色、深色和跟随系统主题。

## 安装

从 [Releases](https://github.com/FlyingMark77/Server-Monitor/releases/latest) 下载 Apple Silicon 版本的 `.dmg`，打开后将 **ServerMonitor.app** 拖入 **Applications**。

| 环境 | 要求 |
| --- | --- |
| 本机 | Apple Silicon Mac；构建目标 macOS 12+ |
| 服务器 | Linux、NVIDIA GPU，以及可用的 `nvidia-smi` |
| 连接 | 本机可免交互登录的 SSH 连接 |

## 快速开始

1. 在终端确认服务器连接可用：

   ```bash
   ssh gpu-lab nvidia-smi
   ```

2. 打开 Server Monitor，点击 **添加服务器**，从 SSH 配置中选择别名，或手动填写连接目标。
3. 点击菜单栏查看各台服务器的 GPU 状态；点击右上角箭头打开总览。
4. 拖动卡片左上角的编号手柄调整顺序。卡片底部提供重试、暂停和编辑操作。

关闭窗口后，应用继续在菜单栏运行；通过设置中的 **退出应用** 结束监控。

## 设置

| 设置 | 选项 |
| --- | --- |
| 刷新间隔 | 1、2、5、10 秒 |
| 外观 | 跟随系统、浅色、深色 |
| 空闲显存阈值 | 0–1024 GB，默认 0.5 GB |
| 空闲利用率阈值 | 0–100%，默认 5% |

GPU 的显存占用和利用率须**同时低于**对应阈值，才计为空闲。在 **设置 → 空闲判定** 中填写数值，点击 **应用** 即可更新菜单栏和详情。GB 使用十进制单位。

配置自动保存在：

```text
~/Library/Application Support/ServerMonitor/config.json
```

## 开发

技术栈：**Go · Wails v2 · React · TypeScript**。

准备 Go 1.25.x、Node.js 22+、npm 和 Xcode Command Line Tools，然后执行：

```bash
git clone https://github.com/FlyingMark77/Server-Monitor.git
cd Server-Monitor
go install github.com/wailsapp/wails/v2/cmd/wails@v2.11.0
export PATH="$(go env GOPATH)/bin:$PATH"
npm ci --prefix frontend
wails dev
```

测试与打包：

```bash
npm --prefix frontend run build
go test -race ./...
go vet ./...
./scripts/package-macos.sh
```

产物输出到 `dist/`。推送 `v*` 版本标签时，GitHub Actions 自动运行测试、生成 DMG 和 SHA-256 校验文件，并发布 Release。

更多说明：[构建与测试](docs/TESTING.md) · [架构设计](docs/ARCHITECTURE.md)

## 路线图

- Windows 托盘与安装包适配。
- macOS Developer ID 签名与公证。

## 致谢与许可

本项目基于 [NVSmiBar](https://github.com/xyhacker404/NVSmiBar)，由 Xingyu Hu 创建。感谢原项目提供的 Go、Wails 与菜单栏应用基础。

采用 [MIT License](LICENSE)。上游版本与版权信息见 [NOTICE](NOTICE)。
