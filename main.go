package main

import (
	"context"
	"encoding/json"
	"flag"
	"fmt"
	"log"
	"os"
	"strings"
	"sync"
	"time"

	goipmi "github.com/bougou/go-ipmi"
)

type IPMIClient struct {
	client    *goipmi.Client
	Host      string
	Username  string
	Password  string
	Interface string
}

type ScheduledOperation struct {
	Time       string `json:"time"`
	Action     string `json:"action"`
	DaysOfWeek []int  `json:"days_of_week"`
}

type ServerConfig struct {
	Name                string               `json:"name"`
	Host                string               `json:"host"`
	Username            string               `json:"username"`
	Password            string               `json:"password"`
	Interface           string               `json:"interface"`
	ScheduledOperations []ScheduledOperation `json:"scheduled_operations"`
}

type Config struct {
	Servers []ServerConfig `json:"servers"`
}

var (
	logFile  *os.File
	logMutex sync.Mutex
	logger   *log.Logger
)

func initLogger() error {
	var err error
	logFile, err = os.OpenFile("log.log", os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0644)
	if err != nil {
		return fmt.Errorf("打开日志文件失败: %w", err)
	}

	logger = log.New(logFile, "", log.LstdFlags)
	return nil
}

func logInfo(format string, args ...interface{}) {
	msg := fmt.Sprintf(format, args...)
	logMutex.Lock()
	defer logMutex.Unlock()
	fmt.Printf("[INFO] %s\n", msg)
	logger.Printf("[INFO] %s", msg)
}

func logError(format string, args ...interface{}) {
	msg := fmt.Sprintf(format, args...)
	logMutex.Lock()
	defer logMutex.Unlock()
	fmt.Fprintf(os.Stderr, "[ERROR] %s\n", msg)
	logger.Printf("[ERROR] %s", msg)
}

func closeLogger() {
	if logFile != nil {
		logFile.Close()
	}
}

func NewIPMIClient(host, username, password, interfaceType string) (*IPMIClient, error) {
	client, err := goipmi.NewClient(host, 623, username, password)
	if err != nil {
		return nil, fmt.Errorf("创建 IPMI 客户端失败: %w", err)
	}

	switch interfaceType {
	case "lan":
		client.WithInterface(goipmi.InterfaceLan)
	case "lanplus", "":
		client.WithInterface(goipmi.InterfaceLanplus)
	default:
		return nil, fmt.Errorf("不支持的接口类型: %s", interfaceType)
	}

	return &IPMIClient{
		client:    client,
		Host:      host,
		Username:  username,
		Password:  password,
		Interface: interfaceType,
	}, nil
}

func (c *IPMIClient) Connect(ctx context.Context) error {
	return c.client.Connect(ctx)
}

func (c *IPMIClient) Close(ctx context.Context) error {
	return c.client.Close(ctx)
}

func (c *IPMIClient) GetPowerStatus() (string, error) {
	ctx := context.Background()
	response, err := c.client.GetChassisStatus(ctx)
	if err != nil {
		return "", fmt.Errorf("获取电源状态失败: %w", err)
	}

	powerIsOn := response.PowerIsOn
	var powerStatus string
	if powerIsOn {
		powerStatus = "on"
	} else {
		powerStatus = "off"
	}

	return powerStatus, nil
}

func (c *IPMIClient) PowerOn() error {
	ctx := context.Background()
	_, err := c.client.ChassisControl(ctx, goipmi.ChassisControlPowerUp)
	if err != nil {
		return fmt.Errorf("开启电源失败: %w", err)
	}
	return nil
}

func (c *IPMIClient) PowerOff() error {
	ctx := context.Background()
	_, err := c.client.ChassisControl(ctx, goipmi.ChassisControlPowerDown) //硬关机
	if err != nil {
		return fmt.Errorf("关闭电源失败: %w", err)
	}
	return nil
}

func (c *IPMIClient) PowerSoftOff() error {
	ctx := context.Background()
	_, err := c.client.ChassisControl(ctx, goipmi.ChassisControlSoftShutdown) //软关机
	if err != nil {
		return fmt.Errorf("关闭电源失败: %w", err)
	}
	return nil
}

func (c *IPMIClient) PowerCycle() error {
	ctx := context.Background()
	_, err := c.client.ChassisControl(ctx, goipmi.ChassisControlPowerCycle)
	if err != nil {
		return fmt.Errorf("重启电源失败: %w", err)
	}
	return nil
}

