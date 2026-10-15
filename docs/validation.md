# 验证报告

2026-09-14 状态复核：当前 HEAD `e80d474` 全量自动测试为 23 项通过、1 项实物探测跳过。下文的 57 组实物及性能数据是 2026-09-13 历史结果，不是新增实拍图/Logo 后构建的重新测量。最新实现和证据边界见 [项目现状](project-state.md)。

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

## 配置模型 v2 与原生 UI

完整 `go test -count=1 ./...`：绑定、同型号设备隔离、v1 迁移等均通过；1 项显式开启的实物探测测试默认跳过。

覆盖场景匹配、双品牌/场景隔离、同品牌多设备、手动只操作所选设备、自动模式阻止手动覆盖、重连、失败重试、旧程序冲突、电量缓存、事件循环空闲不轮询、快照隔离、延迟捕获/取消、配置校验、损坏文件不覆盖、旧格式迁移、导出往返及 Win64 结构 ABI。

配置模型已改为 v2「配置卡片 ↔ 程序绑定」：

- 默认开启自动切换；自动切换管理只在「我的配置」。
- 配置拖到程序上、或点程序完成绑定；删除配置会解绑程序。
- 设备设置页只显示当前选中设备的品牌参数；同型号多设备按设备键分别保存。
- 整张首页设备卡片可点击进入设置；圆角按钮用区域裁剪，消除子窗口黑角。
- 自动切换关闭时，保存会把当前编辑配置应用到选中设备。
- 顶栏显示当前生效配置（如「游戏配置 · LAMZU INCA」），不再固定为 MOUSE HUB。
- 右上角仅保留「设置」；点标题栏 X 收进托盘。
- 自动切换仅在设备拓扑变化时重建首页；其余状态刷新不拆控件。
- 「刷新设备」的「正在检测…」在引擎扫描完成通知后自动清除。
- 首页按型号显示透明底鼠标实拍图（INCA 源图为 `INCA-2.png` 纯白泛洪去背）；托盘与窗口使用嵌入 Logo（`assets/app.ico`）。
- 翻页时新按钮不再闪白底（见下文「按钮白底闪烁」）。

Computer Use 已实际验证：首页与两品牌性能页、标准模式禁用 2K/4K、配置名称编辑保存、重启持久化、GUI 导出和导入、收起到托盘、再次启动恢复同一常驻实例。GUI 导出与原文件比较一致，GUI 导入后界面和文件均显示测试名称。翻页白底闪烁修复后已由用户实机确认消失。

文件选择器现为临时原生辅助进程。实际导入/导出完成后，辅助进程退出，主程序恢复为单常驻进程。隔离测试文件位于 `build/qa/data`，正式配置未改动。

## 性能

最终自动切换开启、三台设备在线、窗口已收起，采样 30.003 秒：

- CPU 时间增量 0.015625 秒，即单核约 0.052%。
- 工作集 28.75 MiB，私有提交 53.65 MiB，330 个句柄。
- 证据：`research/idle-auto-enabled.json`。

此前自动切换关闭、完成 GUI 文件导入导出后，托盘 30 秒采样未测到 CPU 时间增长，工作集约 29.03 MiB（`idle-picker-isolated.json`）。这不代表所有情景绝对零 CPU；真实切换时会执行短时设备通信。

- 没有工作集裁剪。无 WebView/Electron、动画或持续渲染。前台事件与 HID 插拔事件驱动；当前不设置周期性 HID 维护扫描。只有恢复候选存在时，原始鼠标输入才会在移动后合并一次恢复通知；文件选择器隔离修复了旧实现打开对话框后保留 Shell 缓存、工作集升到约 76 MiB 的问题。

## 2026-10-05 本轮低开销恢复验证

