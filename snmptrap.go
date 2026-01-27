package main

import (
	"encoding/json"
	"flag"
	"fmt"
	"io"
	"log"
	"os"
	"os/exec"
	"runtime"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/gosnmp/gosnmp"
)

// Metric 指标配置结构
type Metric struct {
	Name            string  `json:"name"`
	Type            string  `json:"type"`    // "script" or "command"
	Command         string  `json:"command"` // 通用的命令或脚本路径
	WarningOid      string  `json:"warningOid"`
	ClearOid        string  `json:"clearOid"`
	WarningTemplate string  `json:"warningTemplate"`
	ClearTemplate   string  `json:"clearTemplate"`
	Threshold       float64 `json:"threshold"`
}

// Config 全局配置结构
type Config struct {
	Interval int64 `json:"interval"`
	SNMP     struct {
		Community string   `json:"community"`
		Targets   []string `json:"targets"`
		Port      uint16   `json:"port"`
	} `json:"snmp"`
}

// JsonConfig 配置结构
type JsonConfig struct {
	Config     Config                 `json:"config"`
	Metrics    []Metric               `json:"metrics"`
	Extensions map[string]interface{} `json:"extensions"`
}

// MetricData 指标数据结构
type MetricData struct {
	MetricName   string
	CurrentValue float64
	Threshold    float64
	Unit         string
	IsWarning    bool
}

// AlertStatus 告警状态跟踪
type AlertStatus struct {
	IsInAlert map[string]bool
	mu        sync.Mutex
}

var (
	showHelp    = flag.Bool("h", false, "help")
	configPath  = flag.String("c", "config.json", "config path")
	printLog    = flag.Bool("p", false, "print log")
	showVersion = flag.Bool("v", false, "version")
)

const Version = "1.0.0"

// CmdArgs 命令行参数处理结构
type CmdArgs struct {
	ConfigPath string
	PrintLog   bool
}

func main() {
	// 解析并处理命令行参数
	cmdArgs, err := parseCmdArgs()
	if err != nil {
		log.Fatalf("Error parsing command line arguments: %v", err)
	}

	// 设置日志输出
	setupLogger(cmdArgs.PrintLog)

	// 读取配置文件
	config, err := loadConfig(cmdArgs.ConfigPath)
	if err != nil {
		log.Fatalf("Failed to load config: %v", err)
	}

	// 初始化多目标SNMP客户端
	snmpClient := initMultiSNMP(config)
	defer snmpClient.Close()

	// 初始化告警状态跟踪
	alertStatus := AlertStatus{
		IsInAlert: make(map[string]bool),
	}

	// 启动定时任务（异步非阻塞方式）
	go startMonitoringTask(snmpClient, config, &alertStatus)

	// 保持主程序运行
	select {}

}

// 解析命令行参数
func parseCmdArgs() (*CmdArgs, error) {
	// 解析命令行参数
	flag.Parse()

	// 处理版本信息请求
	if *showVersion {
		fmt.Printf("SNMPTrap Monitor v%s\n", Version)
		fmt.Printf("Go Version: %s\n", runtime.Version())
		fmt.Printf("OS/Arch: %s/%s\n", runtime.GOOS, runtime.GOARCH)
		os.Exit(0)
	}

	// 处理帮助信息请求
	if *showHelp {
		fmt.Println("SNMPTrap Monitor - Universal SNMPTrap monitoring and alerting system")
		fmt.Println()
		fmt.Println("Usage:")
		fmt.Printf("  %s [options]\n", os.Args[0])
		fmt.Println()
		fmt.Println("Options:")
		flag.PrintDefaults()
		os.Exit(0)
	}

	return &CmdArgs{
		ConfigPath: *configPath,
		PrintLog:   *printLog,
	}, nil
}

// 设置日志配置
func setupLogger(printLog bool) {
	if !printLog { // 如果不打印详细日志，则禁用日志输出
		log.SetOutput(io.Discard)
	} else { // 如果需要打印详细日志，则添加文件和行号信息
		log.SetFlags(log.LstdFlags | log.Lshortfile)
	}
}

