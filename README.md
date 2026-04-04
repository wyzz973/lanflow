<p align="center">
  <h1 align="center">Lanflow</h1>
  <p align="center">
    <strong>LAN Traffic Monitor</strong> — 局域网流量监控工具
  </p>
  <p align="center">
    <a href="#features">Features</a> •
    <a href="#how-it-works">How it Works</a> •
    <a href="#step-by-step-deployment">Step-by-Step Deployment</a> •
    <a href="#web-dashboard">Dashboard</a> •
    <a href="#maintenance">Maintenance</a> •
    <a href="#api">API</a> •
    <a href="#faq">FAQ</a>
  </p>
</p>

---

## What is Lanflow?

Lanflow 是一个轻量级的局域网流量监控工具。它部署在局域网内**一台 Linux 机器**上，通过将该机器设为网关并抓包的方式，**实时统计每个 IP 的上行/下行流量**，并提供 Web 仪表盘进行可视化展示。

**典型场景：** 实验室/办公室多台电脑共享一个路由器上网，路由器没有 per-IP 流量统计功能，需要知道每个人用了多少流量。

### Why Lanflow?

- **零客户端** — 只需部署在一台机器上，局域网内其他设备无需安装任何软件
- **单一二进制** — Go 编译，前端资源嵌入，一个文件搞定部署
- **高性能** — 基于 libpcap 内核级抓包，Go 原生并发，轻松处理百兆网络
- **低资源** — 运行时内存 < 20MB，SQLite 存储，无外部依赖
- **高可用** — 崩溃自动重启、OOM 保护、watchdog 看门狗，即使 lanflow 挂了也不影响上网
- **Web 仪表盘** — 实时监控 + 历史统计 + 设备管理，开箱即用

## Features