- 引擎在发现或应用失败时保留已知设备、参数和电量；恢复候选只在对应物理鼠标产生用户活动后触发一次恢复尝试。Raw Input 处理只解析设备路径并排队通知，实际设备查询和应用仍在引擎步骤中执行。
- 恢复活动按物理设备隔离；测试确认活动设备只增加该设备的一次写入，其他设备不重发现、不读电量、不写配置。连续输入在一次引擎步骤内合并，恢复候选仍存在时才重新等待下一次活动。
- 如果启动时没有已知设备但发现返回错误，原始输入活动只按 2 秒冷却触发一次重新发现；这条路径仍不使用周期定时器，发现成功后才恢复正常事件驱动流程。
- 前台事件不会触发电量查询；代码移除了周期性 HID 维护扫描和应用失败定时重试。休眠期间不发送唤醒命令，也不以定时器轮询鼠标。
- 本轮已执行：`go test -count=1 ./...`、`build.ps1`、当前构建的 `build/MouseHub.exe --probe`，以及 `--qa --hidden` 启动/退出烟测。探测到 1 台 LAMZU 与 2 台 VAXEE；当时 VAXEE 仍处于休眠导致型号查询返回未知，不能将该结果解释为永久协议不兼容。
- 本轮未完成真实移动唤醒后的端到端恢复、人工拔插、长期运行和分页 UI 视觉验收；这些仍是后续实机验证边界。

## 测试发现并修复的问题

- LAMZU 原写入函数忽略指定设备路径，可能选到另一只设备。
- VAXEE 稳定轨迹旧报文把 READ/WRITE 标志误作模式值。改为 byte3=WRITE、byte5=0/1。
- VAXEE 模式/轨迹切换后，高回报率可能未保持；改为先轨迹、再模式、最后回报率，并跳过未变化值、等待设置稳定。
- 读回不能把同命令的写入 ACK 当作有效 READ；增加读取标志校验和有限重试。
- 第一轮中 E1 曾未能恢复 4K，已立即单独恢复并读回确认，随后 57 组矩阵和独立最终读回均通过。
- INCA 性能档位切换曾先发送 `0x0B`、再发送 `0x13`，固件会用旧的 `0x13` 状态选择传感器配置；现改为基线 `0x0B=0 → 0x13=0`，再按目标发送最终状态，每步间隔 100 ms。
- 2026-10-05 实物验证中，LAMZU 的办公、高速和高性能组合均写入并读回通过，原值恢复为 true；证据为 `research/driver-sequence-verification.json`。同次命令的两个 VAXEE 设备因既有 `command 07` 读回失败未进入测试，不影响 LAMZU 结果。
- 原生按钮结构偏移、品牌/场景编辑隔离、空程序列表误判修改、Ctrl+A、异常电量解析与文件对话框常驻内存。
- 圆角按钮子窗口盖黑角：改为 `SetWindowRgn` 裁剪，四角露出面板。
- 同型号设备曾按型号合并参数：改为按稳定设备键分别保存。
- 配置模型从 v1「日常/游戏」迁移为 v2 配置卡片；迁移写 `config.json.v1.bak` 备份。
- 单实例 `CreateMutexW` 不会清空 last-error，陈旧 `ERROR_ALREADY_EXISTS` 会让首个实例静默退出；现先 `SetLastError(0)`，且互斥存在但找不到窗口时继续作为存活实例。
- 按钮白底闪烁：BUTTON 子窗口在 owner-draw 前会用 `COLOR_BTNFACE` 擦背景。修复为隐藏创建后显示、全部按钮子类化并吞掉 `WM_ERASEBKGND`、窗口类深色背景刷、处理 `WM_CTLCOLORBTN`、重绘不对子窗口 `RDW_ERASE`。`interactive()` 不得二次子类化，否则 `WM_SHOWWINDOW` 栈溢出。

早期失败记录保留用于审计：`hardware-verification-awake.json`、未唤醒时的 `hardware-verification.json`。以修正后的 full-matrix 与最终独立读回为准。

## 覆盖边界

未进行人工真实拔插、长时间压力运行、所有 Windows/DPI 环境、所有固件版本验证。GUI 配置增删和延迟捕获逻辑有自动测试，但尚未逐项完成实物 GUI 场景。不能将以上通过结果扩大为所有型号、固件或长期运行绝无问题。

本机默认关闭 CGO，因此没有声称 `-race` 检查通过。DPI、LOD 等尚未接入的官方功能不在本次已实现功能验收范围内，见 features.md。

## INCA可切换轨迹：基准修正前的软件验证历史