// 启动监控任务（异步非阻塞）
func startMonitoringTask(snmpClient *MultiTargetSNMPClient, c *JsonConfig, alertStatus *AlertStatus) {
	// 设置监控间隔
	interval := time.Duration(c.Config.Interval) * time.Second
	ticker := time.NewTicker(interval)
	defer ticker.Stop()

	log.Printf("Starting SNMPtrap sender with interval %s, targets: %v", interval, c.Config.SNMP.Targets)

	for {
		select {
		case <-ticker.C: // 发送所有指标的trap消息
			for _, metric := range c.Metrics {
				sendMetricTrap(snmpClient, metric, alertStatus)
			}
		}
	}
}

// 加载配置文件
func loadConfig(filename string) (*JsonConfig, error) {
	file, err := os.Open(filename)
	if err != nil {
		return nil, fmt.Errorf("failed to open config file: %w", err)
	}
	defer file.Close()

	var c JsonConfig
	if err := json.NewDecoder(file).Decode(&c); err != nil {
		return nil, fmt.Errorf("failed to decode config: %w", err)
	}
	// 添加配置验证
	if len(c.Config.SNMP.Targets) == 0 {
		return nil, fmt.Errorf("SNMP targets are required")
	}
	if c.Config.SNMP.Community == "" {
		return nil, fmt.Errorf("SNMP community is required")
	}
	if c.Config.SNMP.Port == 0 {
		c.Config.SNMP.Port = 162 // 默认端口
	}

	return &c, nil
}

// MultiTargetSNMPClient 多目标SNMP客户端包装器
type MultiTargetSNMPClient struct {
	clients []*gosnmp.GoSNMP
}

// 初始化多目标SNMP客户端
func initMultiSNMP(c *JsonConfig) *MultiTargetSNMPClient {
	var clients []*gosnmp.GoSNMP

	for _, target := range c.Config.SNMP.Targets {
		g := &gosnmp.GoSNMP{
			Target:    target,
			Port:      c.Config.SNMP.Port,
			Community: c.Config.SNMP.Community,
			Version:   gosnmp.Version2c,
			Timeout:   time.Duration(2) * time.Second,
		}

		if err := g.Connect(); err != nil {
			log.Printf("Failed to connect to SNMP server %s: %v", target, err)
			continue
		}

		clients = append(clients, g)
	}

	return &MultiTargetSNMPClient{
		clients: clients,
	}
}

// SendTrap 发送到所有目标
func (m *MultiTargetSNMPClient) SendTrap(trap gosnmp.SnmpTrap) {
	var wg sync.WaitGroup

	for i, client := range m.clients {
		wg.Add(1)
		go func(index int, c *gosnmp.GoSNMP) {
			defer wg.Done()

			if _, err := c.SendTrap(trap); err != nil {
				log.Printf("Failed to send SNMPtrap to target %s: %v", c.Target, err)
			} else {
				log.Printf("Successfully sent SNMPtrap to target %s", c.Target)
			}
		}(i, client)
	}

	wg.Wait()
}

// Close 关闭所有连接
func (m *MultiTargetSNMPClient) Close() {
	for _, client := range m.clients {
		if client.Conn != nil {
			client.Conn.Close()
		}
	}
}

