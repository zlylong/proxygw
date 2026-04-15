# ProxyGW 运维手册

## 1. 服务管理

```bash
systemctl status proxygw --no-pager
systemctl status xray --no-pager
systemctl status mosdns --no-pager
systemctl status frr --no-pager
systemctl status nftables --no-pager
```

```bash
systemctl restart proxygw
systemctl restart xray
systemctl restart mosdns
```

## 2. 发布步骤（标准）

```bash
cd /root/proxygw/backend
go test ./... -v
go build -o proxygw-backend .
systemctl restart proxygw
```

## 3. 冒烟验证

```bash
TOKEN=$(curl -s -X POST http://127.0.0.1/api/login \
  -H 'Content-Type: application/json' \
  -d '{"Password":"admin"}' | python3 -c 'import sys,json;print(json.load(sys.stdin)["token"])')

curl -s http://127.0.0.1/api/status -H "Authorization: Bearer $TOKEN"
curl -s http://127.0.0.1/api/xray/versions -H "Authorization: Bearer $TOKEN"
```

## 4. 常见故障排查

### 4.1 Mosdns 启动失败

```bash
journalctl -u mosdns -n 200 --no-pager
/usr/local/bin/mosdns start -c /root/proxygw/core/mosdns/config.yaml
```

重点检查：
- 插件引用顺序（被 exec 的插件必须先定义）
- YAML 行拼接污染
- `ip_set` 必须使用 CIDR 文本，不可直接喂 `geoip.dat`

### 4.2 Xray 更新失败

```bash
journalctl -u proxygw -n 100 --no-pager
ls -l /tmp/xray.zip /tmp/xray/xray
xray version
```

## 5. 备份与回滚

数据库：
```bash
cp /root/proxygw/config/proxygw.db /root/proxygw/config/proxygw.db.bak.$(date +%F-%H%M%S)
```

后端二进制：
```bash
cp /root/proxygw/backend/proxygw-backend /root/proxygw/backend/proxygw-backend.bak.$(date +%F-%H%M%S)
```

Xray：
```bash
curl -s -X POST http://127.0.0.1/api/update/rollback_xray -H "Authorization: Bearer $TOKEN"
```
