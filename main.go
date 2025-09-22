// HTTP数据包捕获工具 - REST API版
package main

import (
	"fmt"
	"net/http"
	"strings"
	"sync"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/google/gopacket"
	"github.com/google/gopacket/layers"
	"github.com/google/gopacket/pcap"
)

// 抓包任务配置
type CaptureConfig struct {
	DeviceName     string   `json:"device_name" binding:"required"`
	Protocols      []string `json:"protocols"`       // 支持的协议列表，如["http"]
	PathFilter     string   `json:"path_filter"`     // URL路径过滤
	ContainsFilter string   `json:"contains_filter"` // 内容包含过滤
	SnapshotLen    int32    `json:"snapshot_len" default:"1024"`
	Promiscuous    bool     `json:"promiscuous" default:"false"`
	Timeout        int      `json:"timeout" default:"30"` // 秒
}

// 抓包结果
type PacketInfo struct {
	Timestamp   time.Time `json:"timestamp"`
	SourceIP    string    `json:"source_ip"`
	DestIP      string    `json:"dest_ip"`
	SourcePort  int       `json:"source_port"`
	DestPort    int       `json:"dest_port"`
	Protocol    string    `json:"protocol"`
	Host        string    `json:"host"`
	Path        string    `json:"path"`
	RequestLine string    `json:"request_line"`
	Content     string    `json:"content"`
}

// 抓包任务管理器
var (
	captureTasks = make(map[string]*captureTask)
	tasksMutex   = &sync.Mutex{}
)

// 抓包任务结构体
type captureTask struct {
	config    CaptureConfig
	packets   []PacketInfo
	packetsMu sync.Mutex
	handle    *pcap.Handle
	running   bool
}

func main() {
	// 创建gin引擎
	r := gin.Default()

	// API路由
	r.GET("/devices", listDevices)
	r.POST("/capture/start", startCapture)
	r.GET("/capture/results/:task_id", getCaptureResults)
	r.POST("/capture/stop/:task_id", stopCapture)

	// 启动服务器
	r.Run(":8080")
}

// 列出所有网卡设备
func listDevices(c *gin.Context) {
	devices, err := pcap.FindAllDevs()
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	// 提取设备名称列表
	var deviceNames []string
	for _, device := range devices {
		deviceNames = append(deviceNames, device.Name)
	}

	c.JSON(http.StatusOK, gin.H{"devices": deviceNames})
}

// 开始抓包任务
func startCapture(c *gin.Context) {
	var config CaptureConfig
	if err := c.ShouldBindJSON(&config); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	// 检查设备是否存在
	devices, err := pcap.FindAllDevs()
	if err != nil {
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
		c.JSON(http.StatusBadRequest, gin.H{"error": "指定的网卡设备不存在"})
		return
	}

	// 创建任务ID
	taskID := fmt.Sprintf("task_%d", time.Now().UnixNano())

	// 打开网络设备
	timeout := time.Duration(config.Timeout) * time.Second
	handle, err := pcap.OpenLive(config.DeviceName, config.SnapshotLen, config.Promiscuous, timeout)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "无法打开网卡设备: " + err.Error()})
		return
	}

	// 创建并保存抓包任务
	task := &captureTask{
		config:  config,
		packets: make([]PacketInfo, 0),
		handle:  handle,
		running: true,
	}

	tasksMutex.Lock()
	captureTasks[taskID] = task
	tasksMutex.Unlock()

	// 启动异步抓包
	go startCapturing(taskID, task)

	c.JSON(http.StatusOK, gin.H{
		"task_id": taskID,
		"message": "抓包任务已启动",
		"config":  config,
	})
}

// 获取抓包结果
func getCaptureResults(c *gin.Context) {
	taskID := c.Param("task_id")

	tasksMutex.Lock()
	task, exists := captureTasks[taskID]
	tasksMutex.Unlock()

	if !exists {
		c.JSON(http.StatusNotFound, gin.H{"error": "抓包任务不存在"})
		return
	}

	task.packetsMu.Lock()
	packets := make([]PacketInfo, len(task.packets))
	copy(packets, task.packets)
	task.packetsMu.Unlock()

	c.JSON(http.StatusOK, gin.H{
		"task_id": taskID,
		"running": task.running,
		"count":   len(packets),
		"packets": packets,
	})
}

// 停止抓包任务
func stopCapture(c *gin.Context) {
	taskID := c.Param("task_id")

	tasksMutex.Lock()
	task, exists := captureTasks[taskID]
	tasksMutex.Unlock()

	if !exists {
		c.JSON(http.StatusNotFound, gin.H{"error": "抓包任务不存在"})
		return
	}

	task.running = false
	if task.handle != nil {
		task.handle.Close()
	}

	c.JSON(http.StatusOK, gin.H{
		"task_id":          taskID,
		"message":          "抓包任务已停止",
		"captured_packets": len(task.packets),
	})
}

