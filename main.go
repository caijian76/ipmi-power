package main

import (
	"context"
	"flag"
	"fmt"
	"os"
	"time"
)

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

	if err := initLogger(); err != nil {
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
			client, err := NewIPMIClient(server.Host, server.Username, server.Password, server.Interface)
			if err != nil {
				logError("[%s] 创建客户端失败: %v", server.Name, err)
				continue
			}

			ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
			err = client.Connect(ctx)
			if err != nil {
				logError("[%s] 连接失败: %v", server.Name, err)
				cancel()
				continue
			}

			status, err := client.GetPowerStatus()
			if err != nil {
				logError("[%s] 获取状态失败: %v", server.Name, err)
			} else {
				logInfo("[%s] (%s) 电源状态: %s", server.Name, server.Host, status)
			}

			closeCtx, closeCancel := context.WithTimeout(context.Background(), 10*time.Second)
			if closeErr := client.Close(closeCtx); closeErr != nil {
				logError("[%s] 关闭连接失败: %v", server.Name, closeErr)
			}
			closeCancel()
			cancel()
		}
		return
	}

	runDaemon(*configPath, *interval)
}
