#!/usr/bin/env python3
# -*- coding: utf-8 -*-
"""
用法：sudo python3 trap_receiver.py |jq .
输出标准JSON
"""

import asyncio
import json
import time
import sys
from pysnmp.entity import engine, config
from pysnmp.entity.rfc3413 import ntfrcv
from pysnmp.carrier.asyncio.dgram import udp
from pysnmp.proto.api import v2c

# ===== 配置区 =====
BIND_IP = "0.0.0.0"
BIND_PORT = 162
COMMUNITY = "bjca@2019"

# 创建SNMP引擎
snmpEngine = engine.SnmpEngine()

# 配置UDP传输（使用新版 DOMAIN_NAME 常量）
config.addTransport(
    snmpEngine,
    udp.DOMAIN_NAME,  # 修正1: 替换弃用的 domainName
    udp.UdpTransport().openServerMode((BIND_IP, BIND_PORT))
)

# 配置SNMPv2c认证
config.addV1System(snmpEngine, "my-area", COMMUNITY)

def cbFun(snmpEngine, stateReference, contextEngineId, contextName,
          varBinds, cbCtx):
    """Trap回调函数 - 格式化为JSON输出"""
    # 修正2: 通过 stateReference 获取远程地址（asyncio 安全方式）
    transportDomain, transportAddress = snmpEngine.msgAndPduDsp.getTransportInfo(stateReference)
    
    # 提取Trap元数据（前2个是系统变量: sysUpTime + snmpTrapOID）
    trap_data = {
        "timestamp": int(time.time()),
        "remote_ip": transportAddress[0],
        "remote_port": transportAddress[1],
        "trapType": "v2",
        "trap_oid": str(varBinds[1][1]),  # snmpTrapOID.0
        "uptime": str(varBinds[0][1]),    # sysUpTime
        "Binds": {}
    }
    
    # 提取自定义绑定变量（跳过前2个系统变量）
    for oid, val in varBinds[2:]:
        trap_data["Binds"][str(oid)] = str(val)
    
    # 输出标准JSON（单行，便于日志采集）
    print(json.dumps(trap_data, ensure_ascii=False))
    sys.stdout.flush()

# 注册Trap处理器
ntfrcv.NotificationReceiver(snmpEngine, cbFun)

# 使用 asyncio 事件循环持续运行（无需 dispatch()）
try:
    asyncio.get_event_loop().run_forever()
except KeyboardInterrupt:
    print("\n🛑 接收器已停止")
    snmpEngine.transportDispatcher.closeDispatcher()
