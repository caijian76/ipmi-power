package main

import (
	"context"
	"fmt"
	"time"
)

var weekdayNames = []string{"周日", "周一", "周二", "周三", "周四", "周五", "周六"}

func shouldExecuteOperation(operation ScheduledOperation, currentTime time.Time, lastExecutionTime map[string]time.Time, key string) bool {
	scheduledHour, scheduledMinute, ok := parseScheduledTime(operation.Time)
	if !ok {
		return false
	}

	if !isDayMatch(currentTime.Weekday(), operation.DaysOfWeek) {
		return false
	}

	currentMinute := currentTime.Truncate(time.Minute)
	if currentMinute.Hour() != scheduledHour || currentMinute.Minute() != scheduledMinute {
		return false
	}

	lastExec, exists := lastExecutionTime[key]
	if !exists || !lastExec.Equal(currentMinute) {
		return true
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

func runDaemon(configPath string, checkInterval int) {
	if checkInterval <= 0 {
		logError("检查间隔必须大于 0 秒")
		return
	}

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
			logInfo("  - %s -> %s (星期: %s)", op.Time, op.Action, formatDaysOfWeek(op.DaysOfWeek))
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
			if len(server.ScheduledOperations) == 0 {
				continue
			}

			client, err := NewIPMIClient(server.Host, server.Username, server.Password, server.Interface)
			if err != nil {
				logError("[%s] 创建客户端失败: %v", server.Name, err)
				continue
			}

			connectCtx, connectCancel := context.WithTimeout(context.Background(), 30*time.Second)
			err = client.Connect(connectCtx)
			connectCancel()
			if err != nil {
				logError("[%s] 连接 BMC 失败: %v", server.Name, err)
				continue
			}

			for _, operation := range server.ScheduledOperations {
				key := fmt.Sprintf("%s_%s_%s_%s", server.Name, now.Format("2006-01-02"), operation.Time, operation.Action)
				if shouldExecuteOperation(operation, now, lastExecutionTime, key) {
					logInfo("\n[%s] 时间: %s (%s) - 触发定时任务: %s @ %s",
						server.Name, currentTimeStr, weekdayNames[now.Weekday()], operation.Action, operation.Time)

					err = executeAction(client, operation.Action, server.Name)
					if err != nil {
						logError("[%s] 执行操作失败: %v", server.Name, err)
					}

					lastExecutionTime[key] = now
				}
			}

			closeCtx, closeCancel := context.WithTimeout(context.Background(), 10*time.Second)
			if closeErr := client.Close(closeCtx); closeErr != nil {
				logError("[%s] 关闭连接失败: %v", server.Name, closeErr)
			}
			closeCancel()
		}
	}
}
