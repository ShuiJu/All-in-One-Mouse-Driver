# 验证报告

2026-09-13，Windows x64；实物：LAMZU INCA、VAXEE E1、VAXEE NP-01S V2。

本机三台设备的核心设置写入、硬件读回、前台窗口自动联动和原值恢复均已通过。以下明确区分实物结果、自动测试与未覆盖项目。

## 实物参数组合：57 / 57 通过

| 设备 | 组合数 | 覆盖范围 | 原值恢复 |
|---|---:|---|---|
| LAMZU INCA | 25 | 3 种性能模式 × 4 档回报率 × Motion Sync 开/关；额外 1800 秒休眠 | 通过 |
| VAXEE NP-01S V2 | 16 | 标准 1K、竞技 1K/2K/4K × Motion Sync 开/关 × 两种轨迹 | 通过 |
| VAXEE E1 | 16 | 同上 | 通过 |

完整组合中的 LAMZU 休眠设为 600 秒；前一轮代表性矩阵另外验证了 30、60、300 秒。没有把所有 0–65535 秒的输入值都逐秒实测。

每组调用与正式程序相同的设备适配器，下发后逐字段读取硬件并比较。证据：`research/hardware-full-matrix.json`，全部 Error 为空、Restored 为 true。前一轮 12 组代表性矩阵也全部通过：`research/hardware-verification-fixed.json`。

测试结束后另开只读进程，对三台设备再次读取并与测试前基线比较，三台全部一致（`research/hardware-restored-final.json`）。该文件来自只读模式，其中 Restored 字段不适用；恢复结论来自独立逐字段比对。

恢复的原值：

- LAMZU：办公 / 1000 Hz / Motion Sync 开 / 休眠 30 秒。
- NP-01S V2：标准 / 1000 Hz / Motion Sync 关 / 顺滑灵敏。
- E1：竞技 / 4000 Hz / Motion Sync 关 / 稳定易控。

## 真实前台联动：通过

使用独立的 `build/integration/data` 配置，以实际运行的 `A HUB.exe` 作为关联程序，注册真正的 Windows 前台窗口事件。

1. 程序启动，三台鼠标实际读回日常场景。
2. 用 Computer Use 将 A HUB 窗口置于前台。三台均实际切到游戏场景：LAMZU 20,000 FPS / 4K / MS 开 / 300 秒；VAXEE 竞技 / 4K / MS 开 / 稳定易控。
3. 切回 Mouse Hub 窗口。三台均恢复测试配置的日常场景：1K、MS 关，LAMZU 办公 / 60 秒，VAXEE 标准 / 顺滑灵敏。
4. 停止测试实例，再分别恢复各台原值。

证据：`foreground-daily-before.json`、`foreground-game.json`、`foreground-daily-after.json`（均位于 research）。正式配置文件未修改。

## 实物型号识别：通过

两个 VAXEE 接收器的 USB PID 都是 0x2001，不能据此区分鼠标。采用官方控制集合 FF05:0001 和 0x0A 查询，返回鼠标 PID 0x1010（E1）、0x1011（NP-01S V2）。

证据：`research/device-identity-verification.json` 以及本次各实物矩阵。已移除固定本机 ContainerID 别名；休眠或查询失败时显示未确认，不猜型号。

## 自动测试与原生 UI

完整 `go test -count=1 ./...`：18 项通过、1 项显式开启的实物探测测试默认跳过。证据：`research/tests.jsonl`。

覆盖场景匹配、双品牌/场景隔离、同品牌多设备、手动只操作所选设备、自动模式阻止手动覆盖、重连、失败重试、旧程序冲突、电量缓存、事件循环空闲不轮询、快照隔离、延迟捕获/取消、配置校验、损坏文件不覆盖、旧格式迁移、导出往返及 Win64 结构 ABI。

Computer Use 已实际验证：首页与两品牌性能页、标准模式禁用 2K/4K、配置名称编辑保存、重启持久化、GUI 导出和导入、收起到托盘、再次启动恢复同一常驻实例。GUI 导出与原文件比较一致，GUI 导入后界面和文件均显示测试名称，自动切换保持关闭。

文件选择器现为临时原生辅助进程。实际导入/导出完成后，辅助进程退出，主程序恢复为单常驻进程。隔离测试文件位于 `build/qa/data`，正式配置未改动。

## 性能

最终自动切换开启、三台设备在线、窗口已收起，采样 30.003 秒：

- CPU 时间增量 0.015625 秒，即单核约 0.052%。
- 工作集 28.75 MiB，私有提交 53.65 MiB，330 个句柄。
- 证据：`research/idle-auto-enabled.json`。

此前自动切换关闭、完成 GUI 文件导入导出后，托盘 30 秒采样未测到 CPU 时间增长，工作集约 29.03 MiB（`idle-picker-isolated.json`）。这不代表所有情景绝对零 CPU；真实切换时会执行短时设备通信。

没有工作集裁剪。无 WebView/Electron、动画或持续渲染。前台事件与 HID 插拔事件驱动，5 分钟兜底扫描。文件选择器隔离修复了旧实现打开对话框后保留 Shell 缓存、工作集升到约 76 MiB 的问题。

## 测试发现并修复的问题

- LAMZU 原写入函数忽略指定设备路径，可能选到另一只设备。
- VAXEE 稳定轨迹旧报文把 READ/WRITE 标志误作模式值。改为 byte3=WRITE、byte5=0/1。
- VAXEE 模式/轨迹切换后，高回报率可能未保持；改为先轨迹、再模式、最后回报率，并跳过未变化值、等待设置稳定。
- 读回不能把同命令的写入 ACK 当作有效 READ；增加读取标志校验和有限重试。
- 第一轮中 E1 曾未能恢复 4K，已立即单独恢复并读回确认，随后 57 组矩阵和独立最终读回均通过。
- 原生按钮结构偏移、品牌/场景编辑隔离、空程序列表误判修改、Ctrl+A、异常电量解析与文件对话框常驻内存。

早期失败记录保留用于审计：`hardware-verification-awake.json`、未唤醒时的 `hardware-verification.json`。以修正后的 full-matrix 与最终独立读回为准。

## 覆盖边界

未进行人工真实拔插、长时间压力运行、所有 Windows/DPI 环境、所有固件版本验证。GUI 配置增删和延迟捕获逻辑有自动测试，但尚未逐项完成实物 GUI 场景。不能将以上通过结果扩大为所有型号、固件或长期运行绝无问题。

本机默认关闭 CGO，因此没有声称 `-race` 检查通过。DPI、LOD 等尚未接入的官方功能不在本次已实现功能验收范围内，见 features.md。