- 基准修正前执行 `go test ./...`，所有包通过；当时 Windows 构建为 `build/MouseHub_Trajectory.exe`，SHA-256=`51974a807baea7b2c05c4bab8ae89f2f6141662dc999d7f7e93369f35b388d91`。此 hash 是历史产物证据，不代表 E1 普通周期表及「顺滑轨迹」标签修正后的构建。原有EXE保留，临时UI fixture源和EXE已删除。
- 新协议用1E写smooth/stable、9E查，10字节payload `A1 00 02 04 01 9E 01 MODE 54 52` 的TR标记定义v1。能力解析包含短/错误/旧回复、ReportID和padding边界；profile保存、仅改变轨迹的自动切换、去重和重连行为由当时的软件测试覆盖。旧LAMZU空轨迹表示未指定，旧固件其他设置仍可用。
- 当时启动真实EXE的 `--qa --data build/ui-trajectory-real`，观察不支持能力时轨迹按钮禁用，Motion Sync和全部休眠控件可用；截图 `build/ui-trajectory-real/inca-trajectory-disabled.png`。该观察不保证当时mouse已成功回包，只证明UI对未确认能力的处理。
- 当时的支持路径使用临时 `trajectoryqa` 编译标签的设备能力fixture，`readOnlyHardware.Apply`仍拒绝设备写入。实际Win32按钮点击stable、sleep600并保存，再点击smooth保存，核对 `build/ui-trajectory-supported/config.json` 中设备scope；截图 `inca-trajectory-stable.png`、`inca-trajectory-smooth.png` 位于同目录。界面保留「已应用：尚未确认」，没有用fixture伪称硬件设置成功；这些截图也不是此次标签修正后的界面验收。
- 基准修正前的 `verify_inca_trajectory.py` 执行启动、30个命令/边界场景、4K和8K各1984次补丁IRQ、12个ACK/timeout/失败恢复，以及六直接/八异步DPI场景。当时两档分数节拍均875次126及1109次125，总248875 tick，关闭后恢复125。该旧算法误以 INCA 原厂125 µs 为缩放基准，现已废弃；这些受控CPU/MMIO结果只保留作历史证据，不能用于证明当前 E1 普通周期表实现，也不是实机测时。
- 只读观察到有线鼠标37B0:0009和接收器37B0:0010都正常PnP枚举。当前驱动只使用接收器mi_02；80/87/89/8B/93查询仍返回A0请求回显、未收到有效配置，因此未宣称此次连接已经完成新协议往返。
- 尚未实机刷写新固件、运行更新器、写实物profile、验证新轨迹host往返或测量真实周期/轨迹/延迟。既有57组实机证据不包含新增功能。

## INCA E1普通周期表修正与当前验证边界

当前配套固件为 `INCA_Mouse_840_APP_v0.0.0.19_TrajectorySwitch_E1Table_NoDownshift_FactoryFPS.hex`，更新器为 `LAMZU_INCA_840_v0.0.0.19_TrajectorySwitch_E1Table_NoDownshift_FactoryFPS.exe`。修正不再把原厂 TIMER1 的125 µs乘以 `248875/248000`；旧约250.882/125.441 µs结果已作废。smooth标签改为「顺滑轨迹」，stable仍为「稳定轨迹」；VAXEE标签不变。Motion Sync、休眠与配置的保存流程不变。

无线且profile code与实际divider相符时，E1普通周期表两模式的目标为：1K 996 µs、2K 498 µs、500 Hz 1990 µs、250 Hz 3990 µs、125 Hz 7990 µs；4K smooth/stable为248/248.875 µs；8K为124/124.4375 µs。8K由4K比例扩展，不使用E1特殊123 µs包路径；smooth/stable映射不能反转。完整code/divider表见 [README](../README.md)。这些是请求周期配置，不是实测USB间隔。

TIMER1以1 MHz量化，采用 `base=124 + fraction/64`：1K/2K分子32、500 Hz为24、250 Hz为44、125 Hz为54、4K/8K smooth为0而stable为28。64 tick周期精确累计；未知code、实际divider不匹配或有线状态保持125并清分数相位。模式写入立即恢复125并清相位，下次IRQ应用目标。1E/9E/TR、开机RAM默认stable及profile保存重连应用策略保持；传感器FPS、TIMER0/2不改。

本次修正的软件执行结果与最终构建hash须以统一验证后的记录为准，上节旧软件和UI fixture结果不迁移为新版本通过结论。新固件尚未实机刷写，新增轨迹host往返、真实周期/轨迹/延迟与实物profile写入仍未验证；历史57组及原参数验证不包含新增轨迹功能。

### 本次统一软件验证结果

