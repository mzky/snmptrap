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
	"time"

	"github.com/gosnmp/gosnmp"
)

// Config 配置结构
type Config struct {
	Config struct {
		Interval int `json:"interval"`
		SNMP     struct {
			Community string `json:"community"`
			Target    string `json:"target"`
			Port      int    `json:"port"`
		} `json:"snmp"`
	} `json:"config"`
	Metrics []struct {
		Name            string  `json:"name"`
		Type            string  `json:"type"`    // "script" or "command"
		Command         string  `json:"command"` // 通用的命令或脚本路径
		WarningOid      string  `json:"warningOid"`
		ClearOid        string  `json:"clearOid"`
		WarningTemplate string  `json:"warningTemplate"`
		ClearTemplate   string  `json:"clearTemplate"`
		Threshold       float64 `json:"threshold"`
	} `json:"metrics"`
	Extensions map[string]interface{} `json:"extensions"`
}

// MetricData 指标数据结构
type MetricData struct {
	Hostname     string
	MetricName   string
	CurrentValue float64
	Threshold    float64
	Unit         string
	IsWarning    bool
}

// AlertStatus 告警状态跟踪
type AlertStatus struct {
	IsInAlert map[string]bool
}

var (
	showHelp    = flag.Bool("h", false, "help")
	configPath  = flag.String("c", "config.json", "config path")
	printLog    = flag.Bool("p", false, "print log")
	showVersion = flag.Bool("v", false, "version")
)

const Version = "1.0.0"

func main() {
	// 解析命令行参数
	flag.Parse()

	// 处理版本信息请求
	if *showVersion {
		fmt.Printf("SNMP Trap Monitor v%s\n", Version)
		fmt.Printf("Go Version: %s\n", runtime.Version())
		fmt.Printf("OS/Arch: %s/%s\n", runtime.GOOS, runtime.GOARCH)
		return
	}

	// 处理帮助信息请求
	if *showHelp {
		fmt.Println("SNMP Trap Monitor - Universal SNMP Trap monitoring and alerting system")
		fmt.Println()
		fmt.Println("Usage:")
		fmt.Printf("  %s [options]\n", os.Args[0])
		fmt.Println()
		fmt.Println("Options:")
		flag.PrintDefaults()
		return
	}

	// 设置日志输出
	if !*printLog {
		// 如果不打印详细日志，则禁用日志输出
		log.SetOutput(io.Discard)
	} else {
		// 如果需要打印详细日志，则添加文件和行号信息
		log.SetFlags(log.LstdFlags | log.Lshortfile)
	}

	// 读取配置文件
	config, err := loadConfig(*configPath)
	if err != nil {
		log.Fatalf("Failed to load config: %v", err)
	}

	// 初始化SNMP客户端
	snmpClient := initSNMP(config)
	defer snmpClient.Conn.Close()

	// 获取主机名
	hostname, err := os.Hostname()
	if err != nil {
		hostname = "localhost"
		log.Printf("Failed to get hostname, using %s: %v", hostname, err)
	}

	// 初始化告警状态跟踪
	alertStatus := AlertStatus{
		IsInAlert: make(map[string]bool),
	}

	// 启动定时任务
	interval := time.Duration(config.Config.Interval) * time.Second
	ticker := time.NewTicker(interval)
	defer ticker.Stop()

	log.Printf("Starting SNMP trap sender with interval %s", interval)

	for {
		select {
		case <-ticker.C:
			// 发送所有指标的trap消息
			for _, metric := range config.Metrics {
				sendMetricTrap(snmpClient, hostname, metric, &alertStatus)
			}
		}
	}
}

// 加载配置文件
func loadConfig(filename string) (*Config, error) {
	file, err := os.Open(filename)
	if err != nil {
		return nil, fmt.Errorf("failed to open config file: %w", err)
	}
	defer file.Close()

	var config Config
	if err := json.NewDecoder(file).Decode(&config); err != nil {
		return nil, fmt.Errorf("failed to decode config: %w", err)
	}

	return &config, nil
}

// 初始化SNMP客户端
func initSNMP(config *Config) *gosnmp.GoSNMP {
	g := &gosnmp.GoSNMP{
		Target:    config.Config.SNMP.Target,
		Port:      uint16(config.Config.SNMP.Port),
		Community: config.Config.SNMP.Community,
		Version:   gosnmp.Version2c,
		Timeout:   time.Duration(2) * time.Second,
	}

	if err := g.Connect(); err != nil {
		log.Fatalf("Failed to connect to SNMP server: %v", err)
	}

	return g
}

