# LAMZU INCA / PARO 协议与识别范围

## USB 身份与写入目标

| USB VID:PID | 默认设备名 | 证据与范围 |
|---|---|---|
| `37B0:0007` | PARO 有线应用身份；面板不接受 | PARO 原厂 Config 与 DeviceDescriptor 确认；仅作固件证据 |
| `37B0:0009` | INCA 有线应用身份；面板不接受 | 历史实机刷写/读回证据；当前驱动不控制此接口 |
| `37B0:0010` | LAMZU 8K Receiver | 固定显示名称；当前实机USB Product为LAMZU INCA 8K Receiver，能力另经TR/v1确认 |

PARO 原厂资料是 `LAMZU INCA 00.00.00.19/analysis/decrypted/paro/Config.ini` 和 `LAMZU_PARO_Mouse_840_APP_v0.0.0.19_20260605.hex`。Nordic 配置指定 NewVID=`37B0`、NewPID=`0007`、BLPID=`0008`，`FWFile2` 为空；不能用未启用的 Sonix 样例字段推断 PARO 接收器。原厂 DeviceDescriptor 地址 `0x1D6C8` 报告 VID=`37B0`、PID=`0007`、bcdUSB=`0200`、EP0 max packet=`64`，只确证设备身份，不能单凭 DeviceDescriptor 得到接口编号。

PARO 原厂 HID 接口注册表分别在 `0x1D478`、`0x1D4C0`、`0x1D508` 注册接口 0、1、2；第三条为 `01 02 01 03 83 00 00 00`。应用回调 `0xE8EC` 将实例与接口 2 的 control instance `0x1D520` 比较，匹配分支写 pending `0x20003526`，再写 status A0 到命令缓冲区 `0x200055BC`，由此闭合控制接口编号 2，即 Windows `mi_02`。

共用 Feature descriptor `0x1D440` 为 `06ffff0900a10109001500250175089540b102c0`：8 位 × 64 个 Feature 数据项，无 Report ID item。内部 HID 缓冲区 `0x1D4F4` 为 65 字节，Feature 传输记录 `0x1D788` 指向命令缓冲区且指定 64 字节 payload；Windows 65 字节报告包含前导 ReportID 0。静态证据记录为 `LAMZU INCA 00.00.00.19/analysis/static/paro_trajectory_mapping.md` 与 `.json`。此证据确认固件报文依据，没有 PARO 实机枚举或主机 USB 传输实测。面板仅通过0010接收器的 `mi_02` 使用该共用协议，不接入有线接口、不试探其它接口，也不新增推测的接收器 PID。

枚举、查询与写入共用接收器白名单和接口判断，仅接受37B0:0010/mi_02，排除有线0007/0009、键盘集合、非控制接口及引导PID。指定path应用先重新读取该接收器HID属性，不符时拒绝，不转向其它鼠标。接收器显示名称固定LAMZU 8K Receiver；鼠标身份由原生通用CMD81的回复PID独立识别，轨迹能力另由TR/v1判断。型号查询仅随已有发现/刷新/恢复流程执行，不增加定时扫描；设备ID及已有profile绑定不变。

## 共用 TR/v1

轨迹命令使用现有 LAMZU CMD/VAL 封装。写入 64 字节 payload 的头为 `00 00 02 02 01 1E 01 MODE`，`MODE=0` 是 smooth，`MODE=1` 是 stable。65 字节 Windows HID 报告另有前导 ReportID `00`。

查询 `CMD=9E` 的有效 payload 为：

```text
A1 00 02 04 01 9E 01 MODE 54 52
```

`54 52` 是固定 TR/v1 标记，没有独立版本字节。解码器接受完整十字节 payload、补齐至最多 64 字节的 payload，以及包含前导 ReportID 0 的 65 字节报告；拒绝错误 header、错误命令、错误 value count、非法 MODE、缺失 TR、A2/A3 旧回复与非零 ReportID。没有有效回复就不声明轨迹能力，不能从旧回复或空值推定 smooth。

