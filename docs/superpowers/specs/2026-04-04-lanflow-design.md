# Lanflow — 局域网流量监控工具设计文档

## 背景

实验室所有电脑通过一台普通路由器共享校园网账号。路由器无法安装软件，也没有 per-IP 流量统计功能。需要一个工具部署在局域网内一台 Linux 机器上，作为透明网关，抓包统计每个 IP 的外网流量。

## 部署模型

- **仅部署在一台 Linux 机器上**（如 GPU 服务器）
- 将该机器设为局域网网关（修改路由器 DHCP 的默认网关指向此机器）
- 该机器开启 IP 转发，将流量转发给真正的路由器
- 所有外网流量经过此机器，抓包即可统计全网 IP 流量
- 其他机器（Windows/Linux/Mac）无需安装任何东西

## 技术栈

- **语言**：Go
- **抓包**：gopacket + libpcap
- **存储**：SQLite
- **Web 前端**：原生 HTML/JS + ECharts（CDN），通过 Go embed 嵌入二进制
- **日志**：Go 标准库 log/slog

## 架构

```
┌─────────────────────────────────────────────────┐
│                 lanflow (单一 Go 二进制)           │
│                                                   │
│  ┌───────────┐   ┌───────────┐   ┌────────────┐  │
│  │ Capture   │──>│ Aggregator│──>│  SQLite    │  │
│  │ (libpcap) │   │ (内存统计) │   │  (持久化)  │  │
│  └───────────┘   └─────┬─────┘   └─────┬──────┘  │
│                        │               │          │
│                  ┌─────▼───────────────▼──────┐   │
│                  │     HTTP Server            │   │
│                  │  - REST API (查询统计)      │   │
│                  │  - WebSocket (实时推送)     │   │
│                  │  - 静态文件 (前端嵌入)      │   │
│                  └────────────────────────────┘   │
└─────────────────────────────────────────────────┘
```

### 模块说明

1. **Capture** — 用 gopacket/libpcap 抓包，解析 IP 层，判断上行/下行方向
2. **Aggregator** — 内存中维护 `map[IP]*Counter`，定期（每 60 秒）flush 到 SQLite
3. **Storage** — SQLite 读写，数据清理
4. **HTTP Server** — REST API + WebSocket + 嵌入的前端静态文件

### 流量方向判断

- 配置局域网网段（如 `192.168.1.0/24`）
- 源 IP 在局域网内 → 目的 IP 在外 = 该源 IP 的**上行**
- 目的 IP 在局域网内 → 源 IP 在外 = 该目的 IP 的**下行**
- 局域网内互相通信的流量不统计（非外网流量）

## 数据模型

```sql
-- 设备信息
CREATE TABLE devices (
    ip    TEXT PRIMARY KEY,
    name  TEXT NOT NULL,
    note  TEXT DEFAULT ''
);

-- 每分钟聚合的流量记录
CREATE TABLE traffic_stats (
    id         INTEGER PRIMARY KEY AUTOINCREMENT,
    ip         TEXT    NOT NULL,
    timestamp  INTEGER NOT NULL,  -- Unix 时间戳，按分钟对齐
    tx_bytes   INTEGER NOT NULL,  -- 上行字节数
    rx_bytes   INTEGER NOT NULL,  -- 下行字节数
    tx_packets INTEGER NOT NULL,  -- 上行包数
    rx_packets INTEGER NOT NULL   -- 下行包数
);

CREATE INDEX idx_ip_time ON traffic_stats(ip, timestamp);
CREATE INDEX idx_time ON traffic_stats(timestamp);
```

### 存储策略

- 内存中维护 `map[IP]*Counter`，每个 Counter 记录当前分钟的字节数/包数
- 每 60 秒 flush 到 SQLite，然后重置内存计数器
- 数据保留 90 天（可配置），定期清理旧数据
- 90 天数据量估算：20 IP × 1440 分钟/天 × 90 天 ≈ 260 万行，SQLite 轻松应对

## API 设计

| 方法 | 端点 | 说明 |
|------|------|------|
| GET | `/api/realtime` | 当前所有 IP 的实时速率（bytes/s） |
| GET | `/api/stats?range=day&date=2026-04-04` | 按天/周/月查询流量统计 |
| GET | `/api/stats/{ip}?range=week` | 单个 IP 的详细流量历史 |
| GET | `/api/devices` | 获取设备列表 |
| PUT | `/api/devices/{ip}` | 设置/修改设备名称和备注 |
| WebSocket | `/ws/realtime` | 实时推送（每 2 秒），推送所有 IP 的瞬时速率 |

## 前端设计

三个标签页：

1. **实时监控** — 所有设备表格：名称、IP、当前上行/下行速率、今日累计流量。WebSocket 自动刷新，按总流量排序
2. **历史统计** — 选择时间范围（天/周/月），柱状图展示各设备流量对比；点击某设备查看时间趋势折线图
3. **设备管理** — 设备列表，给 IP 起名、加备注；显示首次/最近出现时间

技术方案：
- 原生 HTML/CSS/JS，无构建工具
- ECharts（CDN 引入）做图表
- 简洁的 CSS，响应式布局
- 通过 Go 的 `embed` 嵌入到二进制中

## 日志

- 使用 Go 标准库 `log/slog`（结构化日志）
- 输出到 stdout + 文件（`logs/lanflow.log`）
- 日志文件按天轮转，保留 30 天

| 级别 | 内容 |
|------|------|
| INFO | 启动/停止、配置加载、抓包接口绑定、flush 统计摘要 |
| WARN | 新发现未命名 IP、丢包率过高、磁盘空间不足 |
| ERROR | 抓包失败、数据库写入失败、网卡异常 |
| DEBUG | 每次 flush 的详细数据、API 请求日志 |

配置：启动参数 `--log-level=info` 控制日志级别，默认 INFO。

## 配置

`config.yaml`：

```yaml
interface: "eth0"           # 抓包网卡
lan_cidr: "192.168.1.0/24"  # 局域网网段
gateway_ip: "192.168.1.1"   # 路由器 IP

db_path: "./data/lanflow.db"
retention_days: 90

listen: ":8080"

log_level: "info"
log_dir: "./logs"
```

## 部署

```bash
# 1. 编译
go build -o lanflow ./cmd/lanflow

# 2. 传到服务器
scp lanflow config.yaml user@server:~/lanflow/

# 3. 开启 IP 转发
sudo sysctl -w net.ipv4.ip_forward=1

# 4. 设置 iptables 转发
sudo iptables -t nat -A POSTROUTING -o eth0 -j MASQUERADE

# 5. 运行
sudo ./lanflow --config config.yaml
```

提供 systemd service 文件，支持开机自启和崩溃自动重启。

## 项目结构

```
lanflow/
├── cmd/lanflow/main.go        # 入口
├── internal/
│   ├── capture/                # libpcap 抓包
│   ├── aggregator/             # 流量聚合
│   ├── storage/                # SQLite 读写
│   ├── api/                    # HTTP + WebSocket
│   ├── config/                 # 配置加载
│   └── logger/                 # 日志初始化
├── web/                        # 前端静态文件（embed）
│   ├── index.html
│   ├── js/
│   └── css/
├── config.yaml
├── lanflow.service             # systemd 配置
├── go.mod
└── go.sum
```
