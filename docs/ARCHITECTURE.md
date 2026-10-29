# 架构与边界

## 数据流

系统 OpenSSH → 远端 `nvidia-smi --query-gpu=… --format=csv,noheader,nounits` → `internal/monitor` → Wails 事件 → React。

- `internal/monitor` 不依赖 Wails/Cocoa，包含配置、SSH 别名发现、CSV 解析、采样调度与快照。
- `app.go` 是桌面适配层，负责启动、配置保存、事件、菜单栏汇总和窗口模式。
- `tray_darwin.go/.m` 管理原生 NSStatusItem、菜单栏定位与唤醒通知；其他平台文件仅是编译边界，不代表 Windows 功能完成。
- 前端通过 `api.ts` 的类型接口调用 Wails。`demo.ts` 仅在开发模式加载，使用虚构数据；生产包不会包含演示数据。

## 配置与接口

配置版本为 1，保存在 `os.UserConfigDir()/ServerMonitor/config.json`。macOS 对应 `~/Library/Application Support/ServerMonitor/config.json`。配置使用同目录临时文件与 rename 保存，文件权限 0600。加载失败时保留原文件并提示，不自动覆盖。

`Server` 包含稳定 ID、显示名、SSH 目标、可选端口及启用状态；端口 0 表示完全使用系统 SSH 配置。应用只保存别名和偏好，不保存密码或复制私钥。

桌面接口：

- `GetInitialState()`：配置、完整快照、窗口模式、版本、启动错误。
- `SaveConfiguration(config)`：校验、原子保存后应用变更，返回规范化配置。
- `ListSSHConfigConnections()`：列出可选择的明确 Host 别名，支持 Include；不连接候选服务器。
- `RetryServer(id)`：唤醒指定服务器采样。
- `ShowMainWindow()` / `ShowMiniWindow()` / `HideWindow()` / `Quit()`：桌面操作。

`monitor:snapshot` 事件提供递增 revision、时间戳、每服务器状态、最后成功时间、重试时间、GPU 数据和 `idleGPUIndices`。空闲卡编号由后端统一计算：显存占用 < 0.5 GB 且利用率 < 5%；将十进制 GB 阈值换算为 MiB 后与原始值比较，缺失指标不纳入，编号升序。菜单栏与详情共享这个结果。前端忽略旧 revision，初始加载与事件订阅不会造成数据倒退。

服务器编号为保存数组中的位置加一，不替代稳定 ID。前端拖动只重排 `config.servers`，调用已有原子保存接口；采样工作循环继续使用原 ID，排序不重启 SSH 连接。保存失败时保留原顺序并显示错误，重启后恢复已保存顺序。

## 调度与连接

每个启用的服务器一个工作循环，全局最多 5 个同时查询，同一服务器不重叠。默认查询完成后等待 2 秒再采样；可选 1/2/5/10 秒，因此慢网络下实际周期会相应延长。连接超时 5 秒，完整查询超时 10 秒。失败等待 2/5/10/20/30 秒；手动重试立即唤醒等待中的循环。

SSH 目标始终保留别名，由系统 OpenSSH 解析 IdentityFile、Port、ProxyJump、Match 等完整语义。发现器仅枚举候选，通配符 Host 不会被当成具体服务器。查询禁用交互、端口转发和 LocalCommand，保留主机密钥验证；明确禁用配置中的 RemoteCommand，保证执行的是固定的只读查询。

macOS 使用应用实例专属的 0700 临时目录和每个工作循环独立的 ControlPath。关闭工作循环时仅向该套接字发送退出请求。超时取消仅终止本应用创建的查询进程组，避免遗留跳板机子进程；不会清理用户其他 SSH 会话。

## 状态与兼容性

状态包括 connecting/live/empty/stale/error/disabled。失败保留最近成功数据并标为过期；超过 max(10 秒, 3×刷新间隔) 未更新也视为过期。菜单栏按启用的服务器显示“#编号 N空闲”，未取得有效数据时显示“#编号 —”，暂停的服务器不参与菜单栏但仍占用其编号；全部暂停时显示“已暂停”。过期数据不提供空闲编号。空 GPU 列表代表连接成功但没有设备，不等同于 SSH 错误。

CSV 使用标准解析器；UUID 是卡片标识。N/A 与不支持的指标使用 null，界面显示 `—`。MIG 细分实例、AMD、远端 Windows 和本机 Apple GPU 不在首版范围内。

未来 Windows 工作限于托盘、窗口定位、OpenSSH 查找／进程树取消、WebView2 与安装包。保持采样和界面代码共享；不把未测试的平台宣称为支持。