非空轨迹请求在写其它参数前确认模式合法和 TR/v1 能力；下发 `1E` 后再次查询 `9E`，读回相符才确认应用成功。原厂或旧固件没有 TR/v1 时仍可应用其它已有参数，但不发送 `1E`；若 profile 明确要求轨迹，返回不支持错误。旧配置及旧配置导入的空 `trajectory` 表示未指定，保留设备当前模式，其他参数仍需读回确认。

## 配置、界面与重连

INCA / PARO 复用同一 LAMZU 参数类型、解码器、应用与读回路径。参数校验按品牌，不按 Name；界面按 LAMZU 品牌与 `TrajectorySupported` 开放「顺滑轨迹」/「稳定轨迹」，Motion Sync 与休眠保持独立。

电脑端 profile 持久保存用户选择，编辑有明确 ID 的设备时使用其独立参数，品牌默认参数只作为初始回退。固件模式只在 RAM 中保留，上电恢复 stable；前台配置切换、用户保存及既有重连恢复机制可以重新应用保存的模式。没有新增持续查询或定期枚举，未证明相关性的接口不合并。

## 固件时序与验证界限

INCA 与 PARO 各自从其原厂固件独立适配。两模式采用 E1 普通周期表：125/250/500/1000/2000 Hz 为 7990/3990/1990/996/498 µs；普通 4K 为 smooth 248 µs、stable 248.875 µs；8K 按普通 4K 比例扩展为 smooth 124 µs、stable 124.4375 µs，不采用 E1 特殊 123 µs 包路径。

这些值是无线 profile code 与实际 divider 匹配时的 TIMER1 平均请求周期配置，不是 USB 回报间隔实测。有线、未知 code 或 divider 不匹配时保留原厂 125 µs 节拍并清分数相位；合法模式切换只更新本模式与相位，不占未知 NVM 或功能 profile 位。传感器 FPS、TIMER0/2、16 位累积/ring/ACK 保留，所有 direct/async downshift CPI 上限统一为 30,000。

INCA E1Table 已实机刷写并完成有线轨迹切换/读回。PARO 没有实机，代码和原厂资料适配不等于刷写、接口或性能验收；实际轨迹改善、无线回报与时延尚未测量。真实运行证据由 [验证记录](validation.md) 单独记录，产物与完整周期表见 [README](../README.md)。

## 鼠标身份状态与面板

VAXEE官方型号请求为ReportID0E、A5 0A、READ01、长度02；有效回复byte5/6为little-endian鼠标PID。原始E1 handler返回1010；未知有效PID不会返回错误或复用旧型号。零值/FFFF、错误帧和超时均表示未取得有效身份，没有证据将它们单独命名为明确断连。E1 USB回复缓冲区在读取后消费/清除，但没有transaction ID，有效回复不保证是刚完成的无线事务。

LAMZU型号请求使用route2/group0通用CMD81，不是设置组group1的CMD81；固件版本回复中的PID来自所连鼠标自身。只有严格匹配状态/route/group/command/长度的回复才能识别。原厂INCA的PID0009、PARO的PID0007分别来自自己的机器码路径，既不读取有线设备也不使用接收器USB Product推断。匹配请求的A4为接收器报告当前无可用鼠标连接，A0回显/超时不等于A4。

`Device`另提供mouseModel、identityStatus、identityCode及identitySource。current表示观察到有效身份回复，unknown表示有效但型号表未收录，unavailable表示本次无有效身份，cached明确表示同receiver本会话的上一回复，disconnected仅用于有证据的无连接回复。拔除接收器丢弃缓存；有效新原码替换旧身份并使旧设置/电量确认失效；明确无连接时不继续下发自动配置。面板将接收器名称、鼠标型号、识别依据分开展示，未知型号不显示成E1或INCA图片。

