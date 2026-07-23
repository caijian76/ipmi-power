# IPMI Power Manager

基于 IPMI 协议的电源管理守护进程，支持定时开关机操作。

## 功能特性

- 支持 IPMI 协议远程管理服务器电源
- 支持定时任务：开机、关机（硬关机/软关机）、重启、复位
- 支持多服务器配置
- 支持按星期设置定时任务执行日期
- 电源状态查询
- 日志记录

## 支持的操作

| 操作 | 说明 |
|------|------|
| `on` | 开机 |
| `off` | 硬关机 |
| `soft` | 软关机 |
| `cycle` | 电源循环（重启） |
| `reset` | 硬重置 |
| `status` | 查询电源状态 |

## 配置文件

配置文件为 JSON 格式，示例：

```json
{
    "servers": [
        {
            "name": "服务器名称",
            "host": "172.16.100.102",
            "username": "ADMIN",
            "password": "密码",
            "interface": "lanplus",
            "scheduled_operations": [
                {
                    "time": "08:00",
                    "action": "on",
                    "days_of_week": [1, 2, 3, 4, 5]
                },
                {
                    "time": "20:00",
                    "action": "soft",
                    "days_of_week": [1, 2, 3, 4, 5]
                }
            ]
        }
    ]
}
```

### 配置说明

- `name`: 服务器名称
- `host`: BMC IP 地址
- `username`: IPMI 用户名
- `password`: IPMI 密码
- `interface`: 接口类型 (`lan` 或 `lanplus`，默认为 `lanplus`)
- `scheduled_operations`: 定时任务列表
  - `time`: 执行时间，格式为 `HH:MM`
  - `action`: 操作类型
  - `days_of_week`: 星期几执行
    - `0` = 周日
    - `1` = 周一
    - `2` = 周二
    - `3` = 周三
    - `4` = 周四
    - `5` = 周五
    - `6` = 周六
    - 空数组或不设置表示每天执行

## 使用方法

```bash
# 启动守护进程（使用默认配置）
./ipmi-power

# 使用指定配置文件
./ipmi-power -config myconfig.json

# 自定义检查间隔（秒）
./ipmi-power -interval 60

# 显示所有服务器状态并退出
./ipmi-power -status
```

## 命令行选项

| 选项 | 说明 | 默认值 |
|------|------|--------|
| `-config` | 配置文件路径 | `config.json` |
| `-interval` | 检查定时任务的间隔（秒） | `30` |
| `-status` | 显示所有服务器当前状态并退出 | `false` |

## 构建

```bash
go build -o ipmi-power main.go
```

### 静态编译

使用 Docker 进行静态编译，生成不依赖 glibc 的二进制文件：

```bash
docker run --rm -v $(pwd):/src -w /src golang:1.23-alpine \
  go build -ldflags '-linkmode external -extldflags "-static"' -o ipmi-power main.go
```

或者使用 CGO 禁用模式：

```bash
CGO_ENABLED=0 go build -ldflags '-linkmode external -extldflags "-static"' -o ipmi-power main.go
```

## 依赖

- Go 1.23+
- github.com/bougou/go-ipmi v0.8.3