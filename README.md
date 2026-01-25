# SNMP Trap 监控系统

使用JSON配置的通用SNMP Trap监控与告警系统，支持自定义指标、脚本和命令执行。

## 功能特性

- **多指标监控**：支持CPU、内存、磁盘等系统指标以及自定义指标监控
- **灵活的数据源**：支持执行shell命令或脚本文件获取监控数据
- **智能告警机制**：基于阈值的告警，支持告警和清除消息

- **可配置间隔**：可设置监控检查的时间间隔

## 系统要求

- Linux操作系统
- Go 1.16+
- SNMP服务（snmpd）

## 安装与编译

```bash
# 编译项目
go build snmptrap.go

# 或者直接运行
go run snmptrap.go
```

## 命令行参数

程序支持以下命令行参数：

- `-h`: 显示帮助信息
- `-c <配置文件路径>`: 指定配置文件路径，默认为config.json
- `-p`: 打印详细日志信息
- `-v`: 显示版本信息

### 使用示例

```bash
# 显示帮助信息
./snmptrap -h

# 使用指定配置文件
./snmptrap -c /path/to/custom_config.json

# 启用详细日志输出
./snmptrap -p

# 显示版本信息
./snmptrap -v

# 组合使用多个参数
./snmptrap -c /path/to/config.json -p
```

## 配置说明

配置文件为 [config.json](file:///root/go/src/snmptrap/config.json)，包含以下部分：

### 全局配置

```json
{
  "config": {
    "interval": 3,
    "snmp": {
      "community": "bjca@2019",
      "target": "192.168.0.100",
      "port": 162
    }
  }
}
```

- `interval`: 监控检查间隔（秒）
- `community`: SNMP社区字符串
- `target`: SNMP接收服务器地址
- `port`: SNMP端口（通常是162）

### 指标配置

```json
{
  "metrics": [
    {
      "name": "指标名称",
      "type": "script|command",
      "command": "脚本路径或命令",
      "warningOid": "告警OID",
      "clearOid": "清除告警OID",
      "warningTemplate": "告警消息模板",
      "clearTemplate": "清除消息模板",
      "threshold": 阈值
    }
  ]
}
```

#### 配置参数详解

- `name`: 指标名称，用于标识不同的监控项
- `type`: 数据获取方式
  - `script`: 执行脚本文件
  - `command`: 执行shell命令
- `command`: 具体的脚本路径或命令
- `warningOid`: 触发告警时发送的OID
- `clearOid`: 告警解除时发送的OID
- `warningTemplate`: 告警消息模板，使用`%.2f`格式化数值
- `clearTemplate`: 清除消息模板，使用`%.2f`格式化数值
- `threshold`: 告警阈值，当指标值超过此值时触发告警

#### 特殊阈值说明

- 所有指标都是数值型指标，会根据阈值判断是否发送告警或清除消息

## 使用示例

### 监控CPU使用率

```json
{
  "name": "cpu_usage",
  "type": "command",
  "command": "top -bn1 | grep \"Cpu(s)\" | awk '{print $2}' | sed 's/%us,//'",
  "warningOid": ".1.3.6.1.4.1.43353.1.100.1",
  "clearOid": ".1.3.6.1.4.1.43353.1.100.4",
  "warningTemplate": "[Warning] CPU High utilization, currently at %.2f%%",
  "clearTemplate": "[Clear] CPU alert cleared, currently at %.2f%%",
  "threshold": 80.0
}
```

### 监控内存使用率

```json
{
  "name": "memory_usage",
  "type": "command",
  "command": "free | grep Mem | awk '{printf(\"%.2f\", $3/$2 * 100.0)}'",
  "warningOid": ".1.3.6.1.4.1.43353.1.100.3",
  "clearOid": ".1.3.6.1.4.1.43353.1.100.6",
  "warningTemplate": "[Warning] Memory High utilization, currently at %.2f%%",
  "clearTemplate": "[Clear] Memory alert cleared, currently at %.2f%%",
  "threshold": 85.0
}
```

### 使用脚本文件

```json
{
  "name": "custom_script",
  "type": "script",
  "command": "/path/to/script.sh",
  "warningOid": ".1.3.6.1.4.1.43353.1.100.7",
  "clearOid": ".1.3.6.1.4.1.43353.1.100.8",
  "warningTemplate": "[Warning] Custom metric exceeded: %.2f",
  "clearTemplate": "[Clear] Custom metric normal: %.2f",
  "threshold": 10.0
}
```

```

## 告警逻辑

1. **告警触发**：当指标值超过阈值且之前未处于告警状态时，发送告警SNMP Trap
2. **告警保持**：当指标值持续超过阈值时，不会重复发送告警消息
3. **告警清除**：当指标值回落到阈值以下且之前处于告警状态时，发送清除SNMP Trap
4. 系统会根据阈值判断是否发送告警或清除消息

## SNMP Trap 格式

SNMP Trap 包含以下变量：
- 自定义OID（告警或清除）
- `.1.3.6.1.2.1.1.5.0` - 系统名称（主机名）
- `.1.3.6.1.2.1.1.3.0` - 系统运行时间

## 扩展功能

- 支持执行任意shell命令获取指标数据
- 支持执行脚本文件获取复杂指标数据
- 支持多种数据类型处理（整数、浮点数、字符串）
- 内置Linux系统指标获取函数（CPU、内存、磁盘）

## 日志记录

程序会记录以下信息：
- SNMP服务启动状态
- 指标采集结果
- 告警和清除事件
- 错误和异常情况

## 故障排除

如果遇到问题，请检查：
1. SNMP服务是否正常运行
2. 配置文件格式是否正确
3. 命令或脚本是否有执行权限
4. 网络连接是否正常
5. OID是否符合SNMP标准