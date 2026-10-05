# Mouse Hub

将 ShuiJu 的 LAMZU / VAXEE AutoSwitch 合并为一个 Windows 原生程序。

## 当前版本

Go + Win32 / GDI，单个常驻进程，无 WebView、Electron、浏览器运行时或网络服务。界面参考 `UI设计` 中的 A HUB 截图：深色设备首页、「我的配置」程序绑定，以及按当前选中设备显示的参数页。顶栏显示当前生效配置（如「游戏配置 · LAMZU INCA」）。首页已嵌入 INCA、E1、NP-01S V2 的透明底实拍图，并使用自定义窗口/托盘图标；图片加载失败时回退到手绘示意图。

**本机三台设备的核心实机验证已通过：57 组参数、真实前台联动及原值恢复。** 验证范围与未覆盖项目见 [验证记录](docs/validation.md)。

## 使用

运行 `build/MouseHub.exe`。配置保存在程序旁的 `data/config.json`。

1. 退出原来的两个 AutoSwitch；检测到它们时，新程序暂停控制相应品牌。
2. 首页点选设备；整张设备卡片可点击进入该设备的参数设置。设备超过四台时，用设备栏两侧箭头分页，所有已发现设备都可选。
3. 在「我的配置」中管理配置卡片与程序。把配置拖到程序上，或先选配置再点程序，即可绑定。自动切换默认开启。
4. 设置页可导入原项目 `.conf`。导入不修改源文件。
5. 点标题栏关闭（X）会收进托盘；双击托盘图标或再次启动程序可打开窗口。托盘右键菜单可退出。

每份配置是一套可复用参数。未绑定的前台程序使用默认配置；绑定后使用对应配置。同型号的多只鼠标按设备标识分别保存参数，可在「设备信息」中改名。

## 已整合范围

- LAMZU INCA：办公/高速/20,000 FPS、1000/2000/4000/8000 Hz、Motion Sync、休眠时间。
- VAXEE：标准/竞技、Motion Sync、1000/2000/4000 Hz、顺滑灵敏/稳定易控。标准模式限制为 1000 Hz。
- 两品牌设备枚举、电量查询、配置导入导出、新建/复制/重命名/删除、配置↔程序拖放绑定、前台程序自动切换。
- VAXEE 用官方 `0x0A` 命令查询鼠标 PID。E1 为 `0x1010`，NP-01S V2 为 `0x1011`，不依赖固定 Windows ContainerID。
- 下发后读回确认；读取失败不会显示为成功。
- 自动切换关闭时，保存会把当前编辑配置应用到首页选中的设备。

按要求排除改键。DPI、LOD 等其余官方功能的差异见 [功能对照](docs/features.md)。

## 低开销设计

- `SetWinEventHook(EVENT_SYSTEM_FOREGROUND)` 接收前台变化事件；没有每秒 HID 枚举。
- HID 插拔通知或用户点击“刷新设备”触发重新枚举；恢复候选对应鼠标的活动只触发该设备的一次恢复尝试，不设置周期性 HID 维护扫描。
- 缓存设备控制通道与成功应用的参数；目标值未变化时不写入。
- 电量查询缓存；只在启动、设备刷新、恢复尝试或显式引擎扫描时查询，不以定时器轮询。
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