- 最终 E1Table 固件受控 CPU/MMIO 验证通过：37 个命令/边界场景，16 行周期矩阵（8 个 profile code × smooth/stable，包含两个 1K code），每行执行 2048 次真实补丁 IRQ；12 个 ACK/timeout/失败恢复、6 个直接 DPI 与 8 个异步 DPI 场景通过。这里的 IRQ 是受控 CPU 执行，不是实机测时或 USB 回报间隔测量。
- 配套 SFX 更新器完整性检查通过。
- 驱动 `go test ./...` 通过；最终 `build/MouseHub_Trajectory.exe` 的 SHA-256 为 `500cd92864a6433073e266578eecabdcfbd0ebe402204a5a18a040d651f6d749`。
- 此次软件通过结果不包含实机刷写或新增轨迹 host 往返。接收器查询仍观察到 A0 请求回显，不能据此宣称新协议实机读回成功；此前 UI 能力 fixture 仍只作历史界面与保存证据。后续实机结果须另行记录。

## INCA E1Table实机刷写与有线驱动验证

- 用户授权后运行E1Table原厂格式更新器；实际观察bootloader000A，随后恢复0009，窗口PASS/100%/Device Connected。记录位于固件目录 `analysis/static/inca_e1table_hardware.json` 和 `inca_e1table_flash_pass.png`。
- 有线协议切换smooth/stable，TR回复MODE分别0/1；驱动实际 `WindowsHardware.Discover/ReadSettings/Apply` 识别有线能力、应用两模式并独立读回与目标完全一致，最后恢复stable及原1000Hz/MS0/办公/sleep30。
- 新增0009mi_02枚举/指定路径校验，保留0010接收器，面板分别标注控制接口。实际Win32 QA发现有线TR，两个轨迹按钮启用并点击，MS/休眠仍可操作；QA无硬件写入，真实写入由上一条独立消费者烟测完成。
- 本次有线接入后 `go test ./...` 通过并构建；驱动SHA-256=`5605cbe1571114df1097717e4a761e771eaf266104980098e9677d33fb82eda8`。最终配送哈希以固件release JSON为准。
- 接收器在鼠标插线时仍只回显A0；无线往返、实际IRQ/SPI/USB周期和光学轨迹未测。硬件成功不能消除原厂16位累计、重置/队列满边界。


## PARO共用驱动与INCA保留验证

- 控制白名单支持PARO有线0007、INCA有线0009和通用receiver0010，拒绝bootloader0008/000A、未知VID/PID、mi00、kbd、伪mi020及实例段伪mi02。只匹配HID身份段的实际mi_02字段，匹配大小写不复制完整path。
- `go test ./...`通过，当前共用驱动SHA-256=`33fb42d19b50d5099bf6b129b47a3b927843e42aa4b370bb09cf5d868c9054d8`。实际 `cmd/readback` 选择INCA0009，回报率1000、MS0、office、sleep30与原值一致；没有给E1或用户配置做修改。
- 实际Win32共用页面在INCA有线TR有效时启用顺滑/稳定按钮、两者点击成功并保留MS/休眠。无遮挡窗口截图 `build/ui-lamzu-trajectory-final/parameters.png` 来自PrintWindow；只读QA不写设备，「已应用：尚未确认」与该限制一致，真实写/独立读回由前一节消费者烟测单独证明。
- PARO没有实物，本次驱动为原厂PID/接口证据及共用协议的软件适配；不声称PARO主机通信或轨迹已实测。接收器0010只显示型号未确认，不把它自动认成PARO。


## PARO固件最终软件验收

- PARO与INCA共用机器码执行验证各通过52命令/边界、16行×2048IRQ、12稳定恢复及两模式28项DPI配置（每mode六直接/八异步）；两model原厂transport15场景通过，PAROmotion14场景通过。新PARO SFX实际内外解包及成员比较通过；无PARO设备，不声称刷写、真实Feature/RF传输或时延测量。
- 失败诊断纠正两项probe假设：wrong-sequence ACK可触发原厂scheduler重组smooth的两FIFO样本，timeout应与其入口包比较、ACK按sent-count消费；有线motion caller的真实USB实例与任意control实例不同，不能把某个control调用默认返回码8重新pin给PARO。固件机器码未因此改变。
- 共用生成器逐字节复现已刷INCA HEX，SHA仍05acd820655c76af585f5e152cb0b72bab26909677588b6d71c754c3bfa81f35。PARO HEX SHA3d665ebf4598a534d8fc7d9a91202a6ae97bbd200a39b2471bfb884cb50840ed、SFX SHAa7a260c69a83dc7042e0929e9379b9c068ffefca3e8c239849fcc59315251243；完整元数据见固件目录analysis/static/paro_trajectory_release.json。