func (c *IPMIClient) PowerReset() error {
	ctx := context.Background()
	_, err := c.client.ChassisControl(ctx, goipmi.ChassisControlHardReset)
	if err != nil {
		return fmt.Errorf("硬重置电源失败: %w", err)
	}
	return nil
}

func executeAction(client *IPMIClient, action string, serverName string) error {
	switch action {
	case "on":
		logInfo("[%s] 正在执行开机操作...", serverName)
		err := client.PowerOn()
		if err != nil {
			return err
		}
		logInfo("[%s] 开机操作成功", serverName)
	case "off":
		logInfo("[%s] 正在执行关机操作...", serverName)
		err := client.PowerOff()
		if err != nil {
			return err
		}
		logInfo("[%s] 关机操作成功", serverName)
	case "soft":
		logInfo("[%s] 正在执行关机操作...", serverName)
		err := client.PowerSoftOff()
		if err != nil {
			return err
		}
		logInfo("[%s] 关机操作成功", serverName)
	case "cycle":
		logInfo("[%s] 正在执行重启操作...", serverName)
		err := client.PowerCycle()
		if err != nil {
			return err
		}
		logInfo("[%s] 重启操作成功", serverName)
	case "reset":
		logInfo("[%s] 正在执行硬重置操作...", serverName)
		err := client.PowerReset()
		if err != nil {
			return err
		}
		logInfo("[%s] 硬重置操作成功", serverName)
	case "status":
		logInfo("[%s] 正在执行状态查询操作...", serverName)
		status, err := client.GetPowerStatus()
		if err != nil {
			return err
		}
		logInfo("[%s] 状态查询结果: %s", serverName, status)
	default:
		return fmt.Errorf("不支持的操作: %s", action)
	}
	return nil
}

func loadConfig(configPath string) (*Config, error) {
	data, err := os.ReadFile(configPath)
	if err != nil {
		return nil, fmt.Errorf("读取配置文件失败: %w", err)
	}

	var config Config
	err = json.Unmarshal(data, &config)
	if err != nil {
		return nil, fmt.Errorf("解析配置文件失败: %w", err)
	}

	return &config, nil
}

func shouldExecuteOperation(operation ScheduledOperation, currentTime time.Time, lastExecutionTime map[string]time.Time, key string) bool {

	scheduledParts := strings.Split(operation.Time, ":")
	if len(scheduledParts) != 2 {
		return false
	}

	scheduledHour := parseInt(scheduledParts[0])
	scheduledMinute := parseInt(scheduledParts[1])

	currentHour := currentTime.Hour()
	currentMinute := currentTime.Minute()

	if currentHour == scheduledHour && currentMinute == scheduledMinute {
		lastExec, exists := lastExecutionTime[key]
		currentDate := currentTime.Format("2006-01-02")
		if !exists || lastExec.Hour() != currentHour || lastExec.Minute() != currentMinute || lastExec.Format("2006-01-02") != currentDate {
			if isDayMatch(currentTime.Weekday(), operation.DaysOfWeek) {
				return true
			}
		}
	}

	return false
}

func isDayMatch(currentWeekday time.Weekday, allowedDays []int) bool {
	if len(allowedDays) == 0 {
		return true
	}

	currentDay := int(currentWeekday)
	for _, day := range allowedDays {
		if day == currentDay {
			return true
		}
	}

	return false
}

func formatDaysOfWeek(days []int) string {
	if len(days) == 0 {
		return "每天"
	}

	weekdayNames := []string{"周日", "周一", "周二", "周三", "周四", "周五", "周六"}
	var result []string

	for _, day := range days {
		if day >= 0 && day <= 6 {
			result = append(result, weekdayNames[day])
		}
	}

	if len(result) == 0 {
		return "每天"
	}

	return strings.Join(result, ", ")
}

func parseInt(s string) int {
	var i int
	fmt.Sscanf(s, "%d", &i)
	return i
}

