<p align="center">
  <h1 align="center">Lanflow</h1>
  <p align="center">
    <strong>LAN Traffic Monitor</strong> — 局域网流量监控工具
  </p>
  <p align="center">
    <a href="#features">Features</a> •
    <a href="#architecture">Architecture</a> •
    <a href="#quick-start">Quick Start</a> •
    <a href="#deployment">Deployment</a> •
    <a href="#configuration">Configuration</a> •
    <a href="#api">API</a> •
    <a href="#faq">FAQ</a>
  </p>
</p>

---

## What is Lanflow?

Lanflow 是一个轻量级的局域网流量监控工具，部署在局域网内一台 Linux 机器上，通过将该机器设为网关并抓包的方式，**实时统计每个 IP 的上行/下行流量**，并提供 Web 仪表盘进行可视化展示。

**典型场景：** 实验室/办公室多台电脑共享一个路由器上网，路由器没有 per-IP 流量统计功能，需要知道每个人用了多少流量。

### Why Lanflow?

- **零客户端** — 只需部署在一台机器上，局域网内其他设备无需安装任何软件
- **单一二进制** — Go 编译，前端资源嵌入，一个文件搞定部署
- **高性能** — 基于 libpcap 内核级抓包，Go 原生并发，轻松处理百兆网络
- **低资源** — 运行时内存 < 20MB，SQLite 存储，无外部依赖
- **Web 仪表盘** — 实时监控 + 历史统计 + 设备管理，开箱即用

## Features

- **实时流量监控** — WebSocket 推送，2 秒刷新，展示每个 IP 的瞬时上行/下行速率
- **历史统计查询** — 支持按天/周/月查看各设备流量，ECharts 图表可视化
- **设备命名管理** — 给 IP 起别名（如"张三-办公机"），方便识别
- **数据持久化** — SQLite 存储，默认保留 90 天，支持自定义
- **自动清理** — 每日自动清理过期数据
- **优雅关停** — 支持 SIGINT/SIGTERM 信号，关停前自动 flush 未保存数据
- **Systemd 集成** — 提供 service 文件，开机自启 + 崩溃重启

## Architecture

```
                          ┌──────────────────┐
                          │    Router         │
                          │  192.168.1.1      │
                          │  (校园网/外网)     │
                          └────────┬─────────┘
                                   │
                          ┌────────▼─────────┐
                          │  Lanflow Server   │
                          │  192.168.1.108    │
                          │  (透明网关)        │
                          └────────┬─────────┘
                                   │
              ┌────────────┬───────┴───────┬────────────┐
              │            │               │            │
        ┌─────▼────┐ ┌─────▼────┐  ┌──────▼───┐ ┌──────▼───┐
        │ PC-01    │ │ PC-02    │  │ GPU-01   │ │ GPU-02   │
        │ Windows  │ │ Windows  │  │ Linux    │ │ Linux    │
        └──────────┘ └──────────┘  └──────────┘ └──────────┘
```

**工作原理：** 将运行 Lanflow 的机器设为局域网默认网关（修改路由器 DHCP 设置），所有外网流量都会经过这台机器。Lanflow 通过 libpcap 抓包，解析 IP 层，按源/目的 IP 统计上行和下行字节数。

### 内部模块

```
┌─────────────────────────────────────────────────────┐
│                lanflow (单一 Go 二进制)                │
│                                                       │
│  ┌─────────┐   ┌────────────┐   ┌──────────────┐    │
│  │ Capture  │──▶│ Aggregator │──▶│   SQLite     │    │
│  │ (libpcap)│   │ (内存计数)  │   │  (持久化)    │    │
│  └─────────┘   └──────┬─────┘   └──────┬───────┘    │
│                       │                │             │
│                 ┌─────▼────────────────▼─────────┐   │
│                 │        HTTP Server             │   │
│                 │  REST API · WebSocket · 前端    │   │
│                 └────────────────────────────────┘   │
└─────────────────────────────────────────────────────┘
```

| 模块 | 职责 |
|------|------|
| **Capture** | libpcap 混杂模式抓包，BPF 过滤 IP 包，解析 IPv4 层 |
| **Aggregator** | 内存中维护 per-IP 计数器，每 60 秒 flush 到数据库 |
| **Storage** | SQLite WAL 模式，分钟粒度存储，定期清理过期数据 |
| **API Server** | REST API + WebSocket 实时推送 + 嵌入式前端 |

## Quick Start

### 前提条件

- Linux 服务器（Ubuntu 18.04+，CentOS 7+ 等）
- 已安装 `libpcap-dev`（Ubuntu）或 `libpcap-devel`（CentOS）
- Go 1.21+（仅编译时需要）
- root 权限（抓包需要）

### 编译

