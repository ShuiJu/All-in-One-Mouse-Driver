# 官方功能对照

官方功能核对时间：2026-09-13；INCA / PARO 轨迹适配更新：2026-10-06。以下区分官方功能、代码适配与实物验证：历史三台实物为 INCA / E1 / NP-01S V2；PARO 没有实机，不能引用该矩阵作为验收。

| 功能 | LAMZU 官方 | VAXEE 官方 | 本项目状态 |
|---|---|---|---|
| 性能模式、Motion Sync | 有 | 有 | 已整合，本机三台实物矩阵通过 |
| 回报率 | 型号相关 | 当前页面含 500–4000 Hz，其他型号还有 8K | 保留原项目档位；VAXEE 500 Hz 尚未加入 |
| DPI 与 DPI 档位 | 有 | 有，当前页面 4 档 | 尚未接入 |
| LOD | 有 | 有，低/高 | 尚未接入 |
| 按键响应时间 / Debounce | 有 | 当前页面 0 / 2 / 4 ms | 尚未接入；不属于改键 |
| 直线修正 / Angle Snap | 有 | 本次页面未见 | 尚未接入 |
| 波纹控制 / Ripple Control | 有 | 本次页面未见 | 尚未接入 |
| 角度调整 / Angle Tune | 有，受型号配置开关限制 | 本次页面未见 | 尚未接入 |
| 休眠时间 | 有 | 本次页面未见 | LAMZU 已整合 |
| 追踪轨迹 | 型号相关 | 有公开 `trackTrailSetting` 协议 | VAXEE 已整合；INCA / PARO 新固件通过 TR/v1 标记确认能力后开放顺滑轨迹（smooth）/稳定轨迹（stable）；PARO 尚无实机验证 |
| 板载配置 / 重置默认 | 有 | 官方提供还原默认 | 尚未接入硬件板载管理；本项目配置为电脑端策略 |
| 鼠标配对 | 有 | 本次页面未见 | 尚未接入 |
| 固件版本读取、检查升级 | 有 | 有 | 尚未接入 GUI |
| 固件升级 | 有 | 页面提供版本检查，不将其等同于所有型号可网页刷写 | 尚未接入 |
| 接收器灯效 | 官方存在型号相关选项 | 本次页面未见 | 尚未接入 |
| 硬件参数实时读回 | 有 | 有 | 新增支持已整合参数的按需读回确认，不做持续轮询 |
| 自定义按键、宏 | 有 | 有自定义按键 | 按用户要求排除 |

## E1 与 NP-01S V2 的识别

官方控制集合为 VID `0x3057`、UsagePage `0xFF05`、Usage `0x01`。

查询报文（包含 Windows ReportID）：`0E A5 0A 01 02 00 ...`。
回应中的第 5、6 字节（从 0 开始）是小端鼠标 PID：`10 10` 为 E1，`11 10` 为 NP-01S V2。

两只实物的 USB 接收器都报告 PID `0x2001`、名称 `VAXEE 4K Wireless 1.0`，因此仅依靠 USB ProductName/PID 无法区分。2026-09-13 的实物查询成功返回两种鼠标 PID，证据在 `research/device-identity-verification.json`。休眠或读取失败时保留“型号未确认”，不猜测。

## 轨迹报文修正

官方 `trackTrailSetting` 使用 `byte3=0x02` 表示写入，`byte5=0` 表示顺滑灵敏、`byte5=1` 表示稳定易控。旧代码将模式放在 byte3，导致稳定选项实际上使用 READ 操作码。新实现使用官方格式并在下发后读回核对。实物基线曾读到 E1 的值为 1、NP-01S V2 的值为 0；本机 E1 与 NP-01S V2 的全部已开放组合写入与读回测试现已通过。

## INCA / PARO 共用新固件轨迹接口

驱动查询 `CMD=0x9E`，仅接受 `A1 00 02 04 01 9E 01 MODE 54 52` 的响应头：`MODE=0` 为 smooth，`MODE=1` 为 stable，`54 52`（TR）固定标记定义协议 v1，没有独立版本字节。原厂无线回复路径只允许此十字节 payload；Windows HID 报告可补零至 64 字节，65 字节报告另含前导 ReportID 0。旧固件 A2/A3 回复、缺失标记、错误命令、非法模式或无回复均不声明能力，`trajectorySupported=false`，界面禁用该选项；其他参数仍可使用。

