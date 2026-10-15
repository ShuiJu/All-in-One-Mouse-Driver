# Mouse Hub

将 ShuiJu 的 LAMZU / VAXEE AutoSwitch 合并为一个 Windows 原生程序。

## 当前版本

Go + Win32 / GDI，单个常驻进程，无 WebView、Electron、浏览器运行时或网络服务。界面参考 `UI设计` 中的 A HUB 截图：深色设备首页、「我的配置」程序绑定，以及按当前选中设备显示的参数页。顶栏显示当前生效配置（如「游戏配置 · LAMZU INCA」）。首页已嵌入 INCA、E1、NP-01S V2 的透明底实拍图，并使用自定义窗口/托盘图标；图片加载失败时回退到手绘示意图。

**既有版本的三台设备核心实机验证已通过：57 组参数、前台联动及原值恢复。** INCA E1Table 新轨迹固件已实机刷写，并通过有线轨迹切换与读回；实际轨迹改善和时延尚未测量。PARO 依据原厂固件适配，没有 PARO 实机，不能视为硬件验收。详见 [验证记录](docs/validation.md)。

## 使用

运行 `build/MouseHub.exe`。配置保存在程序旁的 `data/config.json`。

1. 退出原来的两个 AutoSwitch；检测到它们时，新程序暂停控制相应品牌。
2. 首页点选设备；整张设备卡片可点击进入该设备的参数设置。设备超过四台时，用设备栏两侧箭头分页，所有已发现设备都可选。
3. 在「我的配置」中管理配置卡片与程序。把配置拖到程序上，或先选配置再点程序，即可绑定。自动切换默认开启。
4. 设置页可导入原项目 `.conf`。导入不修改源文件。
5. 点标题栏关闭（X）会收进托盘；双击托盘图标或再次启动程序可打开窗口。托盘右键菜单可退出。

每份配置是一套可复用参数。未绑定的前台程序使用默认配置；绑定后使用对应配置。同型号的多只鼠标按设备标识分别保存参数，可在「设备信息」中改名。

## 已整合范围

- LAMZU INCA / PARO：办公/高速/20,000 FPS、1000/2000/4000/8000 Hz、Motion Sync、休眠时间；运行支持 TR/v1 协议的新固件时另可选顺滑轨迹 / 稳定轨迹。PARO 尚无实机验证。
- VAXEE：标准/竞技、Motion Sync、1000/2000/4000 Hz、顺滑灵敏/稳定易控。标准模式限制为 1000 Hz。
- 两品牌设备枚举、电量查询、配置导入导出、新建/复制/重命名/删除、配置↔程序拖放绑定、前台程序自动切换。
- 通过接收器查询所连鼠标型号：VAXEE读取官方0x0A鼠标PID；LAMZU读取通用CMD81固件版本回复内的鼠标PID，0009对应INCA、0007对应PARO。未知有效PID保留原码，不按USB接收器名称猜型号。
- 下发后读回确认；读取失败不会显示为成功。
- 自动切换关闭时，保存会把当前编辑配置应用到首页选中的设备。
- 窗口标题随生效配置显示，例如「Mouse Hub: 日常配置」或「Mouse Hub: 游戏配置」；后台自动切换同样更新标题，未保存的编辑不改变标题。

LAMZU 的「参数」页将 Motion Sync、追踪轨迹与休眠分开设置。INCA 与 PARO 使用同一轨迹协议，能力由有效 TR/v1 回复决定，不按设备名称判断；旧固件或设备未连接时，轨迹按钮禁用。旧配置的轨迹为空时显示「未指定」，应用其它参数不会强制改变轨迹；选择顺滑轨迹或稳定轨迹后，通过原有保存路径保存在配置中，并在配置生效或设备重连时重新应用。页面区分配置选择与已应用状态，只有读回确认成功的模式才显示为已应用。

INCA / PARO 轨迹模式只在设备本次通电期间保留，不写入设备永久存储；断电后设备恢复稳定轨迹。驱动中的配置会持久保存，因此设备重连且相应配置生效后，可重新应用已保存的选择。两种模式保留原厂传感器 FPS、TIMER0/2 和 16 位累积/ring/ACK 路径；所有 direct/async downshift CPI 上限统一为 30,000。驱动不会自动刷写固件。

面板只通过 LAMZU 接收器 `37B0:0010` 的 `mi_02` 控制鼠标，不枚举或控制 INCA/PARO 有线接口。接收器名称固定为「LAMZU 8K Receiver」，另行显示查询得到的鼠标型号和识别状态；设备名称与鼠标身份不混用。两品牌未收到有效响应时，型号和状态仅显示「离线」，不加括号、历史识别或恢复说明；保留的会话缓存只用于内部恢复，不显示为当前型号或已应用状态。有效未知PID覆盖旧型号；LAMZU匹配请求的A4无连接回复会清除旧身份和已应用状态。重新连接时重新识别，不跨接收器移用缓存。不增加后台型号轮询，不为识别主动唤醒鼠标。轨迹能力仍独立以有效TR/v1回复判断，GUI与指定path写入仅使用所选接收器，既有设备ID与profile绑定保留。

