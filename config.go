package main

import (
	"encoding/json"
	"fmt"
	"os"
	"strings"
	"time"
)

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

func loadConfig(configPath string) (*Config, error) {
	data, err := os.ReadFile(configPath)
	if err != nil {
		return nil, fmt.Errorf("读取配置文件失败: %w", err)
	}

	var config Config
	if err := json.Unmarshal(data, &config); err != nil {
		return nil, fmt.Errorf("解析配置文件失败: %w", err)
	}

	if err := validateConfig(&config); err != nil {
		return nil, fmt.Errorf("配置校验失败: %w", err)
	}

	return &config, nil
}

func validateConfig(config *Config) error {
	if len(config.Servers) == 0 {
		return fmt.Errorf("至少需要配置一个服务器")
	}

	for i, server := range config.Servers {
		if strings.TrimSpace(server.Name) == "" {
			return fmt.Errorf("servers[%d].name 不能为空", i)
		}
		if strings.TrimSpace(server.Host) == "" {
			return fmt.Errorf("servers[%d].host 不能为空", i)
		}
		if strings.TrimSpace(server.Username) == "" {
			return fmt.Errorf("servers[%d].username 不能为空", i)
		}
		if strings.TrimSpace(server.Password) == "" {
			return fmt.Errorf("servers[%d].password 不能为空", i)
		}

		switch strings.ToLower(server.Interface) {
		case "", "lan", "lanplus":
		default:
			return fmt.Errorf("servers[%d].interface 只支持 lan 或 lanplus", i)
		}

		for j, op := range server.ScheduledOperations {
			if strings.TrimSpace(op.Time) == "" {
				return fmt.Errorf("servers[%d].scheduled_operations[%d].time 不能为空", i, j)
			}
			if _, _, ok := parseScheduledTime(op.Time); !ok {
				return fmt.Errorf("servers[%d].scheduled_operations[%d].time 格式必须为 HH:MM", i, j)
			}

			switch op.Action {
			case "on", "off", "soft", "cycle", "reset", "status":
			default:
				return fmt.Errorf("servers[%d].scheduled_operations[%d].action 非法: %s", i, j, op.Action)
			}

			for k, day := range op.DaysOfWeek {
				if day < 0 || day > 6 {
					return fmt.Errorf("servers[%d].scheduled_operations[%d].days_of_week[%d] 超出范围: %d", i, j, k, day)
				}
			}
		}
	}

	return nil
}

func formatDaysOfWeek(days []int) string {
	if len(days) == 0 {
		return "每天"
	}

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

func parseScheduledTime(timeStr string) (int, int, bool) {
	t, err := time.Parse("15:04", timeStr)
	if err != nil {
		return 0, 0, false
	}

	return t.Hour(), t.Minute(), true
}
