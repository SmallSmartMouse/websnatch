package main

import (
	"net/http"
	"path/filepath"

	"github.com/gin-gonic/gin"
)

// SetupRouter 配置所有路由
func SetupRouter() *gin.Engine {
	// 创建gin引擎
	router := gin.Default()
	gin.SetMode(gin.DebugMode)

	// API路由
	setupApiRoutes(router)

	// 静态文件路由
	setupStaticRoutes(router)

	return router
}

// setupApiRoutes 配置API路由
func setupApiRoutes(router *gin.Engine) {
	// 设备列表接口
	router.GET("/devices", ListDevices)

	// 抓包任务相关路由
	router.POST("/capture/start", StartCapture)
	router.POST("/capture/stop", StopCapture)

	router.GET("/capture/results", GetCaptureResults)
	router.GET("/capture/current", GetCurrentRunningTask)

	// WebSocket连接
	router.GET("/ws/capture", WebSocketHandler)

	// 系统信息相关路由
	router.GET("/local-ip", getLocalIpHandler)
}

// setupStaticRoutes 配置静态文件路由
func setupStaticRoutes(router *gin.Engine) {
	// 托管静态文件目录
	router.StaticFS("/static", http.Dir(filepath.Join(RootDir(), "static")))

	// 获取index.html路径
	staticIndexPath := GetStaticIndexPath()

	// 配置根路径路由，确保访问http://127.0.0.1:8081时能自动跳转到index.html
	router.GET("/", func(c *gin.Context) {
		c.File(staticIndexPath)
	})

	// NoRoute处理，确保所有未匹配的路由（不存在的路径）都返回index.html
	router.NoRoute(func(c *gin.Context) {
		// 记录被访问的不存在路径，便于调试
		c.File(staticIndexPath)
	})
}