// 发送指标的trap消息
func sendMetricTrap(snmpClient *MultiTargetSNMPClient, metric Metric, alertStatus *AlertStatus) {
	// 获取实际指标数据
	metricData, err := getMetricData(metric.Type, metric.Command, metric.Threshold)
	if err != nil {
		log.Printf("Failed to get metric data for %s: %v", metric.Name, err)
		return
	}

	// 关键修复：设置指标名称，因为getMetricData中MetricName为空
	metricData.MetricName = metric.Name

	// 检查是否需要发送告警
	if metricData.IsWarning {
		// 当指标值超过阈值时，每次检查都重复发送告警消息
		message := fmt.Sprintf(metric.WarningTemplate, metricData.CurrentValue)

		// 准备trap数据 - 遵循SNMPv2c TRAP2 PDU规范
		// 在SNMPv2c中，trap消息应包含snmpTrapOID作为第一个varbind，然后是enterprise特定的OIDs
		vars := []gosnmp.SnmpPDU{
			{
				// SNMPv2-MIB::snmpTrapOID.0 - 必需的陷阱OID
				Name:  ".1.3.6.1.6.3.1.1.4.1.0", // snmpTrapOID.0
				Type:  gosnmp.ObjectIdentifier,
				Value: metric.WarningOid, // 实际的告警OID
			},
			{
				Name:  metric.WarningOid,
				Type:  gosnmp.OctetString,
				Value: []byte(message),
			},
		}

		// 创建SNMPTrap
		trap := gosnmp.SnmpTrap{
			Variables: vars,
		}

		// 发送trap消息到所有目标
		log.Printf("Sending WARNING SNMPTrap for %s: %s", metric.Name, message)
		snmpClient.SendTrap(trap)
		// 更新状态：标记为在告警状态
		alertStatus.mu.Lock()
		alertStatus.IsInAlert[metric.Name] = true
		alertStatus.mu.Unlock()
	} else {
		// 检查是否需要发送清除消息
		alertStatus.mu.Lock()
		isInAlert := alertStatus.IsInAlert[metric.Name]
		alertStatus.mu.Unlock()

		if isInAlert {
			// 从告警状态转为正常状态，发送清除消息（只发送一次）
			message := fmt.Sprintf(metric.ClearTemplate, metricData.CurrentValue)

			// 准备trap数据 - 遵循SNMPv2c TRAP2 PDU规范
			// 在SNMPv2c中，trap消息应包含snmpTrapOID作为第一个varbind，然后是enterprise特定的OIDs
			vars := []gosnmp.SnmpPDU{
				{
					// SNMPv2-MIB::snmpTrapOID.0 - 必需的陷阱OID
					Name:  ".1.3.6.1.6.3.1.1.4.1.0", // snmpTrapOID.0
					Type:  gosnmp.ObjectIdentifier,
					Value: metric.ClearOid, // 实际的清除OID
				},
				{
					Name:  metric.ClearOid,
					Type:  gosnmp.OctetString,
					Value: []byte(message),
				},
			}

			// 创建SNMPTrap
			trap := gosnmp.SnmpTrap{
				Variables: vars,
			}

			// 发送trap消息到所有目标
			log.Printf("Sending CLEAR SNMPTrap for %s: %s", metric.Name, message)
			snmpClient.SendTrap(trap)
			// 更新状态：标记为不在告警状态
			alertStatus.mu.Lock()
			alertStatus.IsInAlert[metric.Name] = false
			alertStatus.mu.Unlock()
		}
	}
}

// 获取指标数据 - 支持脚本和命令
func getMetricData(metricType, command string, threshold float64) (MetricData, error) {
	// 执行命令或脚本
	var outputStr string
	var err error

	switch metricType {
	case "script": // 将command作为脚本路径执行
		outputStr, err = executeScriptAsPathRaw(command)
	case "command": // 将command作为命令行执行
		outputStr, err = executeCommandRaw(command)
	default:
		return MetricData{}, fmt.Errorf("unsupported metric type: %s, only 'script' or 'command' are supported", metricType)
	}

	if err != nil {
		return MetricData{}, err
	}

	// 尝试解析为浮点数
	resultStr := strings.TrimSpace(outputStr)
	value, parseErr := strconv.ParseFloat(resultStr, 64)

	if parseErr != nil {
		return MetricData{}, fmt.Errorf("command output is not a valid number: %s", resultStr)
	}

	// 判断是否超过阈值
	isWarning := value > threshold

	return MetricData{
		MetricName:   "",
		CurrentValue: value,
		Threshold:    threshold,
		Unit:         "%",
		IsWarning:    isWarning,
	}, nil
}

// 执行脚本文件并返回原始输出
func executeScriptAsPathRaw(scriptPath string) (string, error) {
	if _, err := os.Stat(scriptPath); os.IsNotExist(err) {
		return "", fmt.Errorf("script file does not exist: %s", scriptPath)
	}

	cmd := exec.Command(scriptPath)
	output, err := cmd.Output()
	if err != nil {
		return "", fmt.Errorf("failed to execute script file '%s': %v", scriptPath, err)
	}

	return string(output), nil
}

// 执行命令行并返回原始输出
func executeCommandRaw(command string) (string, error) {
	cmd := exec.Command("/bin/bash", "-c", command)
	output, err := cmd.Output()
	if err != nil {
		return "", fmt.Errorf("failed to execute command '%s': %v", command, err)
	}

	return string(output), nil
}
