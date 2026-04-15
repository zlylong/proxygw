# 1. Update README.md
sed -i '/## 关键特性/a \
\n4) 系统并发与极限优化\n- 数据库引入 SQLite WAL 模式，十倍提升读写并发，彻底解决 `database is locked`。\n- Gin API 路由开启全局 Gzip 压缩，显著降低日志流与配置拉取的带宽占用。\n- /api/login 新增并发限流与错误延时（Rate-limit），防局域网字典爆破。\n- 彻底剥离废弃的 Mosdns 日志嗅探下发逻辑，大幅降低 CPU/IO 占用。\n- 前端资源完全本地化 (100% 离线断网可用)。\n' /root/proxygw/README.md

# 2. Update CHANGELOG.md
sed -i '/## 2026-04-15 (Fake-IP Architecture Upgrade)/i \
## 2026-04-15 (Deep System & UI Optimization)\n\n### Added\n- 后端 SQLite 强制开启 `PRAGMA journal_mode=WAL`，读写全并发。\n- API 响应引入 Gzip 压缩。\n- /api/login 新增动态指数延迟与最大 10 次错误熔断机制。\n- 前端资源 (Vue3, Tailwind, FontAwesome) 完成全量本地化下载，支持 100% 离线脱机运行。\n\n### Changed\n- 彻底移除了 `watchDnsLogs` 协程与相关陈旧的 OSPF 动态路由下发逻辑（由于 Fake-IP 架构已上线，此逻辑为无效高负载开销）。\n\n' /root/proxygw/docs/CHANGELOG.md
