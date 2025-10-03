package main

import (
	"abc/a/util"
	"net/http"

	"github.com/gin-gonic/gin"
)

// getLocalIpHandler 获取本机IP接口处理函数
func getLocalIpHandler(c *gin.Context) {
	c.JSON(http.StatusOK, gin.H{"ip": util.GetLocalIP()})
}

// GetCurrentRunningTask 获取当前运行的任务信息处理函数
func GetCurrentRunningTask(c *gin.Context) {
	TaskMutex.Lock()
	task := CurrentTask
	TaskMutex.Unlock()

	if task == nil || !task.running {
		util.Log.Logger.Info("当前没有运行中的抓包任务，IP: %s", c.ClientIP())
		c.JSON(http.StatusOK, gin.H{"running": false})
		return
	}

	util.Log.Logger.Info("获取运行中任务信息成功，IP: %s", c.ClientIP())
	c.JSON(http.StatusOK, gin.H{
		"running": true,
		"config":  task.config,
	})
}
