# Moltage — 微信公众号 × AI 智能体沙箱平台

Moltage 是一个完整的微信公众号 AI 沙箱解决方案。用户通过微信公众号与 AI 智能体对话，每个用户自动分配独立的 Firecracker 微虚拟机环境，并可通过 SSH 直接访问。

> [!IMPORTANT]
> **使用门槛声明**
> 本系统仅支持 **非个人认证的公众号**（如服务号、企业认证订阅号）。由于微信接口权限限制，个人认证的公众号无法使用客服消息接口进行回复。

> [!TIP]
> **商业合作**
> 如果您需要商用授权、技术支持或定制开发，请联系：**nomorelighthouse@gmail.com**

---

## 目录

- [架构概览](#架构概览)
- [系统要求](#系统要求)
- [安装手册](#安装手册)
  - [快速部署 (Docker Compose)](#快速部署-docker-compose)
  - [分步手动安装](#分步手动安装)
  - [Windows WSL 一键安装](#windows-wsl-一键安装)
- [使用手册](#使用手册)
  - [微信后台配置](#微信后台配置)
  - [用户使用指南](#用户使用指南)
  - [管理员运维指南](#管理员运维指南)
  - [OpenClaw 插件配置](#openclaw-插件配置)
- [环境变量参考](#环境变量参考)
- [常见问题](#常见问题)
- [许可证](#许可证)

---

## 架构概览

```
┌──────────────────────────────────────────────────────────────┐
│                      微信用户 (手机端)                         │
└───────────────┬──────────────────────────────────────────────┘
                │ HTTPS
                ▼
┌──────────────────────────────────────────────────────────────┐
│                    微信公众平台服务器                          │
└───────────────┬──────────────────────────────────────────────┘
                │ HTTP POST
                ▼
┌───────────────────────────────┐     ┌────────────────────────┐
│    WeChat Bridge (Node.js)    │◄───►│   Redis (会话 & 绑定)   │
│    端口: 3000                  │     └────────────────────────┘
│    • 接收微信消息               │                │
│    • 转发至 OpenClaw 插件       │                │
│    • 调用 Orchestrator 分配 VM  │                │
└───────────┬───────────────────┘                │
            │                                    │
            ▼                                    │
┌───────────────────────────────┐                │
│  VM Orchestrator (Go/Gin)     │◄───────────────┘
│  端口: 8080                    │
│  • 管理 Firecracker 微虚拟机    │
│  • IP 池分配                   │
│  • Rootfs 生命周期管理          │
└───────────┬───────────────────┘
            │
            ▼
┌───────────────────────────────┐
│  Firecracker MicroVMs         │
│  (每用户 1 vCPU / 512MB)      │
│  • 运行 OpenClaw + 插件        │
│  • 独立网络命名空间             │
└───────────────────────────────┘
            ▲
            │ SSH (端口 2222)
┌───────────┴───────────────────┐
│  SSH Gateway (Go)             │
│  • 根据 Redis 绑定查找用户 VM   │
│  • 自动密码认证代理             │
└───────────────────────────────┘
```

### 组件列表

| 组件 | 目录 | 语言 | 说明 |
|------|------|------|------|
| **WeChat Bridge** | `clawdbot-wechat-bridge/` | TypeScript / Node.js | 微信消息收发、用户绑定、VM 调度 |
| **Webhook Plugin** | `clawdbot-plugin-webhook-server/` | TypeScript | OpenClaw 插件，接收 Bridge 转发的消息 |
| **VM Orchestrator** | `vm-orchestrator/` | Go (Gin) | Firecracker VM 生命周期管理 API |
| **SSH Gateway** | `ssh-gateway/` | Go | SSH 代理网关，路由用户到对应 VM |
| **Rootfs Builder** | `rootfs/` | Shell | 构建 Alpine Linux 根文件系统镜像 |

---

## 系统要求

### 生产环境（完整部署）

| 项目 | 要求 |
|------|------|
| **操作系统** | Linux (Ubuntu 22.04+ 推荐) |
| **CPU** | 支持 KVM 虚拟化 (AMD-V / Intel VT-x) |
| **内存** | ≥ 4GB (每个 VM 512MB) |
| **磁盘** | ≥ 20GB SSD |
| **Docker** | Docker Engine 24+ / Docker Compose v2 |
| **网络** | 一个公网 IP 或域名 (用于微信回调) |

### 开发/测试环境

| 项目 | 要求 |
|------|------|
| **操作系统** | Linux / macOS / Windows (WSL2) |
| **Docker** | Docker Desktop 或 Docker Engine |
| **Node.js** | 20+ (如需本地开发 Bridge) |
| **Go** | 1.22+ (如需本地开发 Gateway/Orchestrator) |

> [!NOTE]
> Firecracker 微虚拟机功能 **仅在 Linux 裸机** 且开启 KVM 的环境下可用。Docker Desktop (macOS/Windows) 不支持嵌套虚拟化，因此 orchestrator 服务只有 API 层可用，不能实际创建 VM。

---

## 安装手册

### 快速部署 (Docker Compose)

这是推荐的部署方式，适用于拥有公网 Linux 服务器的场景。

#### 1. 克隆项目

```bash
git clone https://github.com/your-org/moltage.git
cd moltage
```

#### 2. 创建环境变量文件

在 `clawdbot-wechat-bridge/` 目录下创建 `.env` 文件：

```bash
cd clawdbot-wechat-bridge
cp .env.example .env   # 如果有 .env.example
```

编辑 `.env` 文件，填入您的配置：

```dotenv
# ===== 微信公众号配置 (必填) =====
WECHAT_APPID=wx1234567890abcdef
WECHAT_APPSECRET=your_wechat_appsecret
WECHAT_TOKEN=your_custom_token
WECHAT_ENCODING_AES_KEY=your_encoding_aes_key

# ===== Bridge 公网地址 (必填) =====
# 微信服务器需要能访问此地址
BRIDGE_BASE_URL=http://your-server-ip:3000

# ===== SSH 配置 =====
SSH_HOST=your-server-ip
SSH_PORT=2222

# ===== VM 配置 (可选, 有默认值) =====
VM_SUBNET=10.0.1.0/24
GATEWAY_IP=10.0.1.1
DEFAULT_VCPU=1
DEFAULT_MEM_MIB=512

# ===== 其他 (可选) =====
LOG_LEVEL=info
VM_READY_TIMEOUT_MS=30000
```

#### 3. 构建并启动所有服务

```bash
# 在 clawdbot-wechat-bridge/ 目录下（docker-compose.yml 所在目录）
docker compose build
docker compose up -d
```

#### 4. 验证服务状态

```bash
# 查看所有服务运行状态
docker compose ps

# 预期输出 — 4 个服务均为 running:
# clawdbot-wechat-bridge   running   0.0.0.0:3000->3000/tcp
# vm-orchestrator           running
# ssh-gateway               running   0.0.0.0:2222->2222/tcp
# clawdbot-wechat-redis     running

# 查看 Bridge 日志
docker compose logs -f bridge

# 查看所有服务日志
docker compose logs -f
```

#### 5. 配置微信后台

参见下方 [微信后台配置](#微信后台配置) 章节。

---

### 分步手动安装

如果您需要单独部署某些组件，或需要在物理机上运行。

#### 安装 WeChat Bridge

```bash
cd clawdbot-wechat-bridge

# 安装依赖
npm install

# 编译 TypeScript
npm run build

# 设置环境变量
export WECHAT_APPID=wx1234567890abcdef
export WECHAT_APPSECRET=your_appsecret
export WECHAT_TOKEN=your_token
export BRIDGE_BASE_URL=http://your-server:3000
export REDIS_URL=redis://localhost:6379
export ORCHESTRATOR_URL=http://localhost:8080

# 启动
node dist/index.js
```

#### 安装 VM Orchestrator

```bash
cd vm-orchestrator

# 编译
go mod tidy
CGO_ENABLED=0 go build -o orchestrator ./cmd/orchestrator

# 设置环境变量
export REDIS_URL=redis://localhost:6379
export FIRECRACKER_BIN=/usr/local/bin/firecracker
export KERNEL_PATH=/var/lib/firecracker/vmlinux
export BASE_ROOTFS_PATH=/var/lib/firecracker/rootfs/clawdbot.ext4

# 启动
./orchestrator
```

> [!WARNING]
> Orchestrator 需要 `firecracker` 二进制文件和 Linux 内核镜像 (`vmlinux`) 才能创建 VM。若未安装，服务仍可启动但会输出警告，创建 VM 的请求将失败。
> 安装 Firecracker:
> ```bash
> ARCH=$(uname -m)
> curl -L https://github.com/firecracker-microvm/firecracker/releases/download/v1.6.0/firecracker-v1.6.0-${ARCH}.tgz | tar xz
> sudo mv release-v1.6.0-${ARCH}/firecracker-v1.6.0-${ARCH} /usr/local/bin/firecracker
> ```

#### 安装 SSH Gateway

```bash
cd ssh-gateway

# 编译
go mod tidy
CGO_ENABLED=0 go build -o ssh-gateway ./cmd/gateway

# 设置环境变量
export REDIS_URL=redis://localhost:6379
export LISTEN_ADDR=0.0.0.0:2222

# 启动
./ssh-gateway
```

#### 构建 Rootfs 镜像

```bash
cd rootfs

# 需要 root 权限 (使用 losetup、mount 等)
sudo bash build-rootfs.sh
```

构建完成后会生成 `clawdbot.ext4` 文件，将其复制到 Orchestrator 配置的路径：

```bash
sudo cp output/clawdbot.ext4 /var/lib/firecracker/rootfs/
```

---

## 使用手册

### 微信后台配置

#### 1. 登录微信公众平台

访问 [mp.weixin.qq.com](https://mp.weixin.qq.com)，使用管理员账号登录。

#### 2. 配置服务器

导航到 **设置与开发** → **基本配置** → **服务器配置**：

| 配置项 | 值 | 说明 |
|--------|-----|------|
| **URL** | `http://您的IP:3000/wechat` | Bridge 服务的微信回调地址 |
| **Token** | 与 `.env` 中的 `WECHAT_TOKEN` 一致 | 签名验证令牌 |
| **EncodingAESKey** | 与 `.env` 中的 `WECHAT_ENCODING_AES_KEY` 一致 | 消息加密密钥 |
| **消息加解密方式** | 安全模式 (推荐) | 也可选择明文或兼容模式 |

点击 **提交** 后，微信服务器会向您的 URL 发送 GET 验证请求。如果 Bridge 运行正常，验证会自动通过。

#### 3. 启用服务器配置

验证通过后，点击 **启用** 按钮。

> [!CAUTION]
> 启用服务器配置后，微信公众号的自动回复、自定义菜单等功能将由您的服务器接管。请确保服务稳定后再启用。

---

### 用户使用指南

#### 关注公众号并对话

1. 扫描公众号二维码或搜索公众号名称，点击 **关注**
2. 直接发送文字消息，即可与 AI 智能体对话
3. 系统会自动为您分配独立的运行环境

#### 绑定 OpenClaw 实例（高级）

如果您希望将公众号连接到自己的 OpenClaw 实例：

```
bind http://你的服务器IP:8789/webhook 你的AuthToken
```

**绑定成功后**，所有对话将转发到您的 OpenClaw 实例处理。

**解除绑定：**

```
unbind
```

#### SSH 远程访问

每个用户的 VM 都可以通过 SSH 访问：

```bash
# 使用分配的用户名连接
ssh -p 2222 你的微信OpenID@your-server-ip

# 默认密码: clawdbot
```

> [!NOTE]
> SSH Gateway 会自动根据您的微信 OpenID 查找对应的 VM，并代理 SSH 连接。

---

### 管理员运维指南

#### 查看服务状态

```bash
# 查看所有服务
docker compose ps

# 查看特定服务日志
docker compose logs -f bridge        # Bridge 日志
docker compose logs -f orchestrator  # Orchestrator 日志
docker compose logs -f ssh-gateway   # SSH Gateway 日志
docker compose logs -f redis         # Redis 日志
```

#### 重启服务

```bash
# 重启所有服务
docker compose restart

# 重启单个服务
docker compose restart bridge
docker compose restart orchestrator
```

#### 更新部署

```bash
# 拉取最新代码
git pull

# 重新构建并启动
docker compose build
docker compose up -d
```

#### 清理资源

```bash
# 停止所有服务
docker compose down

# 停止并删除所有数据 (包括 Redis 数据和 VM 存储)
docker compose down -v

# 清理构建缓存
docker builder prune
```

#### Redis 数据检查

```bash
# 进入 Redis 容器
docker compose exec redis redis-cli

# 查看所有用户绑定
KEYS user:*

# 查看特定用户绑定信息
GET user:<openid>

# 查看所有 VM 分配
KEYS vm:*
```

#### VM Orchestrator API

Orchestrator 提供 RESTful API 用于管理虚拟机：

```bash
# 创建 VM (内部调用)
curl -X POST http://localhost:8080/api/v1/vms \
  -H "Content-Type: application/json" \
  -d '{"user_id": "test-user"}'

# 查询 VM 状态
curl http://localhost:8080/api/v1/vms/test-user

# 停止 VM
curl -X DELETE http://localhost:8080/api/v1/vms/test-user

# 健康检查
curl http://localhost:8080/health
```

---

### OpenClaw 插件配置

#### 安装插件

```bash
# 从 npm 安装
openclaw plugins install @haiyanfengli-llc/webhook-server

# 或从源码安装
openclaw plugins install -l ./clawdbot-plugin-webhook-server
```

#### GUI 界面配置（推荐）

1. 浏览器访问 OpenClaw 控制台：`http(s)://<地址>/config`
2. 在侧边栏找到 **Plugins** (插件管理)
3. 选择 **All** 标签页
4. 找到 **WeChat** 卡片，配置 `callbackUrl` 等参数

#### 手动配置

编辑 OpenClaw 配置文件 (`openclaw.json`)：

```json
{
  "channels": {
    "wechat": {
      "enabled": true,
      "config": {
        "callbackUrl": "http://your-bridge-host:3000/callback"
      }
    }
  }
}
```

#### Ngrok 内网穿透（可选）

如果没有公网 IP，可以启用 ngrok：

```json
{
  "plugins": {
    "entries": {
      "webhook-server": {
        "enabled": true,
        "config": {
          "useNgrok": true,
          "ngrokAuthToken": "您的_NGROK_AUTHTOKEN",
          "ngrokPort": 18789,
          "ngrokRegion": "us"
        }
      }
    }
  }
}
```

启动后，在日志中获取生成的公网 URL 用于公众号配置。

---

## 环境变量参考

### WeChat Bridge

| 变量 | 必填 | 默认值 | 说明 |
|------|------|--------|------|
| `PORT` | 否 | `3000` | Bridge 监听端口 |
| `HOST` | 否 | `0.0.0.0` | 监听地址 |
| `WECHAT_APPID` | **是** | — | 微信公众号 AppID |
| `WECHAT_APPSECRET` | **是** | — | 微信公众号 AppSecret |
| `WECHAT_TOKEN` | **是** | — | 微信后台配置的 Token |
| `WECHAT_ENCODING_AES_KEY` | 否 | — | 消息加密密钥 |
| `BRIDGE_BASE_URL` | **是** | — | Bridge 的公网 URL |
| `REDIS_URL` | 否 | `redis://localhost:6379` | Redis 连接地址 |
| `ORCHESTRATOR_URL` | 否 | `http://orchestrator:8080` | Orchestrator API 地址 |
| `SSH_HOST` | 否 | `your-server.example.com` | SSH 连接的公网主机名 |
| `SSH_PORT` | 否 | `2222` | SSH 网关端口 |
| `VM_READY_TIMEOUT_MS` | 否 | `30000` | 等待 VM 启动超时 (毫秒) |
| `LOG_LEVEL` | 否 | `info` | 日志级别 |

### VM Orchestrator

| 变量 | 必填 | 默认值 | 说明 |
|------|------|--------|------|
| `PORT` | 否 | `8080` | API 监听端口 |
| `REDIS_URL` | 否 | `redis://localhost:6379` | Redis 连接地址 |
| `FIRECRACKER_BIN` | 否 | `/usr/local/bin/firecracker` | Firecracker 二进制路径 |
| `KERNEL_PATH` | 否 | `/var/lib/firecracker/vmlinux` | Linux 内核镜像路径 |
| `BASE_ROOTFS_PATH` | 否 | `/var/lib/firecracker/rootfs/clawdbot.ext4` | 基础 rootfs 镜像路径 |
| `DEFAULT_VCPU` | 否 | `1` | 每个 VM 的 vCPU 数量 |
| `DEFAULT_MEM_MIB` | 否 | `512` | 每个 VM 的内存 (MB) |
| `VM_SUBNET` | 否 | `10.0.1.0/24` | VM 网络 CIDR |
| `GATEWAY_IP` | 否 | `10.0.1.1` | 网桥网关 IP |
| `BRIDGE_NAME` | 否 | `fcbr0` | Linux 网桥名称 |
| `VM_DATA_DIR` | 否 | `/var/lib/firecracker/vms` | VM 数据目录 |
| `SOCKET_DIR` | 否 | `/tmp/firecracker/sockets` | API Socket 目录 |

### SSH Gateway

| 变量 | 必填 | 默认值 | 说明 |
|------|------|--------|------|
| `LISTEN_ADDR` | 否 | `0.0.0.0:2222` | SSH 监听地址 |
| `REDIS_URL` | 否 | `redis://localhost:6379` | Redis 连接地址 |
| `HOST_KEY_PATH` | 否 | `/etc/ssh-gateway/host_key` | SSH Host Key 路径 |
| `VM_SSH_USER` | 否 | `user` | VM 内 SSH 用户名 |
| `VM_SSH_PASS` | 否 | `clawdbot` | VM 内 SSH 密码 |

---

## 常见问题

### 部署相关

**Q: `docker compose build` 失败，提示 Go 模块错误？**

A: 确保网络通畅。Dockerfile 已配置 `GOPROXY=https://goproxy.cn,direct` 作为中国镜像加速。如果仍然失败，请检查 Docker 的 DNS 配置。

**Q: Orchestrator 启动报 "firecracker binary not found" 警告？**

A: 这是正常的。Orchestrator 在没有 Firecracker 的情况下也能启动 API 服务，但无法创建 VM。要完整使用，需要在 Linux 裸机上安装 Firecracker 并挂载 `/dev/kvm`。

**Q: Bridge 启动报 "Missing required environment variable" 错误？**

A: 请检查 `.env` 文件是否包含所有必填变量：`WECHAT_APPID`、`WECHAT_APPSECRET`、`WECHAT_TOKEN`、`BRIDGE_BASE_URL`。

### 微信相关

**Q: 微信后台验证 URL 失败？**

A:
1. 确保 Bridge 服务已启动并可从公网访问 (`curl http://your-ip:3000/wechat`)
2. 检查防火墙是否开放 3000 端口
3. 确保 `.env` 中的 `WECHAT_TOKEN` 与微信后台填写的一致

**Q: 发送消息没有回复？**

A:
1. 检查公众号类型是否为非个人认证
2. 查看 Bridge 日志 (`docker compose logs bridge`)
3. 确认 Redis 正常运行
4. 如果绑定了 OpenClaw，检查 OpenClaw 服务是否在线

**Q: 如何解除绑定？**

A: 在公众号发送 `unbind` 即可。

**Q: 日志显示 ECONNREFUSED？**

A: Bridge 无法连接到目标服务 (OpenClaw 或 Orchestrator)。请检查地址和端口是否正确，目标服务是否运行中。

### SSH 相关

**Q: SSH 连接被拒绝？**

A:
1. 检查 SSH Gateway 是否运行中：`docker compose ps ssh-gateway`
2. 检查防火墙是否开放 2222 端口
3. 确认用户已在 Redis 中有 VM 分配记录

**Q: SSH 连接提示密码错误？**

A: 默认密码是 `clawdbot`。如果 VM 内的 SSH 密码已被修改，需要使用修改后的密码。

### Windows WSL 相关

**Q: WSL 安装失败？**

A: WSL 首次安装可能需要重启系统。重启后重新运行 `install-wsl.ps1` 即可继续。

**Q: 如何查看 WSL 中的服务日志？**

```powershell
wsl -d Ubuntu
cat ~/.clawdbot-wechat/gateway.log
cat ~/.clawdbot-wechat/ngrok.log
```

**Q: 如何重启 WSL 中的服务？**

```bash
# 在 WSL 中执行
pkill -f "clawdbot gateway"
pkill ngrok
nohup clawdbot gateway > ~/.clawdbot-wechat/gateway.log 2>&1 &
nohup ngrok http 18789 > ~/.clawdbot-wechat/ngrok.log 2>&1 &
```

**Q: 如何完全卸载 WSL 安装？**

```powershell
wsl --unregister Ubuntu
wsl --uninstall   # 完全移除 WSL
```

---

## 📝 许可证

MIT License
