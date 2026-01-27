# trap_receiver.py 说明文档

## 用法

```bash
sudo python3 trap_receiver.py | jq .
```

- 可以通过管道连接 `jq` 命令来格式化JSON输出
- 默认监听所有网络接口（0.0.0.0）的162端口
- 修改团体字的字符串进行认证

## 输出结果

程序会输出标准JSON格式的SNMPTrap消息，每条Trap对应一行JSON：

```json
{
  "timestamp": 1706352670,
  "remote_ip": "192.168.0.188",
  "remote_port": 45296,
  "trapType": "v2",
  "trap_oid": "1.3.6.1.4.1.43353.1.100.1",
  "uptime": "1769470260",
  "Binds": {
    "1.3.6.1.4.1.43353.1.100.1": "[Warning] CPU High utilization, currently at 44.60%"
  }
}
```

### 字段说明

- `timestamp`: 接收到Trap的时间戳（Unix时间）
- `remote_ip`: 发送Trap的客户端IP地址
- `remote_port`: 发送Trap的客户端端口
- `trapType`: Trap类型（固定为"v2"表示SNMPv2c）
- `trap_oid`: snmpTrapOID值（实际告警OID）
- `uptime`: 系统运行时间（Timeticks值）
- `Binds`: 自定义变量绑定，包含实际的告警消息内容

注意：输出为单行JSON格式，便于日志采集系统处理。