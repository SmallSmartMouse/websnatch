// HTTP数据包捕获工具 - REST API版
package main

import (
	"abc/a/util"
)

func main() {
	// 初始化日志系统
	util.InitLogger()

	// 设置路由
	r := SetupRouter()

	// 获取本机IP
	localIP := util.GetLocalIP()
	util.Log.Logger.Info("服务器启动在 http://%s:8081", localIP)

	// 启动服务器并记录可能的错误
	util.Log.Logger.Info("服务器开始监听请求...")
	if err := r.Run(":8081"); err != nil {
		util.Log.Logger.Fatal("服务器启动失败: %v", err)
	}
}