- **实时流量监控** — WebSocket 推送，2 秒刷新，展示每个 IP 的瞬时上行/下行速率
- **域名级流量追踪** — 通过 TLS SNI 提取，精确到每个设备访问了哪些网站，流量各占多少
- **智能域名识别** — 内置 1426 个服务的 34,000+ 条域名规则（来自 [v2fly/domain-list-community](https://github.com/v2fly/domain-list-community)），自动将域名映射为友好名称（如 `bilibili.com` → `B站`）
- **历史统计查询** — 支持按天/周/月查看各设备流量，ECharts 图表可视化，排名表格
- **设备命名管理** — 自动发现局域网设备，点击即可给 IP 起别名
- **数据持久化** — SQLite 存储，默认保留 90 天，支持自定义
- **高可用** — 崩溃自动重启、OOM 保护、watchdog 看门狗
- **NAT 流量校正** — 自动识别并校正服务器自身因 NAT 导致的流量重复计算

---

## How it Works

### 原理简述

正常情况下，你的网络是这样的：

```
[外网/校园网] ←→ [路由器 192.168.1.1] ←→ [所有电脑]
```

部署 Lanflow 后，变成这样：

```
[外网/校园网] ←→ [路由器 192.168.1.1] ←→ [Lanflow 服务器] ←→ [所有电脑]
```

**所有电脑的外网流量都会经过 Lanflow 服务器**，Lanflow 在这台服务器上抓包统计，然后把流量转发给路由器。对用户来说完全透明，感觉不到任何区别。

### 你需要做什么

1. 在一台 **Linux 机器**上安装 Lanflow
2. 在**路由器**上改一个设置（把网关指向 Lanflow 机器）
3. 打开浏览器看流量统计

其他所有电脑（Windows/Mac/Linux）**不需要做任何事情**。

### 对你的网络有什么影响？

| 问题 | 答案 |
|------|------|
| 会变慢吗？ | 几乎不会。Lanflow 只是抓包统计，不修改数据包，转发由 Linux 内核完成，性能损耗极低 |
| Lanflow 挂了会断网吗？ | **不会。** IP 转发是 Linux 内核功能，即使 Lanflow 进程崩溃，网络转发照常工作。你只是暂时看不到统计数据 |
| 服务器关机了会断网吗？ | **会。** 因为所有流量都经过这台服务器，服务器关机 = 网关消失 = 断网。解决方法见 [紧急恢复](#emergency-recovery) |

---

## Step-by-Step Deployment

> 下面以 Ubuntu 为例，一步步教你部署。**整个过程大约 10 分钟。**

### 前提条件

- 一台 **Linux 服务器**（Ubuntu 18.04+/CentOS 7+/Debian 10+），需要一直开机
- 这台服务器和其他电脑在**同一个局域网**（连同一个路由器）
- 你有这台服务器的 **root 权限**（或 sudo 权限）
- 你能登录路由器管理界面

### Step 1: 查看你的网络信息

SSH 登录你的 Linux 服务器，运行以下命令：

```bash
# 查看网卡名称和 IP
ip addr show
```

你会看到类似这样的输出：

```
2: enp132s0: <BROADCAST,MULTICAST,UP,LOWER_UP>
    inet 192.168.1.108/24 brd 192.168.1.255 scope global enp132s0
```

记下以下信息（后面要用）：

| 信息 | 示例值 | 你的值 |
|------|--------|--------|
| 网卡名称 | `enp132s0` | ________ |
| 服务器 IP | `192.168.1.108` | ________ |
| 局域网网段 | `192.168.1.0/24` | ________ |
| 路由器 IP | `192.168.1.1` | ________ |

> **提示：** 路由器 IP 通常是 `192.168.1.1` 或 `192.168.0.1`，可以用 `ip route | grep default` 查看。

### Step 2: 安装依赖

```bash
# Ubuntu / Debian
sudo apt update
sudo apt install -y git libpcap-dev iptables-persistent

# CentOS / RHEL
sudo yum install -y git libpcap-devel iptables-services
```

### Step 3: 安装 Go（如果还没装）

```bash
# 下载 Go（中国用户可以用镜像加速）
wget https://go.dev/dl/go1.24.2.linux-amd64.tar.gz
# 或者用镜像：wget https://golang.google.cn/dl/go1.24.2.linux-amd64.tar.gz

# 解压安装
sudo tar -C /usr/local -xzf go1.24.2.linux-amd64.tar.gz

# 添加到 PATH（写入 .bashrc 保证重启后仍然有效）
echo 'export PATH=$PATH:/usr/local/go/bin' >> ~/.bashrc
source ~/.bashrc

# 验证
go version
# 应该输出：go version go1.24.2 linux/amd64
```

### Step 4: 下载并编译 Lanflow

```bash
# 克隆代码
git clone https://github.com/wyzz973/lanflow.git
cd lanflow

# 设置 Go 代理（中国用户加速下载依赖）
export GOPROXY=https://goproxy.cn,direct

# 编译
go build -o lanflow ./cmd/lanflow/

# 验证编译成功
ls -lh lanflow
# 应该看到一个约 17MB 的文件

# 运行测试（可选，确认一切正常）
go test ./... -v
```

### Step 5: 配置 Lanflow

```bash
# 编辑配置文件，把网卡名和 IP 改成你自己的
vim config.yaml
```

需要修改的内容（把示例值改成你在 Step 1 记下的值）：

```yaml
interface: "enp132s0"          # ← 改成你的网卡名称
lan_cidr: "192.168.1.0/24"     # ← 改成你的局域网网段
gateway_ip: "192.168.1.1"      # ← 改成你的路由器 IP

db_path: "./data/lanflow.db"   # 数据库路径，一般不用改
retention_days: 90             # 数据保留天数
listen: ":18973"                # Web 界面端口
log_level: "info"              # 日志级别
log_dir: "./logs"              # 日志目录
```

### Step 6: 部署为系统服务

```bash
# 创建部署目录
sudo mkdir -p /opt/lanflow/{data,logs}

# 复制文件
sudo cp lanflow config.yaml /opt/lanflow/

# 安装 systemd 服务
sudo cp lanflow.service /etc/systemd/system/

# 设置开机自启 + 启动服务
sudo systemctl daemon-reload
sudo systemctl enable lanflow
sudo systemctl start lanflow

# 检查是否启动成功
sudo systemctl status lanflow
```

你应该看到 `Active: active (running)`，表示服务已经在运行了。

### Step 7: 设置网络转发

```bash
# 开启 IP 转发（让这台机器可以转发其他电脑的流量）
echo "net.ipv4.ip_forward=1" | sudo tee /etc/sysctl.d/99-lanflow.conf
sudo sysctl -p /etc/sysctl.d/99-lanflow.conf

# 设置 NAT 转发规则（把 enp132s0 换成你的网卡名）
sudo iptables -t nat -A POSTROUTING -s 192.168.1.0/24 -o enp132s0 -j MASQUERADE

# 持久化 iptables 规则（重启后仍然有效）
sudo netfilter-persistent save
```

### Step 8: 安装 Watchdog（看门狗）

Watchdog 每分钟检查一次，确保 IP 转发、iptables 规则、lanflow 服务都正常运行。如果有异常会自动修复。

创建文件 `/opt/lanflow/watchdog.sh`：

```bash
sudo tee /opt/lanflow/watchdog.sh << 'EOF'
#!/bin/bash
LOG="/opt/lanflow/logs/watchdog.log"
TIMESTAMP=$(date '+%Y-%m-%d %H:%M:%S')

# 确保 IP 转发开启
if [ "$(cat /proc/sys/net/ipv4/ip_forward)" != "1" ]; then
    echo "$TIMESTAMP [WARN] ip_forward was off, re-enabling" >> "$LOG"
    sysctl -w net.ipv4.ip_forward=1 > /dev/null
fi

# 确保 NAT 规则存在（把网段和网卡名换成你自己的）
if ! iptables -t nat -C POSTROUTING -s 192.168.1.0/24 -o enp132s0 -j MASQUERADE 2>/dev/null; then
    echo "$TIMESTAMP [WARN] MASQUERADE rule missing, re-adding" >> "$LOG"
    iptables -t nat -A POSTROUTING -s 192.168.1.0/24 -o enp132s0 -j MASQUERADE
fi

# 确保 lanflow 服务在运行
if ! systemctl is-active --quiet lanflow; then
    echo "$TIMESTAMP [ERROR] lanflow is down, restarting" >> "$LOG"
    systemctl restart lanflow
fi

# 检查能否连通路由器
if ! ping -c 1 -W 2 192.168.1.1 > /dev/null 2>&1; then
    echo "$TIMESTAMP [ERROR] cannot reach router 192.168.1.1" >> "$LOG"
fi
EOF

# 设置可执行权限
sudo chmod +x /opt/lanflow/watchdog.sh

# 添加到 cron，每分钟执行一次
echo "* * * * * root /opt/lanflow/watchdog.sh" | sudo tee /etc/cron.d/lanflow-watchdog
```

### Step 9: 修改路由器设置

**这是最关键的一步。** 修改后，所有电脑的外网流量就会经过你的 Lanflow 服务器。

1. 打开浏览器，访问路由器管理界面（通常是 `http://192.168.1.1`）
2. 找到 **DHCP 服务器** 设置（一般在"路由设置"或"LAN 设置"里）
3. 把 **网关** 从 `0.0.0.0`（或空）改成 **Lanflow 服务器的 IP**（如 `192.168.1.108`）
4. 点 **保存**

```
修改前：网关 = 0.0.0.0（或 192.168.1.1）
修改后：网关 = 192.168.1.108  ← 你的 Lanflow 服务器 IP
```

> **DNS 设置不用改**，保持原样即可。

### Step 10: 让其他电脑生效

修改路由器后，其他电脑需要重新获取 IP 才能生效。有两种方式：

**方式一：等待自动生效（推荐）**

路由器 DHCP 租期到了会自动更新，一般 2 小时内全部生效。

**方式二：手动立即生效**

| 操作系统 | 命令 |
|---------|------|
| **Windows** | 打开 CMD，运行 `ipconfig /release && ipconfig /renew` |
| **Linux** | 运行 `sudo dhclient -r && sudo dhclient` 或者 `sudo nmcli connection down "有线连接" && sudo nmcli connection up "有线连接"` |
| **macOS** | 系统设置 → 网络 → 点击已连接的网络 → 关闭再打开 |

### Step 11: 验证

1. 打开浏览器访问 `http://192.168.1.108:18973`（把 IP 换成你的 Lanflow 服务器 IP）
2. 你应该能看到 Lanflow 的仪表盘
3. 在"实时监控"标签页，应该能看到各个设备的流量数据
4. 在"设备管理"标签页，给每个 IP 起个名字方便识别

**恭喜！部署完成！** 🎉

---

## Web Dashboard

访问 `http://<服务器IP>:18973`，包含三个标签页：

### 实时监控

- 顶部 3 个统计卡片：在线设备数、总上行、总下行
- 设备表格：名称、IP、实时速率、今日累计流量
- **"详情"按钮**：弹出该设备的域名流量明细（访问了哪些网站，各占多少流量）
- WebSocket 自动推送，每 2 秒刷新
- 按总流量降序排列

### 历史统计

- 选择时间范围（天/周/月）和日期进行查询
- 柱状图对比各设备流量
- **流量排行表格**：前 3 名金银铜高亮
- 点击某设备展开详细趋势图 + **域名流量明细**

### 设备管理

- 自动列出所有已发现的 IP，点击即可命名
- 绿色标签 = 已命名，蓝色标签 = 未命名
- 给 IP 设置名称和备注，方便识别

---

## Maintenance

### 日常运维

```bash
# 查看服务状态
sudo systemctl status lanflow

# 查看实时日志
sudo journalctl -u lanflow -f

# 查看 watchdog 日志
tail -f /opt/lanflow/logs/watchdog.log

# 手动重启服务（不影响网络转发）
sudo systemctl restart lanflow
```

### 更新 Lanflow

```bash
# 1. 在编译机器上拉取新代码并编译
cd lanflow
git pull
go build -o lanflow ./cmd/lanflow/

# 2. 传到服务器
scp lanflow user@192.168.1.108:/home/user/

# 3. 在服务器上替换（需要先停再换）
sudo systemctl stop lanflow
sudo cp /home/user/lanflow /opt/lanflow/lanflow
sudo systemctl start lanflow
```

### <a id="emergency-recovery"></a>紧急恢复（服务器宕机导致全部断网）

如果 Lanflow 服务器突然宕机（断电、硬件故障等），所有电脑会断网。按以下步骤恢复：

**方法一：重启服务器（推荐）**

重新开机即可，lanflow 会自动启动，网络自动恢复。

**方法二：临时绕过 Lanflow（服务器无法启动时）**

1. 打开路由器管理界面 `http://192.168.1.1`（用手机连路由器 WiFi 访问）
2. 进入 DHCP 服务器设置
3. 把**网关**从 Lanflow 服务器 IP 改回 `0.0.0.0`
4. 保存
5. 各电脑重新连接网络（或等 DHCP 续租）

> 改回后所有电脑直接通过路由器上网，不再经过 Lanflow，流量不再被监控。等服务器修好后再改回来。

### 迁移到新机器

如果要把 Lanflow 从旧服务器迁移到新服务器：

#### 1. 在新服务器上安装

按照 [Step-by-Step Deployment](#step-by-step-deployment) 的 Step 1 ~ Step 8 在新服务器上完成安装。

**注意修改配置文件中的网卡名称**（新服务器的网卡名可能不同）。

#### 2. 迁移历史数据（可选）

如果你想保留历史流量数据：

```bash
# 在旧服务器上
sudo systemctl stop lanflow
scp /opt/lanflow/data/lanflow.db user@新服务器IP:/tmp/

# 在新服务器上
sudo systemctl stop lanflow
sudo cp /tmp/lanflow.db /opt/lanflow/data/lanflow.db
sudo systemctl start lanflow
```

#### 3. 切换路由器网关

1. 打开路由器管理界面
2. 把 DHCP 网关从旧服务器 IP 改成**新服务器 IP**
3. 保存

#### 4. 关闭旧服务器的 lanflow

```bash
# 在旧服务器上
sudo systemctl stop lanflow
sudo systemctl disable lanflow
```

#### 5. 验证

等各电脑 DHCP 续租后（或手动刷新），检查 `http://新服务器IP:18973` 能否正常显示流量数据。

### 完全卸载

如果不想再用 Lanflow 了：

```bash
# 1. 先改回路由器网关！（否则断网）
#    路由器 DHCP 网关改回 0.0.0.0

# 2. 停止并删除服务
sudo systemctl stop lanflow
sudo systemctl disable lanflow
sudo rm /etc/systemd/system/lanflow.service
sudo systemctl daemon-reload

# 3. 删除 watchdog
sudo rm /etc/cron.d/lanflow-watchdog

# 4. 删除 iptables 规则
sudo iptables -t nat -D POSTROUTING -s 192.168.1.0/24 -o enp132s0 -j MASQUERADE
sudo netfilter-persistent save

# 5. 关闭 IP 转发（如果不需要的话）
sudo rm /etc/sysctl.d/99-lanflow.conf
sudo sysctl -p

# 6. 删除文件
sudo rm -rf /opt/lanflow
```

---

## High Availability

Lanflow 提供了多层高可用保障：

| 保障层级 | 措施 | 说明 |
|---------|------|------|
| **进程级** | systemd `Restart=always` | lanflow 崩溃后 3 秒内自动重启 |
| **进程级** | `OOMScoreAdjust=-900` | 内存不足时，内核优先杀其他进程，保住网关 |
| **网络级** | IP 转发独立于 lanflow | 即使 lanflow 挂了，Linux 内核继续转发流量，**不会断网** |
| **网络级** | iptables 持久化 | 重启后 NAT 规则自动恢复 |
| **系统级** | sysctl 持久化 | 重启后 IP 转发自动开启 |
| **监控级** | watchdog 脚本 | 每分钟检查：IP 转发、iptables、lanflow 服务，异常自动修复 |

**关键设计：** 网络转发（IP forward + iptables）是内核级功能，和 lanflow 进程完全解耦。lanflow 只负责"看"流量，不负责"转"流量。所以 lanflow 挂了 ≠ 断网。

---

## Configuration

`config.yaml` 配置说明：

```yaml
# 网络配置（必填）
interface: "eth0"            # 抓包网卡名称，用 ip addr 查看
lan_cidr: "192.168.1.0/24"   # 局域网网段
gateway_ip: "192.168.1.1"    # 路由器 IP

# 存储配置
db_path: "./data/lanflow.db" # SQLite 数据库路径
retention_days: 90           # 数据保留天数（超过自动清理）

# Web 服务
listen: ":18973"              # 监听端口，访问 http://服务器IP:18973

# 日志
log_level: "info"            # 日志级别：debug / info / warn / error
log_dir: "./logs"            # 日志文件目录
```

命令行参数可覆盖配置文件：

```bash
sudo ./lanflow --config /opt/lanflow/config.yaml --log-level debug
```

---

## API

所有 API 返回 JSON 格式 `{"data": ...}`。

### REST API

| 方法 | 端点 | 说明 |
|------|------|------|
| GET | `/api/realtime` | 所有 IP 的实时流量快照 |
| GET | `/api/stats?range=day&date=2026-04-04` | 按天/周/月查询汇总 |
| GET | `/api/stats/{ip}?range=day&date=2026-04-04` | 单个 IP 的分钟级详细记录 |
| GET | `/api/domains/{ip}?range=day&date=2026-04-04` | 某个 IP 的域名流量明细 |
| GET | `/api/domains?range=day&date=2026-04-04` | 所有 IP 的域名流量汇总 |
| GET | `/api/devices` | 获取设备列表 |
| PUT | `/api/devices/{ip}` | 设置设备名称，Body: `{"name":"GPU-01","note":"301室"}` |

### WebSocket

```
WebSocket /ws/realtime
```

每 2 秒推送一次所有 IP 的实时流量数据。

### 示例

```bash
# 获取实时流量
curl http://192.168.1.108:18973/api/realtime

# 查询今天的流量统计
curl "http://192.168.1.108:18973/api/stats?range=day&date=$(date +%Y-%m-%d)"

# 给某个 IP 起名字
curl -X PUT http://192.168.1.108:18973/api/devices/192.168.1.100 \
  -H "Content-Type: application/json" \
  -d '{"name": "张三-办公机", "note": "301实验室"}'
```

---

## Project Structure

```
lanflow/
├── cmd/lanflow/main.go          # 程序入口
├── internal/
│   ├── aggregator/              # 流量聚合：per-IP + per-domain 计数器
│   ├── api/                     # HTTP 服务：REST + WebSocket + 前端
│   │   └── static/              # 前端资源（嵌入到二进制中）
│   ├── capture/                 # libpcap 抓包 + SNI/Host 提取
│   ├── classifier/              # 域名分类（v2fly 规则，34,000+ 条）
│   │   └── rules.txt            # 嵌入的域名规则文件
│   ├── config/                  # YAML 配置加载
│   ├── logger/                  # 结构化日志
│   └── storage/                 # SQLite 存储
├── config.yaml                  # 默认配置文件
├── lanflow.service              # systemd 服务文件
├── go.mod
└── go.sum
```

### 域名规则更新

域名分类规则来自 [v2fly/domain-list-community](https://github.com/v2fly/domain-list-community)，编译时嵌入二进制。如需更新：

```bash
# 拉取最新规则
cd /tmp/domain-list-community && git pull

# 重新生成 rules.txt
cd /path/to/lanflow && go run /tmp/gen_rules.go

# 重新编译部署
go build -o lanflow ./cmd/lanflow/
```

---

## FAQ

### Lanflow 挂了会断网吗？

**不会。** IP 转发和 NAT 是 Linux 内核功能，和 lanflow 进程无关。lanflow 只负责"统计"，不负责"转发"。lanflow 挂了只是暂时看不到流量数据，3 秒内会自动重启。

### 整台服务器宕机了怎么办？

这时候会断网。两个恢复方法：
1. **快速恢复：** 重启服务器，lanflow 自动启动，网络自动恢复
2. **临时绕过：** 用手机连路由器 WiFi，进路由器管理界面把网关改回 `0.0.0.0`

### 代理/VPN 流量能统计到吗？

能。无论用户是否使用 Clash、V2Ray 等代理，流量都会经过网关，**总流量统计是准确的**。但无法区分哪些是直连流量、哪些是代理流量。使用代理时，域名明细只能看到代理服务器的地址，看不到实际访问的网站。

### 域名识别不全？

域名识别依赖 TLS SNI（HTTPS 握手时的明文字段）。以下情况可能识别不到：
- 使用 ESNI/ECH 加密的网站（目前极少）
- 非 HTTPS 非 HTTP 的协议（如 SSH、游戏等）
- 使用代理/VPN 后的实际目标域名

未识别的流量会显示原始域名或 IP 地址。

### 如何清理流量数据重新统计？

```bash
# 停止服务
sudo systemctl stop lanflow

# 只清流量数据，保留设备命名
sqlite3 /opt/lanflow/data/lanflow.db "DELETE FROM traffic_stats; DELETE FROM domain_stats; VACUUM;"

# 重启
sudo systemctl start lanflow
```

### 如何让设备立即切换网关？

| 系统 | 操作 |
|------|------|
| **Windows** | CMD 运行 `ipconfig /release && ipconfig /renew` |
| **Linux** | 运行 `sudo dhclient -r && sudo dhclient` |
| **macOS** | 系统设置 → 网络 → 断开再连接 |

### 数据库文件在哪？

使用 systemd 部署时在 `/opt/lanflow/data/lanflow.db`。可以直接用 SQLite 工具查看：

```bash
sqlite3 /opt/lanflow/data/lanflow.db \
  "SELECT ip, SUM(tx_bytes+rx_bytes) as total FROM traffic_stats GROUP BY ip ORDER BY total DESC;"
```

### 如何修改 Web 端口？

编辑 `/opt/lanflow/config.yaml`，把 `listen: ":18973"` 改成你想要的端口（如 `:80`），然后 `sudo systemctl restart lanflow`。

### 支持 IPv6 吗？

当前版本仅监控 IPv4 流量。

---

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
