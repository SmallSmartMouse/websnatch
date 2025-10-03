package main

import (
	"sync"
	"time"

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

// 抓包任务结构体
type captureTask struct {
	config    CaptureConfig
	packets   []PacketInfo
	packetsMu sync.Mutex
	handle    *pcap.Handle
	running   bool
}
