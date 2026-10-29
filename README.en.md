# Server Monitor

**Monitor GPUs across your servers from the macOS menu bar.**

[简体中文](README.md) · English

[Download](https://github.com/FlyingMark77/Server-Monitor/releases/latest) · [Releases](https://github.com/FlyingMark77/Server-Monitor/releases) · [Report an issue](https://github.com/FlyingMark77/Server-Monitor/issues)

![Server Monitor example dashboard](docs/preview.png)

## Features

- **Multi-server monitoring** — Track NVIDIA GPU utilization, memory usage, and temperature over SSH.
- **Menu bar summary** — View idle GPU counts by server number at a glance.
- **Custom idle thresholds** — Set memory and utilization limits and see the indices of available GPUs.
- **Drag-and-drop ordering** — Reorder servers using their numbered handles; the menu bar follows the saved order.
- **Flexible connections** — Import local SSH aliases with support for keys, SSH Agent, custom ports, and jump hosts.
- **Independent polling** — Each server refreshes and retries independently while retaining its last successful sample.
- **Compact interface** — Switch between a menu bar popup and a dashboard, with light, dark, and system themes.

## Installation

Download the Apple Silicon `.dmg` from [Releases](https://github.com/FlyingMark77/Server-Monitor/releases/latest), open it, and drag **ServerMonitor.app** into **Applications**.

| Environment | Requirements |
| --- | --- |
| Local machine | Apple Silicon Mac; macOS 12+ build target |
| Remote servers | Linux, NVIDIA GPUs, and a working `nvidia-smi` |
| Connection | Non-interactive SSH access from your Mac |

## Quick start

1. Check that your SSH connection works:

   ```bash
   ssh gpu-lab nvidia-smi
   ```

2. Open Server Monitor and click **添加服务器 (Add Server)**. Select an alias from your SSH configuration or enter a connection target manually.
3. Click the menu bar item to view GPU metrics. Use the arrow in the top-right corner to open the dashboard.
4. Drag the numbered handle on a server card to change its position. Use the card footer to retry, pause, or edit a connection.

Closing the window keeps monitoring active in the menu bar. Select **退出应用 (Quit)** in settings to stop the app.

## Settings

| Setting | Options |
| --- | --- |
| Refresh interval | 1, 2, 5, or 10 seconds |
| Appearance | System, light, or dark |
| Idle memory threshold | 0–1024 GB; default: 0.5 GB |
| Idle utilization threshold | 0–100%; default: 5% |

A GPU is idle when its memory usage and utilization are **both strictly below** their thresholds. Enter values under **设置 → 空闲判定 (Settings → Idle criteria)** and click **应用 (Apply)** to update the menu bar and dashboard. GB uses decimal units.

Configuration is saved automatically to:

```text
~/Library/Application Support/ServerMonitor/config.json
```

## Development

Built with **Go · Wails v2 · React · TypeScript**.

Install Go 1.25.x, Node.js 22+, npm, and Xcode Command Line Tools, then run:

```bash
git clone https://github.com/FlyingMark77/Server-Monitor.git
cd Server-Monitor
go install github.com/wailsapp/wails/v2/cmd/wails@v2.11.0
export PATH="$(go env GOPATH)/bin:$PATH"
npm ci --prefix frontend
wails dev
```

Test and package:

```bash
npm --prefix frontend run build
go test -race ./...
go vet ./...
./scripts/package-macos.sh
```

Build artifacts are written to `dist/`. Pushing a `v*` version tag triggers GitHub Actions to run tests, build the DMG and SHA-256 checksum, and publish a release.

Further reading: [Build and testing guide](docs/TESTING.md) · [Architecture](docs/ARCHITECTURE.md) (Chinese).

## Roadmap

- Windows system tray integration and installer.
- macOS Developer ID signing and notarization.

## Acknowledgments and license

Based on [NVSmiBar](https://github.com/xyhacker404/NVSmiBar), created by Xingyu Hu. Thanks to the original project for its Go, Wails, and menu bar application foundation.

Released under the [MIT License](LICENSE). See [NOTICE](NOTICE) for upstream revision and copyright details.
