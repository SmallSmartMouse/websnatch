package util

import (
	"net"
)

// GetLocalIP 获取本机IP地址
func GetLocalIP() string {
	addrs, err := net.InterfaceAddrs()
	if err != nil {
		Log.Logger.Error("获取本机网卡地址失败: %v", err)
		return "127.0.0.1"
	}
	for _, address := range addrs {
		// 检查ip地址判断是否回环地址
		if ipnet, ok := address.(*net.IPNet); ok && !ipnet.IP.IsLoopback() {
			if ipnet.IP.To4() != nil {
				return ipnet.IP.String()
			}
		}
	}
	Log.Logger.Debug("未能找到有效的非回环IPv4地址，返回127.0.0.1")
	return "127.0.0.1"
}