```bash
# 克隆仓库
git clone https://github.com/yourname/lanflow.git
cd lanflow

# 编译（生成单一二进制文件，前端已嵌入）
go build -o lanflow ./cmd/lanflow/

# 运行测试
go test ./... -v
```

### 快速运行

```bash
# 1. 修改配置文件
cp config.yaml config.yaml.bak
vim config.yaml   # 修改 interface、lan_cidr、gateway_ip

# 2. 开启 IP 转发
sudo sysctl -w net.ipv4.ip_forward=1

# 3. 设置 NAT 转发（将 eth0 替换为你的外网网卡）
sudo iptables -t nat -A POSTROUTING -o eth0 -j MASQUERADE

# 4. 启动
sudo ./lanflow --config config.yaml

# 5. 打开浏览器访问
# http://<服务器IP>:8080
```

## Deployment

### 推荐：Systemd 部署

```bash
# 1. 复制文件到部署目录
sudo mkdir -p /opt/lanflow
sudo cp lanflow config.yaml /opt/lanflow/
sudo mkdir -p /opt/lanflow/{data,logs}

# 2. 修改配置
sudo vim /opt/lanflow/config.yaml

# 3. 安装 systemd service
sudo cp lanflow.service /etc/systemd/system/
sudo systemctl daemon-reload
sudo systemctl enable lanflow   # 开机自启
sudo systemctl start lanflow    # 启动

# 4. 查看状态
sudo systemctl status lanflow
sudo journalctl -u lanflow -f   # 查看日志
```

### 路由器配置

在路由器管理界面（通常是 `192.168.1.1`）的 **DHCP 服务器** 设置中：

| 项目 | 修改前 | 修改后 |
|------|--------|--------|
| 网关 | `0.0.0.0`（或路由器 IP） | Lanflow 服务器的 IP（如 `192.168.1.108`） |

保存后，局域网设备在 DHCP 续租时会自动切换网关。若想立即生效，可在各设备上断开重连网络。

> **回退方案：** 如需停止监控，将路由器 DHCP 网关改回 `0.0.0.0` 或路由器 IP 即可。

### 网络拓扑要求

```
[外网] ←→ [路由器] ←→ [Lanflow 服务器] ←→ [局域网设备]
                          ↑
                    所有外网流量经过这里
```

Lanflow 服务器需要：
- 有固定 IP（建议在路由器中绑定 MAC-IP）
- 与路由器在同一子网
- 已开启 IP 转发（`net.ipv4.ip_forward=1`）

## Configuration

`config.yaml` 配置说明：

```yaml
# 网络配置（必填）
interface: "eth0"            # 抓包网卡名称，用 `ip addr` 查看
lan_cidr: "192.168.1.0/24"   # 局域网网段
gateway_ip: "192.168.1.1"    # 路由器 IP

# 存储配置
db_path: "./data/lanflow.db" # SQLite 数据库路径
retention_days: 90           # 数据保留天数

# Web 服务
listen: ":8080"              # 监听地址和端口

# 日志
log_level: "info"            # 日志级别：debug / info / warn / error
log_dir: "./logs"            # 日志文件目录
```

也可通过命令行参数覆盖：

```bash
sudo ./lanflow --config /path/to/config.yaml --log-level debug
```

### 如何查看网卡名称

```bash
ip addr show
# 找到有局域网 IP 的网卡，如 eth0、enp0s3、ens33 等
```

## Web Dashboard

访问 `http://<服务器IP>:8080`，包含三个标签页：

### 实时监控

实时展示所有设备的流量状态：

| 列 | 说明 |
|----|------|
| 设备名称 | 已命名显示名称，未命名显示 IP |
| IP 地址 | 设备的局域网 IP |
| 上行/下行速率 | 当前瞬时速率（bytes/s） |
| 今日上行/下行 | 当天累计流量 |
| 今日总计 | 上行 + 下行总和 |

- WebSocket 自动推送，每 2 秒刷新
- 按总流量降序排列
- 绿色圆点表示 WebSocket 连接正常

### 历史统计

- 选择时间范围（天/周/月）和日期进行查询
- 柱状图对比各设备流量
- 点击某设备查看详细的时间趋势折线图

### 设备管理

- 给 IP 地址设置名称和备注
- 方便识别哪台机器是谁的

## API

所有 API 返回 JSON 格式 `{"data": ...}`。

### REST API

#### 获取实时流量

```
GET /api/realtime
```

```json
{
  "data": [
    {
      "ip": "192.168.1.10",
      "name": "GPU-01",
      "tx_bytes": 15234,
      "rx_bytes": 892341,
      "tx_packets": 120,
      "rx_packets": 650
    }
  ]
}
```

#### 查询统计数据

