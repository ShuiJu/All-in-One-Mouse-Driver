# 官方功能对照

核对时间：2026-09-13。以下区分官方驱动具有该功能与 INCA / E1 / NP-01S V2 每台实物已验证支持；未做实物验证的项目不保证所有型号支持。

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
| 追踪轨迹 | 型号相关 | 有公开 `trackTrailSetting` 协议 | VAXEE 已整合，修正旧版报文 |
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
