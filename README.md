# ProxyGW - 现代化的透明代理网关

ProxyGW 是一个高性能、易于使用的透明代理网关系统。它提供了美观的 Web 管理界面，让你能够轻松接管家庭或办公室的网络流量，实现智能分流。


![ProxyGW Dashboard](docs/assets/dashboard.jpg)

## ✨ 核心特性

- **开箱即用**：提供完善的 Web 管理后台，告别繁琐的命令行配置，全图形化管理节点、规则与设备。
- **智能分流**：内置强大的域名和 IP 规则库，国内网站直连，特殊流量走代理，彻底告别卡顿与 DNS 污染。
- **无感接管**：支持全局网关劫持 (Mode A) 或 OSPF 动态旁路由播报 (Mode B)，局域网手机、电视、PC 等设备无需任何设置即可科学访问网络。
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