## 最新面板编译交付

- 按用户要求再次从当前源码编译 `build/MouseHub_Latest.exe`（trimpath、Windows GUI、去除符号），与共用 `build/MouseHub_Trajectory.exe` 逐字节相同，SHA-256=`33fb42d19b50d5099bf6b129b47a3b927843e42aa4b370bb09cf5d868c9054d8`。
- 在独立data目录和只读QA模式实际启动，观察到Win32窗口「Mouse Hub · UI 验证」，随后仅结束本次临时QA实例，没有修改用户配置或写入设备。
- 原 `build/MouseHub.exe` 保留；正常运行新版前应退出旧面板实例，避免单实例机制切回旧窗口。编译与启动记录同时保存于两型号release JSON。


## 默认程序名恢复

- 按用户要求重新编译到项目默认 `build/MouseHub.exe`，SHA-256仍为 `33fb42d19b50d5099bf6b129b47a3b927843e42aa4b370bb09cf5d868c9054d8`。实际只读QA窗口启动成功，当前交付清单和README已同步。
- 本次另命名的Latest/Trajectory编译产物已清理；以后编译沿用 `MouseHub.exe`，不再给驱动程序另起版本名称。


## 接收器专用驱动与固定名称

- 按用户要求撤销有线适配，枚举/查询/写入仅接受37B0:0010/mi_02。测试覆盖有线0007/0009、boot0008/000A、其它VID/PID、kbd和伪mi_02字段的拒绝，go test ./...通过。
- 实机最初返回USB Product LAMZU INCA 8K Receiver，原厂设置及TR/v1稳定模式有效；后续查询仅得到A0，未被当作有效读回。显示名称按用户最终要求固定为LAMZU 8K Receiver，不再显示型号未确认。
- 实际编译后--probe只发现1台LAMZU，固定名称正确；只读Win32 QA参数页保留顺滑/稳定、MS与休眠控制。未知TR状态下轨迹按钮禁用，不能把A0或截图当作当前写入成功。
- 默认build/MouseHub.exe最终SHA f08dcfa294d8c8fdaf66cdcb9dcf606e4b5d14e89641af215321698b69a2ba00，程序名不变，用户当前旧进程及配置未强制退出或修改。


## 接收器鼠标身份与面板改进

- 原厂证据闭合：LAMZU两个840鼠标通用CMD81回复包含BE PID0009/0007；原始type5两片、820重组与8字节bridge保持完整12字节回复，305侧带ISA限制的静态重组/GET_REPORT对应。A4为接收器当前无可用连接，不能解释为无持久配对记录。VAXEE原始E1 CMD0A handler返回LE PID1010，USB消费型缓冲在读取后清除；无transaction ID，不保证同CMD回复fresh。
- Go全套测试通过：双品牌严格帧解析、未知/零值语义、同receiver缓存、拔除清缓存、有效未知替代旧型号、换鼠标清旧Applied、A4停止写入和恢复重新确认均覆盖。没有周期身份轮询、配对/设置写入、主动唤醒或有线接入。
- 默认MouseHub.exe的实际--probe得到LAMZU MousePID9/Model LAMZU INCA和VAXEE MousePID4112/Model VAXEE E1，两者current且身份来源正确；LAMZU接收器名仍固定LAMZU 8K Receiver。PARO没有实机，0007只由原厂机器码/重组和parser验证。
- 实际只读Win32面板的两品牌首页与设备信息页均显示对应型号、依据和正确图片；等待/缓存/未知状态与当前回复区分。QA自动应用被禁止，其通用无响应状态提示不是身份读取失败。用户既有窗口与配置保留，本次临时QA已结束。
- 默认程序SHA-256=`0cb0efb70542aabbc0defa393bc008cce63cab222f16ba0b307aa9bae07fb45b`。完整硬件/截图记录 [receiver-mouse-identity.json](../research/receiver-mouse-identity.json)，实际截图在build/ui-identity-research。固件和更新器内容未修改，不需要为了型号查询重新刷写。


## 无响应只显示离线

