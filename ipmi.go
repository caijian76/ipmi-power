package main

import (
	"context"
	"fmt"
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
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	response, err := c.client.GetChassisStatus(ctx)
	if err != nil {
		return "", fmt.Errorf("获取电源状态失败: %w", err)
	}

	if response.PowerIsOn {
		return "on", nil
	}
	return "off", nil
}

func (c *IPMIClient) PowerOn() error {
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	_, err := c.client.ChassisControl(ctx, goipmi.ChassisControlPowerUp)
	if err != nil {
		return fmt.Errorf("开启电源失败: %w", err)
	}
	return nil
}

func (c *IPMIClient) PowerOff() error {
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	_, err := c.client.ChassisControl(ctx, goipmi.ChassisControlPowerDown)
	if err != nil {
		return fmt.Errorf("关闭电源失败: %w", err)
	}
	return nil
}

func (c *IPMIClient) PowerSoftOff() error {
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	_, err := c.client.ChassisControl(ctx, goipmi.ChassisControlSoftShutdown)
	if err != nil {
		return fmt.Errorf("关闭电源失败: %w", err)
	}
	return nil
}

func (c *IPMIClient) PowerCycle() error {
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	_, err := c.client.ChassisControl(ctx, goipmi.ChassisControlPowerCycle)
	if err != nil {
		return fmt.Errorf("重启电源失败: %w", err)
	}
	return nil
}

func (c *IPMIClient) PowerReset() error {
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

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
		if err := client.PowerOn(); err != nil {
			return err
		}
		logInfo("[%s] 开机操作成功", serverName)
	case "off":
		logInfo("[%s] 正在执行关机操作...", serverName)
		if err := client.PowerOff(); err != nil {
			return err
		}
		logInfo("[%s] 关机操作成功", serverName)
	case "soft":
		logInfo("[%s] 正在执行软关机操作...", serverName)
		if err := client.PowerSoftOff(); err != nil {
			return err
		}
		logInfo("[%s] 软关机操作成功", serverName)
	case "cycle":
		logInfo("[%s] 正在执行重启操作...", serverName)
		if err := client.PowerCycle(); err != nil {
			return err
		}
		logInfo("[%s] 重启操作成功", serverName)
	case "reset":
		logInfo("[%s] 正在执行硬重置操作...", serverName)
		if err := client.PowerReset(); err != nil {
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
