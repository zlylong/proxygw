# 🧠 SOUL.md - Developer Identity & Project Doctrine

> 这份文档定义了开发者 (longlongago) 的开发哲学、系统架构偏好以及 ProxyGW 项目的核心基准规范。

## 1. 👨‍💻 核心身份与开发哲学 (Identity & Philosophy)

- **原生至上 (Native First)**：彻底拒绝 Docker 等容器化方案带来的网络损耗与复杂度，所有核心服务必须在 **Debian 13 (Trixie)** 环境下 Native 裸机运行。
- **直击底层 (Deep System Control)**：喜欢直接进行代码级修复和深度的 Linux 系统级审计（如：dpkg 依赖修复、内核网络栈调优）。
- **供应链纯净 (Supply Chain Security)**：极度厌恶且**绝对不使用**不稳定的第三方公共镜像（如 mirror.ghproxy.com）。所有组件更新必须直连官方 GitHub API。
- **掌控感 (Precise Control)**：偏好精确的版本控制、特定版本的 Rollback（回滚）能力，以及对底层组件进行原生的 JSON 配置动态注入，拒绝黑盒。

## 2. 🏗️ 核心架构：ProxyGW (Transparent Proxy Gateway)

ProxyGW 是一个高性能的透明代理与网络分流管控系统，核心技术栈为 **Go (Backend) + Vue 3 (Frontend) + SQLite**。

### 核心组件协作机制：
- **Xray-core**：处理所有出入站流量的协议加解密与核心路由。
- **Mosdns (v5+)**：处理 DNS 智能分流与防泄漏。绝不能使用二进制的 geoip.dat 作为 IP 集合，必须拉取纯文本 CIDR。
- **FRR (OSPF)**：用于 Mode B 模式下的动态路由宣告，将代理节点伪装成内网网关。
- **Nftables**：用于 Mode A 模式下的 TProxy 流量透明劫持。

### 运行模式 (Routing Modes)：
- **Mode A (全局网关/TProxy)**：使用 Nftables 劫持所有流量，必须**显式停止 FRR 服务**以防止路由冲突。
- **Mode B (旁路由发布/OSPF)**：使用 FRR 向主路由发送 OSPF 路由宣告，实现无感知的网关接管。

## 3. 💻 技术栈规范与代码约定 (Tech Stack Conventions)

### 🛡️ 系统与安全 (System & Security)
- **内核级调优**：系统必须开启 BBR 拥塞控制与 FQ 队列调度，且 nf_conntrack 连接数上限需放开至 **1,048,576 (104万)** 以上。
- **Systemd 权限沙箱**：服务守护进程必须配置安全加固（NoNewPrivileges=yes 等），防范越权与 Shell 注入。

### ⚙️ 后端 (Go 1.25)
- **数据库**：SQLite，开启 **WAL 模式** 提升并发读写性能。
- **轻量化**：移除不必要的中间件。

### 🎨 前端 (Vue 3 + TailwindCSS)
- **极致的 UX**：表单提交必须有防抖（Debounce）和加载锁（isSubmitting, isLoading）。
- **智能解析**：本地正则表达式自动解析 Share Links (vmess/vless/trojan/ss)。
- **实时监控**：基于 WebSocket 的日志推送引擎，自带断线自动重连机制。

## 4. 🌐 网络资产与拓扑 (Infrastructure Assets)

- **Dev Server**: Debian 13, IP 192.168.20.161 / 155, root.
- **Router**: MikroTik ROS, IP 192.168.20.162, admin.
