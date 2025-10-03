package main

import (
	"abc/a/util"
	"encoding/json"
	"net/http"
	"sync"

	"github.com/gin-gonic/gin"
	"github.com/gorilla/websocket"
)

var upgrader = websocket.Upgrader{
	ReadBufferSize:  1024,
	WriteBufferSize: 1024,
	CheckOrigin: func(r *http.Request) bool {
		// 允许所有来源，在生产环境中应该根据需要限制
		return true
	},
}

// WebSocket连接管理器
var wsConnections = struct {
	connections map[string]*websocket.Conn
	mutex       sync.Mutex
}{}

// 初始化WebSocket连接管理器
func init() {
	wsConnections.connections = make(map[string]*websocket.Conn)
}

// WebSocketHandler 处理WebSocket连接
func WebSocketHandler(c *gin.Context) {
	// 将HTTP连接升级为WebSocket连接
	conn, err := upgrader.Upgrade(c.Writer, c.Request, nil)
	if err != nil {
		util.Log.Logger.Error("升级WebSocket连接失败: %v, IP: %s", err, c.ClientIP())
		return
	}

	defer conn.Close()

	// 生成连接ID
	connID := c.ClientIP() + ":" + c.Request.RemoteAddr

	// 添加连接到管理器
	wsConnections.mutex.Lock()
	wsConnections.connections[connID] = conn
	wsConnections.mutex.Unlock()

	util.Log.Logger.Info("WebSocket连接已建立: %s, IP: %s", connID, c.ClientIP())

	// 发送当前任务状态
	sendCurrentTaskStatus(conn)

	// 处理从客户端接收的消息
	for {
		// 读取消息类型，不需要实际处理消息内容
		_, _, err := conn.ReadMessage()
		if err != nil {
			util.Log.Logger.Error("WebSocket连接读取失败: %v, 连接ID: %s, IP: %s", err, connID, c.ClientIP())
			break
		}
	}

	// 从管理器中移除连接
	wsConnections.mutex.Lock()
	delete(wsConnections.connections, connID)
	wsConnections.mutex.Unlock()

	util.Log.Logger.Info("WebSocket连接已关闭: %s, IP: %s", connID, c.ClientIP())
}

// 发送当前任务状态
func sendCurrentTaskStatus(conn *websocket.Conn) {
	TaskMutex.Lock()
	defer TaskMutex.Unlock()

	response := gin.H{"type": "task_status"}

	if CurrentTask != nil && CurrentTask.running {
		response["running"] = true
		response["config"] = CurrentTask.config
	} else {
		response["running"] = false
	}

	jsonData, err := json.Marshal(response)
	if err != nil {
		util.Log.Logger.Error("JSON序列化失败: %v", err)
		return
	}

	err = conn.WriteMessage(websocket.TextMessage, jsonData)
	if err != nil {
		util.Log.Logger.Error("WebSocket消息发送失败: %v", err)
	}
}

// 向所有连接的客户端广播新的数据包
func BroadcastNewPacket(packet PacketInfo) {
	wsConnections.mutex.Lock()
	defer wsConnections.mutex.Unlock()

	response := gin.H{
		"type":   "new_packet",
		"packet": packet,
	}

	jsonData, err := json.Marshal(response)
	if err != nil {
		util.Log.Logger.Error("JSON序列化失败: %v", err)
		return
	}

	// 向所有连接发送消息
	for connID, conn := range wsConnections.connections {
		err := conn.WriteMessage(websocket.TextMessage, jsonData)
		if err != nil {
			util.Log.Logger.Error("WebSocket消息发送失败 (%s): %v", connID, err)
			// 移除不可用的连接
			conn.Close()
			delete(wsConnections.connections, connID)
		}
	}
}

// 向所有连接的客户端广播任务状态更新
func BroadcastTaskStatus(update gin.H) {
	wsConnections.mutex.Lock()
	defer wsConnections.mutex.Unlock()

	update["type"] = "task_update"

	jsonData, err := json.Marshal(update)
	if err != nil {
		util.Log.Logger.Error("JSON序列化失败: %v", err)
		return
	}

	// 向所有连接发送消息
	for connID, conn := range wsConnections.connections {
		err := conn.WriteMessage(websocket.TextMessage, jsonData)
		if err != nil {
			util.Log.Logger.Error("WebSocket消息发送失败 (%s): %v", connID, err)
			// 移除不可用的连接
			conn.Close()
			delete(wsConnections.connections, connID)
		}
	}
}
