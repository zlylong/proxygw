# ProxyGW API 文档

Base URL: `http://<host>/api`

## 认证

### POST /login
请求：
```json
{ "password": "admin" }
```
返回：
```json
{ "token": "<dynamic-token>" }
```

说明：token 为后端启动时动态生成。

## 配置查看

- GET `/config/xray`
- GET `/config/mosdns`

## 系统状态

### GET /status
返回关键字段：
- `xray`, `ospf`, `mosdns`（服务状态）
- `mode`（A/B）
- `xrayVersion`, `geoVersion`
- `cpu`, `ram`, `up`, `down`

## 模式切换

### POST /mode
请求：
```json
{ "mode": "A" }
```
行为：
- A: `start nftables`, `stop frr`
- B: `start nftables`, `start frr`
- 自动触发 Mosdns/Xray 配置重生成

## 节点管理

- GET `/nodes`
- POST `/nodes`
- POST `/nodes/import`（支持 vmess:// 与 vless://）
- POST `/nodes/ping`
- PUT `/nodes/:id/toggle`
- DELETE `/nodes/:id`

## 规则管理

- GET `/rules/categories`
- GET `/rules`
- POST `/rules`
- DELETE `/rules/:id`

## DNS

- GET `/dns`
- POST `/dns`

请求示例：
```json
{ "local": "223.5.5.5,114.114.114.114", "remote": "8.8.8.8,1.1.1.1", "lazy": true }
```

- GET `/dns/logs`
- GET `/dns/logs/ws`

## OSPF

- GET `/ospf`

## 组件更新

- GET `/xray/versions`
- POST `/update/geodata`
- POST `/update/xray`
- POST `/update/rollback_xray`

### POST /update/xray
请求（可选 version）：
```json
{ "version": "v26.3.27" }
```
- 不传或传 `latest` => 最新版
- 自定义版本需通过白名单正则：`^v[0-9A-Za-z._-]+$`

## 一键应用

- POST `/apply`
