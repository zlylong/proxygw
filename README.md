# ProxyGW

ProxyGW 是一个 Debian 原生部署的透明代理网关，整合 Xray、Mosdns、nftables(TProxy)、FRR(OSPF)，并提供 Web 管理界面。

## 当前架构（第三轮安全强化后）

- 后端：Go + Gin（模块化路由）
- 前端：Vue 构建产物（`frontend/dist`）
- 数据库：`/root/proxygw/config/proxygw.db`
- 运行配置输出：
  - Xray: `core/xray/config.json`
  - Mosdns: `core/mosdns/config.yaml`
  - 架构：实现了真正的 Fake-IP (198.18.0.0/16) 零延迟方案，结合 Xray 内置 fakedns 与 Mosdns 瞬时返回机制。

后端目录（关键）：

- `backend/main.go`：核心运行逻辑、配置生成、守护任务
- `backend/update_routes.go`：Xray 和 GeoData 的下载与安全校验（支持 SHA256/512 签名防篡改）
- `backend/helpers.go`：可测试的纯函数

## 快速启动与安装

推荐使用提供的一键安装脚本，它将自动配置依赖、编译后端并施加 Systemd 安全沙箱：

```bash
cd /root/proxygw
bash scripts/install.sh
```

手动开发启动：
```bash
cd /root/proxygw/backend
go build -o proxygw-backend .
systemctl restart proxygw
```

## 访问与初始密码

- Web 管理地址: `http://<server-ip>/`
- **安全提示**：系统已**移除**硬编码的默认密码（原 `admin` 弃用）。
- 首次全新安装时，后端会自动生成随机高强度初始密码，并保存在 `/root/proxygw/config/bootstrap_password.txt`。请使用此密码登录并**立即修改**。

## 核心安全特性

1) 供应链防投毒 (Hash Validation)
- Xray 二进制包：下载时强制拉取 `.dgst` 文件，在内存中完成 SHA256/512 比对。若哈希不符，直接阻断安装。
- GeoData (规则库)：更新时同步拉取 `.sha256sum` 并执行强校验。
- 全程使用官方直连，规避不安全的公共镜像站点劫持风险。

2) Systemd 原生沙箱强化 (Systemd Hardening)
- `ProtectSystem=strict`：系统底层完全只读，防止越权篡改核心文件。
- `ReadWritePaths`：基于最小权限原则，仅放开网关配置及通信必要路径（`- /root/proxygw`, `- /usr/local/bin`, `- /etc/frr`）。
- `NoNewPrivileges=yes`：彻底阻断 SUID 提权。
- 独立的 `PrivateTmp`，屏蔽内核参数及 Cgroups 修改权限。

## 其他特性

4) 系统并发与极限优化
- 数据库引入 SQLite WAL 模式，十倍提升读写并发，彻底解决 `database is locked`。
- Gin API 路由开启全局 Gzip 压缩，显著降低日志流与配置拉取的带宽占用。
- /api/login 新增并发限流与错误延时（Rate-limit），防局域网字典爆破。
- 彻底剥离废弃的 Mosdns 日志嗅探下发逻辑，大幅降低 CPU/IO 占用。
- 前端资源完全本地化 (100% 离线断网可用)。


1) 运行模式切换
- Mode A: 纯 TProxy + nftables（停用 FRR）
- Mode B: 基于 OSPF 的动态路由注入（启用 FRR）

2) DNS 与分流（Fake-IP 零延迟架构）
- 路由分流中 `policy=proxy` 的 domain 规则会同步到 `core/mosdns/proxy_domains.txt`
- 命中代理规则的域名由 Mosdns 直接返回保留的 Fake-IP (`198.18.0.0/16`)，完全避免 DNS 泄露与路由表泛洪延迟。
- Xray 的 TProxy 入站开启 sniffing (http, tls, fakedns)，动态还原真实域名。
- OSPF (Mode B) 默认静态宣告 `198.18.0.0/16` 路由，使得首包路由秒级响应。
- 远端 DNS 上游支持 socks5 出站（127.0.0.1:10808）

## 文档索引

- `docs/API.md`
- `docs/OPERATIONS.md`
- `docs/CHANGELOG.md`
