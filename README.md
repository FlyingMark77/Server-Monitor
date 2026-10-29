# Server Monitor

简洁的 macOS 多服务器 GPU 菜单栏监控。通过本机已有的免密 SSH 连接 Linux 服务器，查询 NVIDIA GPU 利用率、显存与温度，无需安装远端监控服务。

## 来源与许可

基于 [NVSmiBar](https://github.com/xyhacker404/NVSmiBar)，保留完整 Git 历史和原作者 Xingyu Hu 的 MIT 许可。采用的上游提交：`7d56394265d0dadd75fe5b19fe8da229dbdb6218`。

本项目沿用 Go / Wails v2 / React / TypeScript 及菜单栏交互，增加多服务器独立采样、汇总、配置持久化与中文界面。使用独立应用标识，不从上游自动更新。

## Git

- `main`：本项目开发分支；`upstream`：NVSmiBar 源仓库。
- 尚未配置个人 GitHub `origin`，不会自动发布或推送。
- 源码与锁文件入库；安装包、依赖、本地配置、日志和密钥不入库。
- 如需发布，在 GitHub 创建空仓库后添加自己的 `origin` 并推送。完整历史包含上游作者提交，请保留来源说明。

构建、运行与验证说明随实现补充。首版目标为 Apple Silicon / macOS 12+；Windows 适配随后补充。
