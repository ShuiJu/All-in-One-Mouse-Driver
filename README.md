# Mouse Hub

<img src="assets/logo-256.png" width="96" alt="Mouse Hub 图标">

一个面向 LAMZU 与 VAXEE 鼠标的 Windows 原生配置工具。根据前台程序自动切换性能模式和参数，在日常使用与游戏之间复用不同配置，减少手动调整。

本项目由 ShuiJu 的 [lamzu-autoswitch](https://github.com/ShuiJu/lamzu-autoswitch) 与 [vaxee-autoswitch](https://github.com/ShuiJu/vaxee-autoswitch) 整合发展而来：将两个品牌原本独立的自动切换工具合并为一个程序，并加入统一的设备界面、配置管理与程序绑定。

**Windows x64 · Go + Win32 / GDI · 单进程常驻 · GPL-3.0**

## 主要功能

- **按程序自动切换**：为游戏绑定专用配置，切回未绑定程序时使用默认配置。自动切换默认开启。
- **统一管理两个品牌**：查看设备、连接状态与电量，在同一个界面调整已支持的参数。
- **多设备独立配置**：同型号的多只鼠标按设备标识分别保存参数，可重命名；首页每页四台，支持分页选择。
- **配置与程序绑定**：复制、重命名、删除、导入和导出配置；支持拖放绑定、选择 `.exe`，以及倒计时捕获前台程序。
- **写入后读回确认**：参数下发后核对设备返回值，读回失败不会显示为应用成功。
- **便携与托盘运行**：配置保存在程序旁，关闭窗口收进托盘；可导入两个前身项目的 `.conf` 文件。

## 设备与功能范围

| 品牌 | 型号 | 当前验证范围 |
|---|---|---|
| LAMZU | INCA | 已有核心参数及魔改固件轨迹写入、读回的实物验证 |
| LAMZU | PARO | 已做代码与固件协议适配，尚无 PARO 实机验收 |
| VAXEE | E1 | 已有核心参数、轨迹和型号识别的实物验证 |
| VAXEE | NP-01S V2 | 已有核心参数、轨迹和型号识别的实物验证 |

LAMZU 当前仅通过 `37B0:0010` 接收器的 `mi_02` 控制接口通信，不支持 INCA / PARO 的有线接口。接收器名称与所连鼠标型号分开展示；型号来自鼠标身份回复，不根据接收器名称猜测。其他型号尚未列入已验证支持范围。

| 参数 | LAMZU INCA / PARO | VAXEE E1 / NP-01S V2 |
|---|---|---|
| 性能模式 | 办公、高速、20,000 FPS | 标准、竞技 |
| 回报率 | 1000 / 2000 / 4000 / 8000 Hz | 1000 / 2000 / 4000 Hz；标准模式限 1000 Hz |
| Motion Sync | 支持 | 支持 |
| 休眠时间 | 支持 | 尚未接入 |
| 轨迹模式 | 顺滑轨迹 / 稳定轨迹，**需要魔改固件** | 顺滑灵敏 / 稳定易控 |

当前未接入 DPI、LOD、按键响应时间、配对、板载配置管理及固件升级等功能；不提供改键或宏。完整范围见 [功能对照](docs/features.md)。

### LAMZU 轨迹模式与魔改固件

**LAMZU 的「顺滑轨迹 / 稳定轨迹」是通过魔改固件增加的功能，不是原厂固件原生提供的功能。** VAXEE 的轨迹设置使用其已有控制协议，与 LAMZU 的魔改固件功能分别实现。

- Mouse Hub 只有收到有效的 `TR/v1` 能力回复后才开放 LAMZU 轨迹按钮；原厂固件、旧固件或设备离线时，该选项不可用。在线的原厂固件设备仍可使用其他已支持参数。
- Mouse Hub 不会下载、刷写或自动升级固件。INCA 与 PARO 分别使用各自型号的适配固件，不能混刷。
- 旧配置的轨迹为空时表示「未指定」，应用其他参数不会强制改变轨迹。明确选择并保存后，会随配置生效或设备重连重新应用。
- 固件中的轨迹选择只在本次通电期间保留，断电后恢复稳定轨迹；电脑端保存的选择仍保留在配置文件中。
- INCA 已有实物刷写与轨迹写入、读回记录；PARO 尚无实机验证。轨迹效果、时延和真实 USB 回报间隔尚无量化测量结论。

协议、能力判断和验证边界见 [协议说明](docs/protocol-notes.md) 与 [验证记录](docs/validation.md)。

<details>
<summary>魔改固件产物名称与时序配置（开发参考）</summary>

以下为适配使用的产物名称，不是下载链接：

| 型号 | 固件 | 更新器 |
|---|---|---|
| INCA | `INCA_Mouse_840_APP_v0.0.0.19_TrajectorySwitch_E1Table_NoDownshift_FactoryFPS.hex` | `LAMZU_INCA_840_v0.0.0.19_TrajectorySwitch_E1Table_NoDownshift_FactoryFPS.exe` |
| PARO | `LAMZU_PARO_Mouse_840_APP_v0.0.0.19_TrajectorySwitch_E1Table_NoDownshift_FactoryFPS.hex` | `LAMZU_PARO_840_v0.0.0.19_TrajectorySwitch_E1Table_NoDownshift_FactoryFPS.exe` |

PARO 从自己的原厂固件独立适配。两种型号的顺滑与稳定模式均按 E1 普通周期表适配无线请求定时。下表仅表示 profile code 与实际 divider 匹配时的 TIMER1 平均请求周期配置，**不是实测 USB 回报间隔**；驱动界面仍提供 1K / 2K / 4K / 8K 档位。

| 无线档位 | profile code / divider | 顺滑轨迹 | 稳定轨迹 |
|---|---|---:|---:|
| 125 Hz | `08` / 64 | 7990 µs | 7990 µs |
| 250 Hz | `04` / 32 | 3990 µs | 3990 µs |
| 500 Hz | `02` / 16 | 1990 µs | 1990 µs |
| 1000 Hz | `01` 或 `10` / 8 | 996 µs | 996 µs |
| 2000 Hz | `20` / 4 | 498 µs | 498 µs |
| 4000 Hz | `40` / 2 | 248 µs | 248.875 µs |
| 8000 Hz | `80` / 1 | 124 µs | 124.4375 µs |

8K 由普通 4K 比例扩展，不采用 E1 的特殊 123 µs 包路径。未知 code、code 与实际 divider 不匹配，或有线状态下保持原厂 125 µs TIMER1 节拍并清零分数相位；切换模式先恢复 125 µs，下次 IRQ 再应用目标。传感器 FPS、TIMER0/2 和 16 位累积/ring/ACK 路径保留，所有 direct/async downshift CPI 上限统一为 30,000。

</details>

## 快速开始

### 下载运行

1. 打开 [最新发布](https://github.com/ShuiJu/All-in-One-Mouse-Driver/releases/latest)，下载 Windows x64 可执行文件，首版附件为 `MouseHub.V0.1.0.exe`。
2. 将程序放在有写入权限的独立目录中，双击运行；无需安装 Go，也无需安装常驻服务。
3. 程序会在同目录创建 `data/`。升级时退出旧实例、替换可执行文件并保留 `data/`，即可继续使用已有配置。

发布沿用 VAXEE AutoSwitch 的方式：大写 `V` 版本标签、中文更新说明，直接提供带版本号的 `.exe` 附件。

### 从源码构建

构建环境为 **Windows x64、Go 1.24 或更新版本**，没有外部 Go 模块依赖。运行编译后的程序不需要安装 Go。

在 PowerShell 中执行：

```powershell
git clone https://github.com/ShuiJu/All-in-One-Mouse-Driver.git
Set-Location All-in-One-Mouse-Driver
powershell -NoProfile -ExecutionPolicy Bypass -File .\build.ps1
.\build\MouseHub.exe
```

`build.ps1` 会先执行 `go test ./...`，再生成 `build/MouseHub.exe`。`build/` 是本地构建输出，不随 Git 提交。

### 首次配置

1. **退出旧工具**：先退出 LAMZU AutoSwitch 与 VAXEE AutoSwitch。若检测到旧工具仍在运行，Mouse Hub 会暂停控制相应品牌，避免同时下发设置。
2. **选择设备**：让鼠标处于唤醒状态，从首页选中设备，点击设备卡片进入性能或参数页。超过四台设备时，用设备栏两侧箭头翻页。
3. **编辑配置**：在「我的配置」中选择或复制一份配置，进入设备参数页调整并保存。同一份配置可包含不同品牌、不同设备的参数。
4. **绑定程序**：将配置拖到程序上，或先选配置再点程序；也可添加 `.exe`，或点击「5 秒后捕获并绑定」后切到目标程序。
5. **设置默认配置**：将日常使用的配置「设为默认」。例如把「游戏配置」绑定到游戏，离开游戏、切到未绑定程序后便回到默认配置。

自动切换开关位于「我的配置」。暂停自动切换后，保存会将当前编辑配置应用到首页选中的设备，可用于手动调整。窗口标题显示已保存并生效的配置名称，未保存的编辑不改变标题。

### 配置迁移与备份

- 在设置页使用「导入 LAMZU」或「导入 VAXEE」，读取前身项目的 `.conf` 文件；源文件不会被修改。
- 在「我的配置」中使用「导入配置 / 导出配置」，迁移或备份 Mouse Hub 配置。
- 默认数据目录为**可执行文件旁的 `data/`**：`config.json` 保存配置，`mousehub.log` 保存日志。从源码构建后，对应 `build/data/`。
- 同型号设备按设备标识保存参数；更换 USB 端口或接收器后能否沿用原标识，取决于系统提供的设备身份，不能保证始终不变。

### 托盘与启动参数

点击窗口关闭按钮（X）会收进托盘。双击托盘图标或再次启动程序可恢复窗口，托盘右键菜单可退出。

| 参数 | 用途 |
|---|---|
| `--hidden` | 启动后直接收进托盘 |
| `--data <directory>` | 指定独立数据目录 |
| `--probe` | 输出设备身份与旧工具冲突信息，不应用配置 |
| `--qa` | 不写入设备的隔离 UI 验证模式 |

```powershell
.\build\MouseHub.exe --hidden
.\build\MouseHub.exe --data .\my-data
```

## 后台开销与休眠

程序使用原生 Win32 / GDI 界面，单进程常驻，没有 WebView、Electron、浏览器运行时或网络服务；运行时不依赖官方站点。

- 前台程序变化由 Windows 事件通知，HID 插拔通知触发重新发现；没有周期性 HID 维护扫描或应用失败定时重试。
- 游戏期间不以定时器轮询鼠标。切换到已绑定程序时，仍会按需应用目标配置；目标参数未变化时复用成功应用的缓存。
- 每次将 Mouse Hub 切到前台，会在后台刷新型号、连接状态、电量和当前设置；保留参数下发缓存，避免仅因刷新重复写入未变化配置。
- 电量平时使用缓存，不按固定频率查询。设备休眠或无有效身份回复时，面板显示「离线」，不会把历史型号或已应用状态当作当前结果展示。
- 不发送专门的唤醒命令。存在恢复候选时，才按需监听原始鼠标输入；对应鼠标活动后合并一次恢复尝试，其他设备不会因此被轮询。
- 窗口隐藏后释放子控件与字体；文件选择器使用临时辅助进程，选完退出。没有动画、持续绘制循环或工作集强制裁剪。

## 开发与验证

```powershell
# 运行自动测试
go test -count=1 ./...

# 查询设备身份，不应用配置
go run . --probe

# 读取设备当前设置
go run ./cmd/readback
```

需要写入硬件进行验收时，先退出 Mouse Hub 和旧 AutoSwitch，确保相关鼠标醒着，再按 [验证记录](docs/validation.md) 中的流程保存基线、测试并恢复。自动测试通过不代表所有固件、Windows 版本或长期休眠场景均已完成实机验证。

历史 INCA、E1、NP-01S V2 的核心实机矩阵覆盖 57 组参数、前台联动与原值恢复；该结果不扩大为 PARO 或后续所有固件的验收结论。`tools/measure-idle.ps1` 可采样实际进程 CPU 时间和内存；已有性能数据及其适用版本见验证记录。

| 文档 | 内容 |
|---|---|
| [功能对照](docs/features.md) | 已接入功能、官方功能差异与型号范围 |
| [协议说明](docs/protocol-notes.md) | 接收器控制接口、身份识别与魔改固件轨迹协议 |
| [验证记录](docs/validation.md) | 自动测试、实物记录、测量与未覆盖边界 |
| [项目状态](docs/project-state.md) | 开发交接与历史构建状态 |

## 项目前身与许可

两个前身项目最初分别为对应品牌提供基于前台程序的性能模式自动切换，以减少手动调整、兼顾续航：

- [ShuiJu/lamzu-autoswitch](https://github.com/ShuiJu/lamzu-autoswitch) — LAMZU AutoSwitch。
- [ShuiJu/vaxee-autoswitch](https://github.com/ShuiJu/vaxee-autoswitch) — VAXEE AutoSwitch。

整合起点分别为 LAMZU [`d673ccf`](https://github.com/ShuiJu/lamzu-autoswitch/commit/d673ccf41ddac1d632527be9f0cdd498cdadb406) 与 VAXEE [`4ceb1df`](https://github.com/ShuiJu/vaxee-autoswitch/commit/4ceb1df22a3809dde4a286298261a706b9cd7c38)。Mouse Hub 在此基础上统一设备与配置管理，继续保留原项目的 [GPL-3.0 许可证](LICENSE)。

本项目为社区工具。官方网页代码仅用于核对公开协议与功能；LAMZU 的魔改固件扩展与官方原厂功能在上文分别说明。