// 发送指标的trap消息
func sendMetricTrap(snmpClient *gosnmp.GoSNMP, hostname string, metric struct {
	Name            string  `json:"name"`
	Type            string  `json:"type"`
	Command         string  `json:"command"`
	WarningOid      string  `json:"warningOid"`
	ClearOid        string  `json:"clearOid"`
	WarningTemplate string  `json:"warningTemplate"`
	ClearTemplate   string  `json:"clearTemplate"`
	Threshold       float64 `json:"threshold"`
}, alertStatus *AlertStatus) {
	// 获取实际指标数据
	metricData, err := getMetricData(hostname, metric.Type, metric.Command, metric.Threshold)
	if err != nil {
		log.Printf("Failed to get metric data for %s: %v", metric.Name, err)
		return
	}

	// 对于数值型指标，使用正常的告警逻辑
	// 检查是否需要发送告警
	if metricData.IsWarning && !alertStatus.IsInAlert[metric.Name] {
		// 超过阈值且不在告警状态，发送告警
		message := fmt.Sprintf(metric.WarningTemplate, metricData.CurrentValue)

		// 准备trap数据
		vars := []gosnmp.SnmpPDU{
			{
				Name:  metric.WarningOid,
				Type:  gosnmp.OctetString,
				Value: []byte(message),
			},
			{
				Name:  ".1.3.6.1.2.1.1.5.0", // sysName
				Type:  gosnmp.OctetString,
				Value: []byte(hostname),
			},
			{
				Name:  ".1.3.6.1.2.1.1.3.0", // sysUpTime
				Type:  gosnmp.TimeTicks,
				Value: uint32(time.Now().Unix() / 100),
			},
		}

		// 创建SNMP trap
		trap := gosnmp.SnmpTrap{
			Variables: vars,
		}

		// 发送trap消息
		log.Printf("Sending WARNING SNMP trap for %s: %s", metric.Name, message)
		if _, err := snmpClient.SendTrap(trap); err != nil {
			log.Printf("Failed to send WARNING SNMP trap for %s: %v", metric.Name, err)
		} else {
			log.Printf("Successfully sent WARNING SNMP trap for %s", metric.Name)
			// 更新告警状态
			alertStatus.IsInAlert[metric.Name] = true
		}
	} else if !metricData.IsWarning && alertStatus.IsInAlert[metric.Name] {
		// 低于阈值且在告警状态，发送一次clear消息
		message := fmt.Sprintf(metric.ClearTemplate, metricData.CurrentValue)

		// 准备trap数据
		vars := []gosnmp.SnmpPDU{
			{
				Name:  metric.ClearOid,
				Type:  gosnmp.OctetString,
				Value: []byte(message),
			},
			{
				Name:  ".1.3.6.1.2.1.1.5.0", // sysName
				Type:  gosnmp.OctetString,
				Value: []byte(hostname),
			},
			{
				Name:  ".1.3.6.1.2.1.1.3.0", // sysUpTime
				Type:  gosnmp.TimeTicks,
				Value: uint32(time.Now().Unix() / 100),
			},
		}

		// 创建SNMP trap
		trap := gosnmp.SnmpTrap{
			Variables: vars,
		}

		// 发送trap消息
		log.Printf("Sending CLEAR SNMP trap for %s: %s", metric.Name, message)
		if _, err := snmpClient.SendTrap(trap); err != nil {
			log.Printf("Failed to send CLEAR SNMP trap for %s: %v", metric.Name, err)
		} else {
			log.Printf("Successfully sent CLEAR SNMP trap for %s", metric.Name)
			// 更新告警状态
			alertStatus.IsInAlert[metric.Name] = false
		}
	}
}

// 获取指标数据 - 支持脚本和命令
func getMetricData(hostname, metricType, command string, threshold float64) (MetricData, error) {
	// 执行命令或脚本
	var outputStr string
	var err error

	switch metricType {
	case "script":
		// 将command作为脚本路径执行
		outputStr, err = executeScriptAsPathRaw(command)
	case "command":
		// 将command作为命令行执行
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
		Hostname:     hostname,
		MetricName:   "",
		CurrentValue: value,
		Threshold:    threshold,
		Unit:         "%",
		IsWarning:    isWarning,
	}, nil
}

// 执行脚本文件并返回原始输出
func executeScriptAsPathRaw(scriptPath string) (string, error) {
	// 检查是否是存在的文件路径
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
