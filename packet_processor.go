package main

import (
	"abc/a/util"
	"bufio"
	"io"
	"net/http"
	"strings"
	"time"

	"github.com/google/gopacket/tcpassembly"
	"github.com/google/gopacket/tcpassembly/tcpreader"

	"github.com/google/gopacket"
	"github.com/google/gopacket/layers"
)

// 开始抓包过程
func startCapturing(task *captureTask) {
	util.Log.Logger.Info("开始抓包任务，设备: %s, 过滤器: %s", task.config.DeviceName)

	defer func() {
		TaskMutex.Lock()
		task.running = false
		if task.handle != nil {
			task.handle.Close()
		}
		TaskMutex.Unlock()
		util.Log.Logger.Info("抓包任务已停止，设备: %s", task.config.DeviceName)
	}()

	// 使用handle作为数据包源
	packetSource := gopacket.NewPacketSource(task.handle, task.handle.LinkType())

	// 处理每个捕获的数据包
	for packet := range packetSource.Packets() {
		// 检查任务是否已停止
		TaskMutex.Lock()
		running := task.running
		TaskMutex.Unlock()

		if !running {
			util.Log.Logger.Info("抓包任务停止 %v", task.config.DeviceName)
			break
		}

		// 处理数据包
		processPacket(packet, task)
	}
}

type httpStreamFactory struct{}

// httpStream will handle the actual decoding of http requests.
type httpStream struct {
	net, transport gopacket.Flow
	r              tcpreader.ReaderStream
}

func (h *httpStreamFactory) New(net, transport gopacket.Flow) tcpassembly.Stream {
	hstream := &httpStream{
		net:       net,
		transport: transport,
		r:         tcpreader.NewReaderStream(),
	}
	go hstream.run() // Important... we must guarantee that data from the reader stream is read.

	// ReaderStream implements tcpassembly.Stream, so we can return a pointer to it.
	return &hstream.r
}
func (h *httpStream) run() {
	buf := bufio.NewReader(&h.r)
	for {
		req, err := http.ReadRequest(buf)
		if err == io.EOF {
			// We must read until we see an EOF... very important!
			return
		} else if err != nil {
			//log.Println("Error reading stream", h.net, h.transport, ":", err)
		} else {
			bodyBytes := tcpreader.DiscardBytesToEOF(req.Body)
			req.Body.Close()
			//log.Println("Received request from stream", h.net, h.transport, ":", req, "with", bodyBytes, "bytes in request body")
		}
	}
}

// 处理单个数据包
func processPacket(packet gopacket.Packet, task *captureTask) {
	defer func() {
		if r := recover(); r != nil {
			util.Log.Logger.Error("处理数据包时发生恐慌: %v", r)
		}
	}()

	// 检查TCP层信息
	tcpLayer := packet.Layer(layers.LayerTypeTCP)
	if tcpLayer == nil {
		// 非TCP包，跳过
		return
	}

	tcp, _ := tcpLayer.(*layers.TCP)

	// 检查应用层数据
	appLayer := packet.ApplicationLayer()
	if appLayer == nil {
		// 没有应用层数据，跳过
		return
	}

	data := appLayer.Payload()
	dataStr := string(data)
	util.Log.Logger.Info("抓包数据：%v", dataStr)

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
			util.Log.Logger.Debug("数据包不符合协议过滤条件，跳过")
			return
		}
	}
	FlushOlderThan(time.Now().Add(time.Minute * -2))

	// 只处理HTTP请求
	if isHTTPRequest {
		processHTTPRequest(packet, tcp, dataStr, task)
	}
}

// 处理HTTP请求数据包
func processHTTPRequest(packet gopacket.Packet, tcp *layers.TCP, dataStr string, task *captureTask) {
	defer func() {
		if r := recover(); r != nil {
			util.Log.Logger.Error("处理HTTP请求时发生恐慌: %v", r)
		}
	}()

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
	} else {
		// 尝试获取IPv6信息
		ip6Layer := packet.Layer(layers.LayerTypeIPv6)
		if ip6Layer != nil {
			ip6, _ := ip6Layer.(*layers.IPv6)
			packetInfo.SourceIP = ip6.SrcIP.String()
			packetInfo.DestIP = ip6.DstIP.String()
		}
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
		util.Log.Logger.Debug("数据包不符合路径过滤条件，跳过")
		return
	}

	// 应用内容包含过滤
	if task.config.ContainsFilter != "" && !strings.Contains(dataStr, task.config.ContainsFilter) {
		util.Log.Logger.Debug("数据包不符合内容过滤条件，跳过")
		return
	}

	// 保存数据包信息
	task.packetsMu.Lock()
	task.packets = append(task.packets, packetInfo)
	task.packetsMu.Unlock()

	util.Log.Logger.Debug("捕获HTTP请求: %s %s", packetInfo.RequestLine, packetInfo.Host)

	// 通过WebSocket广播新数据包
	go BroadcastNewPacket(packetInfo)
}