```
GET /api/stats?range=day&date=2026-04-04
GET /api/stats?range=week&date=2026-04-04
GET /api/stats?range=month&date=2026-04-04
```

#### 查询单个 IP 详细记录

```
GET /api/stats/{ip}?range=day&date=2026-04-04
```

返回该 IP 在指定时间范围内每分钟的流量记录。

#### 设备管理

```
GET /api/devices                    # 获取设备列表
PUT /api/devices/{ip}               # 设置设备名称
    Body: {"name": "GPU-01", "note": "lab room 301"}
```

### WebSocket

```
WebSocket /ws/realtime
```

每 2 秒推送一次所有 IP 的实时流量数据，数据格式与 `GET /api/realtime` 相同。

## Project Structure

```
lanflow/
├── cmd/lanflow/main.go          # 程序入口，模块组装，信号处理
├── internal/
│   ├── aggregator/              # 流量聚合：内存计数器 + flush
│   │   ├── aggregator.go
│   │   └── aggregator_test.go
│   ├── api/                     # HTTP 服务：REST + WebSocket + 前端
│   │   ├── server.go            # 路由注册，静态文件嵌入
│   │   ├── handlers.go          # REST API 处理器
│   │   ├── handlers_test.go
│   │   ├── websocket.go         # WebSocket Hub + 广播
│   │   ├── websocket_test.go
│   │   └── static/              # 前端资源（嵌入到二进制）
│   │       ├── index.html
│   │       ├── css/style.css
│   │       └── js/
│   │           ├── app.js       # 标签页路由 + 工具函数
│   │           ├── realtime.js  # 实时监控页
│   │           ├── history.js   # 历史统计页
│   │           └── devices.js   # 设备管理页
│   ├── capture/                 # libpcap 抓包
│   │   └── capture.go
│   ├── config/                  # YAML 配置加载
│   │   ├── config.go
│   │   └── config_test.go
│   ├── logger/                  # 结构化日志
│   │   ├── logger.go
│   │   └── logger_test.go
│   └── storage/                 # SQLite 存储
│       ├── storage.go
│       └── storage_test.go
├── config.yaml                  # 默认配置文件
├── lanflow.service              # systemd service 文件
├── go.mod
└── go.sum
```

## FAQ

### Lanflow 挂了怎么办？

如果使用 systemd 部署，服务会在 5 秒后自动重启。如果机器本身宕机，局域网设备会断网。恢复方法：

1. 重启 Lanflow 服务器
2. 或在路由器中把 DHCP 网关改回 `0.0.0.0`

### 如何让设备立即切换网关？

修改路由器 DHCP 网关后，设备需要等 DHCP 续租才会生效。加速方式：
- **Windows**: 运行 `ipconfig /release && ipconfig /renew`
- **Linux**: 运行 `sudo dhclient -r && sudo dhclient`
- **macOS**: 系统偏好设置 → 网络 → 断开再连接

### 代理/VPN 流量能统计到吗？

能。无论用户是否使用 Clash、V2Ray 等代理，流量都会经过网关，**总流量统计是准确的**。但无法区分哪些是直连流量、哪些是代理流量。

### 数据库文件在哪？

默认在 `./data/lanflow.db`（相对于工作目录），使用 systemd 部署时在 `/opt/lanflow/data/lanflow.db`。可以用任何 SQLite 工具查看：

```bash
sqlite3 /opt/lanflow/data/lanflow.db "SELECT ip, SUM(tx_bytes+rx_bytes) as total FROM traffic_stats GROUP BY ip ORDER BY total DESC;"
```

### 支持 IPv6 吗？

当前版本仅监控 IPv4 流量。IPv6 支持计划在未来版本中添加。

### 如何持久化 IP 转发和 iptables 规则？

```bash
# 持久化 IP 转发
echo "net.ipv4.ip_forward=1" | sudo tee /etc/sysctl.d/99-lanflow.conf
sudo sysctl -p /etc/sysctl.d/99-lanflow.conf

# 持久化 iptables（Ubuntu）
sudo apt install iptables-persistent
sudo iptables -t nat -A POSTROUTING -o eth0 -j MASQUERADE
sudo netfilter-persistent save
```

## Tech Stack

| 组件 | 技术 |
|------|------|
| 语言 | Go 1.21+ |
| 抓包 | [gopacket](https://github.com/google/gopacket) + libpcap |
| 存储 | [modernc.org/sqlite](https://pkg.go.dev/modernc.org/sqlite)（纯 Go SQLite） |
| WebSocket | [gorilla/websocket](https://github.com/gorilla/websocket) |
| 图表 | [ECharts](https://echarts.apache.org/) |
| 日志 | Go 标准库 `log/slog` |
| 前端 | 原生 HTML/CSS/JS（无构建工具） |

## License

MIT