最新编译面板为 [build/MouseHub.exe](build/MouseHub.exe)，始终使用项目默认程序名。INCA 配套固件为 `INCA_Mouse_840_APP_v0.0.0.19_TrajectorySwitch_E1Table_NoDownshift_FactoryFPS.hex`，更新器为 `LAMZU_INCA_840_v0.0.0.19_TrajectorySwitch_E1Table_NoDownshift_FactoryFPS.exe`；PARO 配套固件为 `LAMZU_PARO_Mouse_840_APP_v0.0.0.19_TrajectorySwitch_E1Table_NoDownshift_FactoryFPS.hex`，更新器为 `LAMZU_PARO_840_v0.0.0.19_TrajectorySwitch_E1Table_NoDownshift_FactoryFPS.exe`。PARO 从自己的原厂固件独立适配，不能混刷 INCA 固件。两种型号的顺滑与稳定均按 E1 普通周期表适配无线请求定时。下表为 profile code 与实际 divider 相符时的 TIMER1 平均请求周期配置，不是实测 USB 间隔；驱动界面仍保留原有 1K/2K/4K/8K 档位。

| 无线档位 | profile code / divider | 顺滑轨迹 | 稳定轨迹 |
|---|---|---:|---:|
| 125 Hz | `08` / 64 | 7990 µs | 7990 µs |
| 250 Hz | `04` / 32 | 3990 µs | 3990 µs |
| 500 Hz | `02` / 16 | 1990 µs | 1990 µs |
| 1000 Hz | `01` 或 `10` / 8 | 996 µs | 996 µs |
| 2000 Hz | `20` / 4 | 498 µs | 498 µs |
| 4000 Hz | `40` / 2 | 248 µs | 248.875 µs |
| 8000 Hz | `80` / 1 | 124 µs | 124.4375 µs |

8K 按用户选择由普通 4K 比例扩展，不使用 E1 的特殊 123 µs 包路径。未识别的 code、code 与实际 divider 不匹配或有线状态保持原厂 125 µs TIMER1 节拍并清零分数相位；切换模式先立即恢复 125 µs 并清相位，下次 IRQ 再应用目标。旧版将 INCA 原厂 125 µs 基准误乘 `248875/248000` 得到约 250.882/125.441 µs 的配置已废弃。协议与识别证据限制见 [协议说明](docs/protocol-notes.md)。


按要求排除改键。DPI、LOD 等其余官方功能的差异见 [功能对照](docs/features.md)。

## 低开销设计

- `SetWinEventHook(EVENT_SYSTEM_FOREGROUND)` 接收前台变化事件；没有每秒 HID 枚举。
- MouseHub每次切到前台会在后台刷新设备型号、连接状态、电量和当前设置；首页/设备信息页不再提供手动刷新按钮。HID插拔通知仍触发重新枚举，恢复候选对应鼠标的活动只触发该设备的一次恢复尝试，不设置周期性HID维护扫描。
- 缓存设备控制通道与成功应用的参数；目标值未变化时不写入。
- 电量平时使用缓存；切到MouseHub前台时立即重新查询，不等待缓存到期，也不以定时器轮询。状态刷新保留下发缓存，不因刷新重复写入未变化配置，读回值不会被旧配置覆盖。
- 原生文件选择器放在临时辅助进程中，选完即退出，避免主程序长期保留 Shell 扩展及缩略图缓存。
- 隐藏后销毁子控件和字体，显示时重建；无动画、持续绘制循环或工作集强制裁剪。
- 设备发现或应用失败时保留已知设备状态；仅在恢复候选存在时订阅原始鼠标输入，匹配到对应物理鼠标的活动后合并一次该设备恢复，休眠期间不主动轮询，也不会用输入活动唤醒其他设备。

## 构建与验证

Windows x64，Go 1.24 或更新版本，无外部 Go 依赖。

```powershell
.\build.ps1
go test -count=1 ./...
go run . --probe
go run ./cmd/readback
go run ./cmd/verify -out research/hardware-baseline.json
# 仅在三只鼠标醒着、旧版及新 GUI 均退出后执行；自动保存基线、测试、恢复：
go run ./cmd/verify -write -out research/hardware-verification.json
```

`--hidden` 从托盘启动，`--data <directory>` 指定独立数据目录，`--qa` 为不写入设备的隔离 UI 验证模式。`tools/measure-idle.ps1` 采样真实进程 CPU 时间与内存，不进行工作集裁剪。

## 来源与许可

GPL-3.0，保留原项目的 [LICENSE](LICENSE)。整合所用本地源代码基于：

- https://github.com/ShuiJu/lamzu-autoswitch — `d673ccf41ddac1d632527be9f0cdd498cdadb406`
- https://github.com/ShuiJu/vaxee-autoswitch — `4ceb1df22a3809dde4a286298261a706b9cd7c38`

官方网页代码只用于核对公开协议和功能；新程序运行时不连接官方站点。
