package main

import (
	"abc/a/util"
	"net/http"
	"strings"
	"sync"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/google/gopacket/pcap"
)

// 单任务模式变量
var (
	// CurrentTask 当前正在运行的抓包任务
	CurrentTask *captureTask
	// TaskMutex 任务访问互斥锁
	TaskMutex = &sync.Mutex{}
)

// ListDevices 列出所有网卡设备
func ListDevices(c *gin.Context) {
	devices, err := pcap.FindAllDevs()
	if err != nil {
		util.Log.Logger.Error("获取设备列表失败: %v, IP: %s", err, c.ClientIP())
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	util.Log.Logger.Info("获取设备列表成功，设备数量: %d, IP: %s", len(devices), c.ClientIP())

	// 提取设备名称列表
	var deviceNames []string
	for _, device := range devices {
		if !strings.Contains(device.Name, "utun") {
			deviceNames = append(deviceNames, device.Name)
		}

	}

	c.JSON(http.StatusOK, gin.H{"devices": deviceNames})
}

// StartCapture 开始抓包任务
func StartCapture(c *gin.Context) {
	var config CaptureConfig
	if err := c.ShouldBindJSON(&config); err != nil {
		util.Log.Logger.Error("参数绑定失败: %v, IP: %s", err, c.ClientIP())
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	util.Log.Logger.Info("接收启动抓包请求，设备: %s, 过滤器: %s, IP: %s", config.DeviceName, c.ClientIP())
	if !test(c.ClientIP()) {
		util.Log.Logger.Error("未开启网络采集权限，IP: %s", c.ClientIP())
		c.JSON(http.StatusInternalServerError, gin.H{"error": "未开启网络采集权限"})
		return
	}

	// 检查是否已经有任务在运行
	TaskMutex.Lock()
	if CurrentTask != nil && CurrentTask.running {
		TaskMutex.Unlock()
		util.Log.Logger.Warn("已有抓包任务在运行，拒绝新的抓包请求，IP: %s", c.ClientIP())
		c.JSON(http.StatusConflict, gin.H{"error": "已有抓包任务在运行，请先停止当前任务"})
		return
	}
	TaskMutex.Unlock()

	// 检查设备是否存在
	devices, err := pcap.FindAllDevs()
	if err != nil {
		util.Log.Logger.Error("无法获取设备列表: %v, IP: %s", err, c.ClientIP())
		c.JSON(http.StatusInternalServerError, gin.H{"error": "无法获取设备列表: " + err.Error()})
		return
	}

	deviceExists := false
	for _, device := range devices {
		if device.Name == config.DeviceName {
			deviceExists = true
			break
		}
	}

	if !deviceExists {
		util.Log.Logger.Error("指定的网卡设备不存在: %s, IP: %s", config.DeviceName, c.ClientIP())
		c.JSON(http.StatusBadRequest, gin.H{"error": "指定的网卡设备不存在"})
		return
	}

	// 检查是否有网络监控权限
	if !hasNetworkCapturePermission() {
		// 检查是否是本机请求（127.0.0.1、localhost 或本机IP）
		requestIP := c.ClientIP()
		isLocalRequest := requestIP == "127.0.0.1" || requestIP == "localhost" || requestIP == util.GetLocalIP()

		// 在后台线程提示用户，避免阻塞API响应
		go func() {
			promptUserForPermission(isLocalRequest)
		}()

		util.Log.Logger.Warn("需要网络监控权限才能抓包，请求IP: %s", c.ClientIP())
		c.JSON(http.StatusUnauthorized, gin.H{
			"error": "需要网络监控权限才能抓包，请在系统设置中授予权限",
			"hint":  "在macOS上，请前往系统设置 > 隐私与安全性 > 网络监控，添加并启用本程序",
		})
		return
	}

	// 打开网络设备
	timeout := time.Duration(config.Timeout) * time.Second
	handle, err := pcap.OpenLive(config.DeviceName, config.SnapshotLen, config.Promiscuous, timeout)
	if err != nil {
		util.Log.Logger.Error("无法打开网卡设备: %v, 设备: %s, IP: %s", err, config.DeviceName, c.ClientIP())
		c.JSON(http.StatusInternalServerError, gin.H{"error": "无法打开网卡设备: " + err.Error()})
		return
	}
	util.Log.Logger.Info("成功打开网卡设备: %s, IP: %s", config.DeviceName, c.ClientIP())

	// 创建并保存抓包任务
	task := &captureTask{
		config:  config,
		packets: make([]PacketInfo, 0),
		handle:  handle,
		running: true,
	}

	TaskMutex.Lock()
	CurrentTask = task
	TaskMutex.Unlock()

	// 启动异步抓包
	go startCapturing(task)

	// 广播任务启动状态
	go BroadcastTaskStatus(gin.H{
		"running": true,
		"message": "抓包任务已启动",
		"config":  config,
	})

	util.Log.Logger.Info("抓包任务已成功启动，设备: %s, 过滤器: %s, IP: %s", config.DeviceName, c.ClientIP())
	c.JSON(http.StatusOK, gin.H{
		"message": "抓包任务已启动",
		"config":  config,
	})
}

// GetCaptureResults 获取抓包结果
func GetCaptureResults(c *gin.Context) {
	TaskMutex.Lock()
	if CurrentTask == nil {
		TaskMutex.Unlock()
		util.Log.Logger.Error("没有正在运行的抓包任务，IP: %s", c.ClientIP())
		c.JSON(http.StatusNotFound, gin.H{"error": "没有正在运行的抓包任务"})
		return
	}
	task := CurrentTask
	TaskMutex.Unlock()

	task.packetsMu.Lock()
	packets := make([]PacketInfo, len(task.packets))
	copy(packets, task.packets)
	task.packetsMu.Unlock()

	c.JSON(http.StatusOK, gin.H{
		"running": task.running,
		"count":   len(packets),
		"packets": packets,
	})
}

// StopCapture 停止抓包任务
func StopCapture(c *gin.Context) {
	TaskMutex.Lock()
	if CurrentTask == nil {
		TaskMutex.Unlock()
		c.JSON(http.StatusNotFound, gin.H{"error": "没有正在运行的抓包任务"})
		return
	}
	task := CurrentTask

	// 停止任务
	task.running = false
	if task.handle != nil {
		task.handle.Close()
	}

	capturedPackets := len(task.packets)

	// 清除当前运行任务
	// 注意：已经在函数开始时获取了锁，这里不需要再次获取
	CurrentTask = nil
	TaskMutex.Unlock() // 只解锁一次

	// 广播任务停止状态
	go BroadcastTaskStatus(gin.H{
		"running":          false,
		"message":          "抓包任务已停止",
		"captured_packets": capturedPackets,
	})

	util.Log.Logger.Info("抓包任务已停止，共捕获 %d 个数据包，IP: %s", capturedPackets, c.ClientIP())
	c.JSON(http.StatusOK, gin.H{
		"message":          "抓包任务已停止",
		"captured_packets": capturedPackets,
	})
}
