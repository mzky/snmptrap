# 不同Linux发行版的命令兼容性说明

本文档介绍了SNMPTrap监控系统中使用的命令在CentOS、openEuler、Debian和Ubuntu发行版上的兼容性情况。

注意为了兼容中文环境，命令前缀需要增加`LC_ALL=C`

## 命令兼容性分析

### 1. CPU使用率监测

#### 方案1: 使用top命令 (推荐)
```bash
top -bn1 | grep '%Cpu(s):' | awk '{print 100-$8}' | sed 's/id,//'
```
- **CentOS**: ✓ 兼容
- **openEuler**: ✓ 兼容
- **Debian**: ✓ 兼容
- **Ubuntu**: ✓ 兼容
- **注意**: 提取idle百分比并计算使用率

#### 方案2: 使用sar命令
```bash
LC_ALL=C sar -u 1 1 | awk 'END {print 100 -$NF}'
```
- **CentOS**: ✓ 需安装sysstat包
- **openEuler**: ✓ 需安装sysstat包
- **Debian**: ✓ 需安装sysstat包
- **Ubuntu**: ✓ 需安装sysstat包

### 2. 内存使用率监测

#### 方案1: 使用free命令
```bash
free | awk '/^Mem:/ {printf "%.2f", ($3-$6)/$2*100}'
```
- **CentOS**: ✓ 兼容
- **openEuler**: ✓ 兼容
- **Debian**: ✓ 兼容
- **Ubuntu**: ✓ 兼容

#### 方案2: 使用/proc/meminfo (推荐)
```bash
awk '/^MemTotal:|^MemAvailable:/ {a[$1]=$2} END {printf "%.2f", (a["MemTotal:"]-a["MemAvailable:"])/a["MemTotal:"]*100}' /proc/meminfo
```
- **CentOS**: ✓ 兼容
- **openEuler**: ✓ 兼容
- **Debian**: ✓ 兼容
- **Ubuntu**: ✓ 兼容
- **优势**: 不依赖外部命令，直接读取内核接口

### 3. 磁盘使用率监测

#### 方案1: 使用df命令
```bash
df / | awk 'NR==2 {print $5}' | sed 's/%//'
```
- **CentOS**: ✓ 兼容
- **openEuler**: ✓ 兼容
- **Debian**: ✓ 兼容
- **Ubuntu**: ✓ 兼容

### 4. 其他监控项

#### 系统负载
```bash
cat /proc/loadavg | awk '{print $1}'
```
- **兼容性**: 所有主流发行版均支持

#### 进程数量
```bash
ps -e --no-headers | wc -l
```
- **兼容性**: 所有主流发行版均支持

## 推荐配置策略

1. **优先使用/proc文件系统**: 直接从内核接口获取数据最稳定
2. **备用传统命令**: 如free、top等，作为兼容性备选
3. **统一locale**: 使用`LC_ALL=C`确保输出格式一致
4. **错误处理**: 程序应能处理命令执行失败的情况

## 安装依赖

### CentOS/openEuler
```bash
sudo yum install sysstat -y
```

### Debian/Ubuntu
```bash
sudo apt-get install sysstat -y
```

## 测试命令

在部署前，建议在目标系统上测试相关命令：

```bash
# 测试CPU使用率
echo "CPU Usage: $(top -bn1 | grep '%Cpu(s):' | awk '{print 100-$8}' | sed 's/id,//')%"

# 测试内存使用率
echo "Memory Usage: $(free | awk '/^Mem:/ {printf "%.2f", ($3-$6)/$2*100}')%"

# 测试内存使用率 (/proc/meminfo)
echo "Memory Usage: $(awk '/^MemTotal:|^MemAvailable:/ {a[$1]=$2} END {printf "%.2f", (a["MemTotal:"]-a["MemAvailable:"])/a["MemTotal:"]*100}' /proc/meminfo)%"

# 测试磁盘使用率
echo "Disk Usage: $(df / | awk 'NR==2 {print $5}' | sed 's/%//')%"

# 测试系统负载
echo "Load Average: $(cat /proc/loadavg | awk '{print $1}')"

# 测试进程数量
echo "Process Count: $(ps -e --no-headers | wc -l)"
```