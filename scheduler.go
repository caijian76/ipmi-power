package main

import (
	"context"
	"fmt"
	"strconv"
	"strings"
	"time"

	"github.com/robfig/cron/v3"
)

var weekdayNames = []string{"周日", "周一", "周二", "周三", "周四", "周五", "周六"}

func buildCronSpec(operation ScheduledOperation) (string, error) {
	scheduledHour, scheduledMinute, ok := parseScheduledTime(operation.Time)
	if !ok {
		return "", fmt.Errorf("无效的定时格式: %s", operation.Time)
	}

	if len(operation.DaysOfWeek) == 0 {
		return fmt.Sprintf("%d %d * * *", scheduledMinute, scheduledHour), nil
	}

	parts := make([]string, 0, len(operation.DaysOfWeek))
	for _, day := range operation.DaysOfWeek {
		parts = append(parts, strconv.Itoa(day))
	}

	return fmt.Sprintf("%d %d * * %s", scheduledMinute, scheduledHour, strings.Join(parts, ",")), nil
}

func runScheduledOperation(server ServerConfig, operation ScheduledOperation) {
	now := time.Now()
	currentTimeStr := now.Format("2006-01-02 15:04:05")

	logInfo("\n[%s] 时间: %s (%s) - 触发定时任务: %s @ %s",
		server.Name, currentTimeStr, weekdayNames[now.Weekday()], operation.Action, operation.Time)

	client, err := NewIPMIClient(server.Host, server.Username, server.Password, server.Interface)
	if err != nil {
		logError("[%s] 创建客户端失败: %v", server.Name, err)
		return
	}

	connectCtx, connectCancel := context.WithTimeout(context.Background(), 30*time.Second)
	err = client.Connect(connectCtx)
	connectCancel()
	if err != nil {
		logError("[%s] 连接 BMC 失败: %v", server.Name, err)
		return
	}

	err = executeAction(client, operation.Action, server.Name)
	if err != nil {
		logError("[%s] 执行操作失败: %v", server.Name, err)
	}

	closeCtx, closeCancel := context.WithTimeout(context.Background(), 10*time.Second)
	if closeErr := client.Close(closeCtx); closeErr != nil {
		logError("[%s] 关闭连接失败: %v", server.Name, closeErr)
	}
	closeCancel()
}

func runDaemon(configPath string) {
	logInfo("========================================")
	logInfo("IPMI 电源管理守护进程启动")
	logInfo("配置文件: %s", configPath)
	logInfo("调度器: cron/v3")
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
			logInfo("  - %s -> %s (星期: %s)", op.Time, op.Action, formatDaysOfWeek(op.DaysOfWeek))
		}
	}
	logInfo("")

	scheduler := cron.New(cron.WithLocation(time.Local))
	for _, server := range config.Servers {
		for _, operation := range server.ScheduledOperations {
			server := server
			operation := operation
			spec, err := buildCronSpec(operation)
			if err != nil {
				logError("[%s] 生成 cron 表达式失败: %v", server.Name, err)
				continue
			}

			_, err = scheduler.AddFunc(spec, func() {
				runScheduledOperation(server, operation)
			})
			if err != nil {
				logError("[%s] 注册定时任务失败: %v", server.Name, err)
				continue
			}

			logInfo("[%s] 已注册定时任务: %s -> %s (cron: %s)", server.Name, operation.Time, operation.Action, spec)
		}
	}

	scheduler.Start()
	logInfo("定时任务已启动，等待触发...\n")
	select {}
}