// 开始抓包过程
func startCapturing(taskID string, task *captureTask) {
	defer func() {
		tasksMutex.Lock()
		task.running = false
		if task.handle != nil {
			task.handle.Close()
		}
		tasksMutex.Unlock()
	}()

	// 使用handle作为数据包源
	packetSource := gopacket.NewPacketSource(task.handle, task.handle.LinkType())

	// 处理每个捕获的数据包
	for packet := range packetSource.Packets() {
		// 检查任务是否已停止
		tasksMutex.Lock()
		running := task.running
		tasksMutex.Unlock()

		if !running {
			break
		}

		// 处理数据包
		processPacket(packet, task)
	}
}

// 处理单个数据包
func processPacket(packet gopacket.Packet, task *captureTask) {
	// 检查TCP层信息
	tcpLayer := packet.Layer(layers.LayerTypeTCP)
	if tcpLayer == nil {
		return
	}

	tcp, _ := tcpLayer.(*layers.TCP)

	// 检查应用层数据
	appLayer := packet.ApplicationLayer()
	if appLayer == nil {
		return
	}

	data := appLayer.Payload()
	dataStr := string(data)

	// 判断是否为HTTP请求
	isHTTPRequest := strings.HasPrefix(strings.ToUpper(dataStr), "GET ") ||
		strings.HasPrefix(strings.ToUpper(dataStr), "POST ") ||
		strings.HasPrefix(strings.ToUpper(dataStr), "PUT ") ||
		strings.HasPrefix(strings.ToUpper(dataStr), "DELETE ") ||
		strings.HasPrefix(strings.ToUpper(dataStr), "HEAD ")

	// 检查协议过滤
	if len(task.config.Protocols) > 0 {
		protocolMatch := false
		for _, proto := range task.config.Protocols {
			if strings.EqualFold(proto, "http") && isHTTPRequest {
				protocolMatch = true
				break
			}
		}
		if !protocolMatch {
			return
		}
	}

	// 只处理HTTP请求
	if isHTTPRequest {
		processHTTPRequest(packet, tcp, dataStr, task)
	}
}

// 处理HTTP请求数据包
func processHTTPRequest(packet gopacket.Packet, tcp *layers.TCP, dataStr string, task *captureTask) {
	// 解析数据包信息
	packetInfo := PacketInfo{
		Timestamp:  time.Now(),
		SourcePort: int(tcp.SrcPort),
		DestPort:   int(tcp.DstPort),
		Protocol:   "HTTP",
	}

	// 获取IP层信息
	ipLayer := packet.Layer(layers.LayerTypeIPv4)
	if ipLayer != nil {
		ip, _ := ipLayer.(*layers.IPv4)
		packetInfo.SourceIP = ip.SrcIP.String()
		packetInfo.DestIP = ip.DstIP.String()
	}

	// 解析HTTP信息
	lines := strings.Split(dataStr, "\r\n")

	// 提取请求行和路径
	if len(lines) > 0 {
		packetInfo.RequestLine = lines[0]
		// 从请求行提取路径
		parts := strings.Split(lines[0], " ")
		if len(parts) >= 2 {
			packetInfo.Path = parts[1]
		}
	}

	// 提取域名
	for _, line := range lines {
		if strings.HasPrefix(strings.ToLower(line), "host: ") {
			packetInfo.Host = strings.TrimSpace(strings.TrimPrefix(strings.ToLower(line), "host: "))
			break
		}
	}

	// 提取请求内容
	bodyStartIndex := -1
	for i, line := range lines {
		if line == "" && i < len(lines)-1 {
			bodyStartIndex = i + 1
			break
		}
	}

	if bodyStartIndex > 0 && bodyStartIndex < len(lines) {
		body := strings.Join(lines[bodyStartIndex:], "\n")
		packetInfo.Content = body
	} else {
		packetInfo.Content = "(无内容)"
	}

	// 应用路径过滤
	if task.config.PathFilter != "" && !strings.Contains(packetInfo.Path, task.config.PathFilter) {
		return
	}

	// 应用内容包含过滤
	if task.config.ContainsFilter != "" && !strings.Contains(dataStr, task.config.ContainsFilter) {
		return
	}

	// 保存数据包信息
	task.packetsMu.Lock()
	task.packets = append(task.packets, packetInfo)
	task.packetsMu.Unlock()
}
