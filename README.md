# YZU-Network

[![CI](https://github.com/lzy18001500226/YZU-Network/actions/workflows/ci.yml/badge.svg)](https://github.com/lzy18001500226/YZU-Network/actions/workflows/ci.yml)

Windows 下的扬州大学校园网自动认证工具，支持 Wi-Fi 和以太网。程序定时检测网络状态，发现认证页后直接请求校园网认证接口，不依赖浏览器、Selenium 或 ChromeDriver。

> 仅用于本人已获授权的校园网络账号。学校认证接口、服务名称和策略可能变化，请以实际页面为准。

## 功能

- 按配置的校园网 IPv4 网段或路由器 SSID，自动识别当前校园网络接口。
- 不切换 Wi-Fi、不禁用网络适配器；手机热点不匹配时不会被当作校园网。
- 支持每 60 秒检测一次网络，认证前等待 5 秒。
- 连续失败达到阈值后记录接口状态，并只对识别出的校园网接口尝试 DHCP 续租。
- 支持当前用户后台开机自启，登录 Windows 30 秒后启动。
- 日志默认写入 `%APPDATA%\YZU-WLAN\yzu-wlan.log`。

## 快速开始

1. 下载源码并进入项目目录。
2. 安装 Go 1.22 或更高版本。
3. 从 `.env.example` 创建私有配置文件，填写校园网账号。
4. 运行一次检查，确认日志出现 `internet is available` 或 `login verified`。
5. 确认无误后安装后台开机自启。

## 前置条件

- Windows 10/11
- Go 1.22 或更高版本，仅在自行构建时需要
- 已能在浏览器中正常完成一次校园网认证
- 校园网地址段或校园路由器 SSID 已配置在 `.env`

## 配置

私有配置固定放在：`%APPDATA%\YZU-WLAN\.env`。不要在项目目录创建、提交或分享 `.env`。

```powershell
$configDir = Join-Path $env:APPDATA 'YZU-WLAN'
New-Item -ItemType Directory -Force $configDir
Copy-Item .\.env.example (Join-Path $configDir '.env')
notepad (Join-Path $configDir '.env')
```

至少填写：

```dotenv
SSO_USERNAME=your_student_id
SSO_PASSWORD=your_password
CAMPUSNET_SERVICE=电信互联网服务
```

`CAMPUSNET_SERVICE` 必须与认证页服务名称完全一致。常见名称见 [`.env.example`](.env.example)。

不要把账号、密码、完整日志或包含个人信息的截图提交到 GitHub。密码变更后，只需修改 AppData 下的 `.env`，不需要修改源码。

### Wi-Fi 与有线网络

程序不会主动连接或断开 Wi-Fi。它按校园 IPv4 地址段和已配置的校园路由器 SSID 识别校园网接口，并分别通过每条接口检查网络、按需认证；一条路径已联网不会跳过另一条路径。手机热点不匹配时不会发送校园认证请求。

默认按扬州大学校园网地址段 `10.135.0.0/17` 识别有线或校园 Wi-Fi 接口。若校园路由器给设备分配的是其他网段，可在 `CAMPUS_SSIDS` 中填入该路由器的 Wi-Fi 名；多个网段或名称用逗号分隔。示例配置不包含个人路由器名称。

## 构建与运行

```powershell
go test ./...
powershell -ExecutionPolicy Bypass -File .\build.ps1
```

单次检查：

```powershell
.\dist\yzu-wlan.exe -once -e "$env:APPDATA\YZU-WLAN\.env"
```

持续运行：

```powershell
.\dist\yzu-wlan.exe -e "$env:APPDATA\YZU-WLAN\.env"
```

也可以使用 `run_once.bat` 或 `run_daemon.bat`。首次运行时它们会构建程序并复制到 `%APPDATA%\YZU-WLAN`，不会复制账号密码文件。

### 命令参数

| 参数           | 作用                                     |
| -------------- | ---------------------------------------- |
| `-e <path>`  | 指定配置文件，默认读取当前目录的`.env` |
| `-once`      | 只执行一次检测和认证，然后退出           |
| `-logout`    | 登录自助服务并尝试下线当前账号的在线设备 |
| `-install`   | 创建 Windows 计划任务，需要管理员权限    |
| `-uninstall` | 删除 Windows 计划任务，需要管理员权限    |

## 开机自启

推荐当前用户启动项，无需管理员权限，且窗口隐藏：

```powershell
powershell -ExecutionPolicy Bypass -File .\install_user_startup.ps1
```

卸载当前用户启动项：

```powershell
powershell -ExecutionPolicy Bypass -File .\uninstall_user_startup.ps1
```

`install_task.ps1` 会创建计划任务，需要以管理员身份运行。计划任务包含开机检查和每小时一次的单次检查；守护进程模式的间隔由 `CHECK_INTERVAL` 控制。

当前用户启动项运行常驻守护进程；计划任务中的开机检查和每小时检查各自只运行一次后退出。二者是独立的启动方式。

安装脚本只复制程序，不会把项目目录中的 `.env` 复制到 AppData。配置文件不存在时，脚本会直接报错，避免误用或泄露凭据。

## 网络恢复与睡眠

默认每 60 秒检查一次。连续 5 次失败后，守护进程最多每 10 分钟对本轮识别出的校园网接口执行一次 `ipconfig /renew`；手机热点及其他未识别接口不会被续租：

```dotenv
RECOVERY_FAILURES=5
RECOVERY_COOLDOWN=10m
RECOVERY_TIMEOUT=45s
```

Windows 处于睡眠状态时，用户态程序和网络适配器都会暂停，无法维持在线。唤醒后程序会在下一个周期继续检测并在需要时认证。

## 日志与排障

```powershell
Get-Content "$env:APPDATA\YZU-WLAN\yzu-wlan.log" -Encoding UTF8 -Tail 100
```

- `internet is available through ...`：指定网卡的联网检测成功。
- `login verified`：认证请求及后续检测均成功。
- `get portal URL ... timeout`：认证入口不可达，不能据此判断账号或密码错误。
- `network recovery`：连续失败后尝试 DHCP 续租。