写入沿用 64 字节 payload `00 00 02 02 01 1E 01 MODE`，若 HID 长度为 65 则增加前导 ReportID 0。非空轨迹请求在写入任何参数前检查参数与固件能力，写入后用 0x9E 再次查询，只有读回与请求一致才确认成功。无线请求时序由固件控制，驱动不写 timer 数值。

轨迹复用电脑端 profile 的 `trajectory` 字段并持久化 smooth/stable；LAMZU 标签为「顺滑轨迹」/「稳定轨迹」，VAXEE 仍为「顺滑灵敏」/「稳定易控」。LAMZU 参数不按设备 Name 限制接受轨迹，界面按 Brand 与实际 TR/v1 能力决定是否开放。参数、休眠及轨迹继续使用原有保存路径，自动切换和设备重连会重应用用户明确选择的模式。旧 LAMZU 配置及旧配置导入保留空字符串，含义是未指定，不写轨迹，也不强制其读回与某个默认值匹配。支持的设备按实际查询返回轨迹；不支持的设备读回轨迹为空，不把零值推断为 smooth。

固件轨迹仅在本次开机的 RAM 中保留，断电后恢复 stable，不使用未知 NVM 字段。驱动会在重连后重新应用已保存的明确选择。INCA E1Table 已刷写并通过有线轨迹写入与读回；PARO 依据原厂固件适配，没有 PARO 实机。实际轨迹改善、无线回报间隔和时延均须分别测量，不能由协议读回或历史参数矩阵推定。

INCA 配套固件为 `INCA_Mouse_840_APP_v0.0.0.19_TrajectorySwitch_E1Table_NoDownshift_FactoryFPS.hex`；PARO 从自己的原厂 `LAMZU_PARO_Mouse_840_APP_v0.0.0.19_20260605.hex` 独立适配。两型号 smooth 与 stable 均按 E1 普通周期表配置，映射不反转：无线 125/250/500 Hz 两模式分别为 7990/3990/1990 µs；1K/2K 两模式为 996/498 µs；4K 为 smooth 248 µs、stable 248.875 µs；8K 为 smooth 124 µs、stable 124.4375 µs。8K 采用普通 4K 比例扩展，不采用 E1 特殊 123 µs 包路径；完整七档 code/divider 表见 [README](../README.md)。这些是 TIMER1 平均请求周期配置，不是实测 USB 回报间隔。未识别或实际 divider 不匹配以及有线状态保持 125 µs，并清零分数相位；模式写入也先复位到 125 µs，下次 IRQ 应用目标。传感器 FPS、TIMER0/2 与原 16 位累积/ring/ACK 保留，所有 direct/async downshift CPI 上限统一为 30,000。

驱动只接受 LAMZU 接收器 `37B0:0010` 的 `mi_02`，不支持有线控制。按用户要求固定显示「LAMZU 8K Receiver」。查询和写入均使用选中的接收器，拒绝有线0007/0009及引导模式0008/000A。PARO 固件与共用轨迹协议的软件适配保留，原厂控制接口静态证据不代表驱动需要连接该有线接口；没有 PARO 实机枚举、传输或刷写验证。详细证据范围和报文说明见 [协议说明](protocol-notes.md)。

鼠标型号独立从接收器返回的原生身份读取：LAMZU通用CMD81固件版本回复内的big-endian PID0009/0007对应INCA/PARO；VAXEE官方CMD0A返回little-endian鼠标PID，未知有效PID保留原码。面板分开展示接收器名称、鼠标型号和识别依据；无响应仅显示「离线」，不加括号或解释，内部会话缓存不作为当前型号、图片或已应用状态展示，拔除后不继承；LAMZU匹配A4无可用连接回复清旧身份/已应用并等待重新连接。身份查询随原有发现/刷新/恢复，不增加定时器或主动唤醒。真实接收器识别INCA/E1已观察，PARO只有原厂CPU/重组与parser证明，没有实机。

## 公开来源

- https://vcc.vaxee.co/settings.php
- https://vcc.vaxee.co/mode.php
- https://vcc.vaxee.co/api/paramter.js?v=3.6
- https://vcc.vaxee.co/api/api.js?v=3.6
- https://vcc.vaxee.co/api/api_core.js?v=3.6
- https://vcc.vaxee.co/assets/js/vcc-mouse/connect.js?v=3.6
- https://lamzu.net/#/project/items
- https://lamzu.net/static/js/index-93e057a1.js （v1.0.32；功能文案与型号能力开关）

Wireshark 与 USBPcap 已安装。本次直接找到了公开查询协议，尚未进行 USB 抓包。
