# ProxyGW - 现代化的透明代理网关

ProxyGW 是一个高性能、易于使用的透明代理网关系统。它提供了美观的 Web 管理界面，让你能够轻松接管家庭或办公室的网络流量，实现智能分流。


![ProxyGW Dashboard](docs/assets/dashboard.jpg)

## ✨ 核心特性

- **开箱即用**：提供完善的 Web 管理后台，告别繁琐的命令行配置，全图形化管理节点、规则与设备。
- **智能分流**：内置强大的域名和 IP 规则库，国内网站直连，特殊流量走代理，彻底告别卡顿与 DNS 污染。
- **无感接管**：支持全局网关接管 (Mode A)、纯 Fake-IP 旁路 (Mode B) 或 纯 OSPF 动态播报 (Mode C)，局域网设备无需设置即可科学访问网络。
- **远程节点部署**：独创的一键式远程节点部署系统，支持录入多台海外 Linux 主机，由网关中控自动通过 SSH 下发、配置、并实时监控 WireGuard/VLESS 隧道协议。
- **极致安全**：系统随机生成高强度初始密码，前端自带防爆破延时；SQLite 落库的 SSH 凭证由内存态 AES-256-CFB 动态加解密，Web UI 资源完全本地化。

## 🚀 快速安装

推荐在全新安装的 Debian 12 / 13 服务器上使用 root 用户执行以下命令进行一键部署：

```bash
git clone https://github.com/zlylong/proxygw.git /root/proxygw
cd /root/proxygw
bash scripts/install.sh
```

## 🔑 初始登录

为了系统安全，ProxyGW **没有默认固定密码**。首次安装完成后，请在服务器终端查看系统为您随机生成的初始密码：

```bash
cat /root/proxygw/config/bootstrap_password.txt
```

在浏览器中输入网关服务器的 IP 地址（如 `http://192.168.x.x/`），使用该初始密码登录。
**⚠ 强烈建议：请在首次登录后立即前往系统设置修改您的密码。** 修改后该 txt 文件将自动作废。

## 🕹️ 路由模式与使用指南

ProxyGW 设计了三种物理隔离的网络接管模式，以适应不同级别的家庭/办公网络拓扑。

### 🟢 Mode A: 全局网关劫持 (推荐新手使用)
在这个模式下，ProxyGW 作为局域网的“旁路由”存在，强行接管所有设备的流量。
**适用场景**：主路由是普通的家用路由器（如小米、TP-Link、华硕等），无需任何高级网络知识。

**举例使用方式：**
1. **全屋接管**：登录您家里的主路由器后台，找到 **DHCP 服务器设置**。将 **默认网关 (Default Gateway)** 和 **DNS 服务器** 修改为 ProxyGW 服务器的局域网 IP 地址（例如 `192.168.1.100`）。保存并重启路由器。此时，连上 WiFi 的所有设备都会自动翻墙。
2. **单设备独享（按需科学）**：如果您不想影响家人，只想让自己的手机或电脑走代理。只需在手机/电脑的 WiFi 设置中，将 IP 获取方式从“自动(DHCP)”改为“手动/静态”，然后把 **网关** 和 **DNS** 填成 ProxyGW 的 IP 即可。

### 🔵 Mode B: 纯 Fake-IP 模式 (零延迟 / 免防环路配置)
这是极其纯粹的性能模式。Mosdns 将开启 Fake-IP，而 OSPF **仅仅**向主路由宣告一个虚拟的 IP 池 (`198.18.0.0/16`)，完全不需要下发真实的海外 GeoIP。
**适用场景**：主路由是 ROS / OpenWrt 等支持 OSPF 的路由器。推荐不想折腾“防环路”的高级玩家。

**举例使用方式：**
1. 您的手机、电视和电脑的网关依然指向主路由器。**⚠ 但 DHCP 下发的 DNS 服务器，必须指向 ProxyGW 的 IP 地址**。
2. 在主路由器中配置 OSPF，将 ProxyGW 设为邻居。
3. **免疫环路**：因为您的海外节点真实 IP 不可能是 `198.18.x.x`，所以主路由永远不会将发往节点的包踢回给 ProxyGW，天然免疫 OSPF 环路。

### 🟣 Mode C: 纯 OSPF 动态旁路模式 (传统的 GeoIP 分流)
关闭 Fake-IP 功能。Mosdns 返回真实的海外目标 IP，ProxyGW 将数以千计的真实 GeoIP (如 Netflix, Telegram 等网段) 动态推给主路由。
**适用场景**：对 Fake-IP 机制敏感（如某些严格校验目标 IP 的 P2P 游戏或 App报错）的高级玩家。

**⚠️ Mode C 的局限性与 Geosite 智能回退说明**：
在纯 OSPF 模式下，路由器的物理分流只能依赖**真实 IP (GeoIP)**，而无法直接处理域名。
为了对齐用户的直觉，当您在管理面板添加一条 `geosite` 域名规则（如 `geosite:telegram`）时，后端不仅会做 DNS 劫持，还会**触发智能回退（Smart Fallback）机制**：自动去 `geoip.dat` 中寻找同名标签，提取其包含的真实 IP 网段并广播给主路由。
*由于现代 CDN 的复杂性，如果某网站的动态 CDN IP 未被开源的 GeoIP 库收录，该流量将无法被 OSPF 路由接管，从而导致直连或漏网。如果您对“不漏网”有极致要求，请使用 Mode B。*

**注意防环路**：由于 OSPF 会播报真实的海外网段，如果您的代理节点 IP 刚好在这个网段里，就会形成死循环断网！您必须在主路由器上配置 **源地址绕过 (PBR 策略路由)**：让来自 ProxyGW IP 的流量强制走外网，无视 OSPF 路由。
*(👉 ROS v7 示例：新建一个 `bypass_proxy` 路由表指向公网 WAN 口，然后执行 `/routing rule add src-address=<ProxyGW_IP>/32 action=lookup-only-in-table table=bypass_proxy`，详见[运维文档](./docs/OPERATIONS.md))*


## 📚 文档指南

如果您是系统管理员、网络工程师或希望进行二次开发的工程师，请查阅 `docs/` 目录下的深入文档：

- 🛠️ [开发者与架构指南](./docs/DEVELOPER.md) - 深入系统底层架构、源码结构、Fake-IP 零延迟原理与内核级调优。
- ⚙️ [运维与故障排查](./docs/OPERATIONS.md) - 系统管理员的服务管理、脚本升级、系统卸载与常见故障排除手册。
- 🔌 [后端 API 接口参考](./docs/API.md) - 面向开发者的 RESTful API 接口说明。


## 🙏 致谢 (Acknowledgments)

本项目底层的核心网络代理与 DNS 解析分流能力，离不开以下优秀的开源项目，特此致谢：

- [XTLS/Xray-core](https://github.com/XTLS/Xray-core) - 提供了极其强大且高性能的透明代理与协议卸载能力。
- [IrineSistiana/mosdns](https://github.com/IrineSistiana/mosdns) - 提供了灵活且高效的 DNS 转发、GeoIP 分流与防泄漏解析引擎。

感谢所有开源贡献者为构建更自由、开放的网络环境所做出的无私奉献！