func runDaemon(configPath string, checkInterval int) {
	logInfo("========================================")
	logInfo("IPMI 电源管理守护进程启动")
	logInfo("配置文件: %s", configPath)
	logInfo("检查间隔: %d 秒", checkInterval)
	logInfo("========================================\n")

	config, err := loadConfig(configPath)
	if err != nil {
		logError("加载配置失败: %v", err)
		return
	}

	logInfo("成功加载 %d 个服务器配置\n", len(config.Servers))

	for _, server := range config.Servers {
		logInfo("服务器 [%s]: %s (定时任务数: %d)",
			server.Name, server.Host, len(server.ScheduledOperations))
		for _, op := range server.ScheduledOperations {
			daysStr := formatDaysOfWeek(op.DaysOfWeek)
			logInfo("  - %s -> %s (星期: %s)", op.Time, op.Action, daysStr)
		}
	}
	logInfo("")

	ticker := time.NewTicker(time.Duration(checkInterval) * time.Second)
	defer ticker.Stop()

	lastExecutionTime := make(map[string]time.Time)

	for range ticker.C {
		now := time.Now()
		currentTimeStr := now.Format("2006-01-02 15:04:05")

		for _, server := range config.Servers {
			for _, operation := range server.ScheduledOperations {
				key := fmt.Sprintf("%s_%s_%s_%s", server.Name, now.Format("2006-01-02"), operation.Time, operation.Action)

				if shouldExecuteOperation(operation, now, lastExecutionTime, key) {
					weekdayNames := []string{"周日", "周一", "周二", "周三", "周四", "周五", "周六"}
					currentWeekday := weekdayNames[now.Weekday()]

					logInfo("\n[%s] 时间: %s (%s) - 触发定时任务: %s @ %s",
						server.Name, currentTimeStr, currentWeekday, operation.Action, operation.Time)

					client, err := NewIPMIClient(server.Host, server.Username,
						server.Password, server.Interface)
					if err != nil {
						logError("[%s] 创建客户端失败: %v", server.Name, err)
						continue
					}

					ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)

					err = client.Connect(ctx)
					if err != nil {
						logError("[%s] 连接 BMC 失败: %v", server.Name, err)
						cancel()
						continue
					}

					err = executeAction(client, operation.Action, server.Name)
					if err != nil {
						logError("[%s] 执行操作失败: %v", server.Name, err)
					}

					client.Close(ctx)
					cancel()
					lastExecutionTime[key] = now
				}
			}
		}
	}
}

func main() {
	configPath := flag.String("config", "config.json", "配置文件路径")
	interval := flag.Int("interval", 30, "检查定时任务的间隔（秒）")
	showStatus := flag.Bool("status", false, "显示所有服务器当前状态并退出")
	flag.Usage = func() {
		fmt.Fprintf(os.Stderr, "用法: %s [选项]\n\n", os.Args[0])
		fmt.Fprintf(os.Stderr, "IPMI 电源管理守护进程 - 支持定时开关机\n\n")
		fmt.Fprintf(os.Stderr, "选项:\n")
		flag.PrintDefaults()
		fmt.Fprintf(os.Stderr, "\n配置文件格式 (config.json):\n")
		fmt.Fprintf(os.Stderr, "  days_of_week: 星期几执行 (0=周日, 1=周一, ..., 6=周六)\n")
		fmt.Fprintf(os.Stderr, "               空数组或不设置表示每天执行\n\n")
		fmt.Fprintf(os.Stderr, "\n示例:\n")
		fmt.Fprintf(os.Stderr, "  %s                          # 使用默认配置启动守护进程\n", os.Args[0])
		fmt.Fprintf(os.Stderr, "  %s -config myconfig.json     # 使用指定配置文件\n", os.Args[0])
		fmt.Fprintf(os.Stderr, "  %s -interval 60             # 每60秒检查一次\n", os.Args[0])
		fmt.Fprintf(os.Stderr, "  %s -status                  # 显示状态并退出\n", os.Args[0])
	}

	flag.Parse()

	err := initLogger()
	if err != nil {
		fmt.Fprintf(os.Stderr, "初始化日志系统失败: %v\n", err)
		os.Exit(1)
	}
	defer closeLogger()

	if *showStatus {
		config, err := loadConfig(*configPath)
		if err != nil {
			logError("加载配置失败: %v", err)
			os.Exit(1)
		}

		logInfo("显示所有服务器状态:\n")
		for _, server := range config.Servers {
			client, err := NewIPMIClient(server.Host, server.Username,
				server.Password, server.Interface)
			if err != nil {
				logError("[%s] 创建客户端失败: %v", server.Name, err)
				continue
			}

			ctx := context.Background()
			err = client.Connect(ctx)
			if err != nil {
				logError("[%s] 连接失败: %v", server.Name, err)
				continue
			}

			status, err := client.GetPowerStatus()
			if err != nil {
				logError("[%s] 获取状态失败: %v", server.Name, err)
			} else {
				logInfo("[%s] (%s) 电源状态: %s", server.Name, server.Host, status)
			}

			client.Close(ctx)
		}
		return
	}

	runDaemon(*configPath, *interval)
}