- 按用户要求，无响应、会话缓存和明确无连接状态的型号/状态/轨迹应用提示只显示「离线」，没有括号、恢复或历史说明；缓存型号图片也不作为当前鼠标呈现。内部身份协议、恢复与设置缓存逻辑保留。
- Go全套测试通过；移除了绑定旧中文提示的断言，设备保留/恢复行为断言继续通过。临时只读状态注入的实际Win32首页/参数页已确认「离线」二字；这是界面场景验证，非实机断连证明。临时入口/EXE均清理，默认真实入口与原始文本一致。
- build/MouseHub.exe重新编译并实际启动成功，SHA-256=`7c7d472476d2847bcd539a04611f7f979365507b61507ca54b33e1e7dc64621c`。临时QA结束，用户配置和正在使用的旧实例保留；离线截图位于build/ui-identity-research/offline-home.png及offline-parameters.png。


## MouseHub前台自动刷新

- WM_ACTIVATEAPP激活时仅排队后台状态刷新，查询型号/连接、电量和实际设置；电量不等待缓存到期。状态刷新不模拟USB重连、不清下发缓存，读回观察值不会被后续普通事件替换成旧目标值。
- 首页「刷新设备」及设备信息页「刷新设备信息」按钮删除。设备插拔与已有恢复路径保留，无新周期轮询。
- Go全套测试通过；新增回归覆盖重复请求合并、短时间内电量更新、实际设置保留、未变化配置不重复下发、暂停自动切换时的读取失败/恢复。真实只读启动观察到型号发现、电量和设置查询完成。
- 自动前台/后台切换测试期间用户报告火绒结束执行进程，验证工具状态亦丢失；未取得准确拦截规则，完整窗口切换和后台静止检查不记为通过。已停止此类窗口自动化，清理临时进程、源文件和EXE，没有修改火绒设置或增加排除项。
- 补充受控处理器烟测：向实际Win32窗口发送失活/激活消息，失活后计数不变，两次激活分别使discover/battery/settings计数从4/12/12到5/15/15、再到6/18/18；执行真实接收器读回，QA禁止写设置。此为WM_ACTIVATEAPP处理器执行证据，不冒充完整实际Alt+Tab测试。默认程序实际首页/信息页控件检查无刷新按钮，信息页截图已强制绘制并目视确认：build/ui-identity-research/foreground-refresh-information.png。
- 最终build/MouseHub.exe SHA-256=`fffb9899460e20ed529ce701cfc42e059d0e1bc9bd89737992f3bdcd238c8644`；程序名、接收器专用、仅「离线」显示和固件内容保持。


## INCA VaxeeSmooth无线模式实机验收

- 新版固件通过0010/mi_02接收器原生控制路径写入smooth0/stable1，分别独立查询9E得到 `A1 00 02 04 01 9E 01 00 54 52` 与 `A1 00 02 04 01 9E 01 01 54 52`。最后恢复无线测试前smooth0，回报率code1、MS0、性能0/0、sleep30逐项与测试前一致。
- 用户指出无响应由分析期间鼠标进入30秒休眠造成；醒着时连续验收通过。没有据A0回显宣布固件无线不兼容；未经证实必要的跨进程互斥实验已撤回，新锁/回归文件与临时程序已删除。
- 默认MouseHub.exe仍为fffb9899460e20ed529ce701cfc42e059d0e1bc9bd89737992f3bdcd238c8644，原面板和配置恢复，接收器专用保持。实际依据在固件报告分组的 `inca_smooth_tuning_wireless.json`；模式往返不等于光学轨迹或实际RF/USB间隔测量。


## 生效配置窗口标题

- OS窗口标题按引擎已保存的配置状态更新，例如「Mouse Hub: 日常配置」「Mouse Hub: 游戏配置」，自定义配置名也直接使用。选中设备的当前profile优先；无已选设备时复用配置解析规则，暂停自动切换时用默认配置。编辑但未保存的配置不改变标题。
- appUpdate在可见性分支之前更新标题，托盘隐藏期间的配置切换也更新；只有配置名称改变时才发送WM_SETTEXT。布局选择设备后同步标题，单实例查找仍使用窗口class。
- 临时无设备IO烟测用真实Win32窗口执行实际appUpdate：日常→游戏→日常、自定义名称、暂停自动切换和未保存编辑隔离均通过；没有前台焦点自动化。临时源已删除，go test ./...通过。
- 默认build/MouseHub.exe已编译，SHA-256=`a9a3a78dd049bfd02f4d780f4fa578c07002a9972ce5e9ffbe01590083e4b3aa`。现有旧面板实例保留；退出旧实例后打开默认文件加载新版。用户配置和设备固件未修改。
