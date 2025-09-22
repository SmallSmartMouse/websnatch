# HTTP数据包捕获工具 API文档

## 项目概述

本项目提供了一个基于Gin框架的HTTP数据包捕获工具，通过REST API接口允许用户：
- 查询可用的网卡设备列表
- 开始基于指定网卡设备的数据包捕获任务
- 获取抓包任务的结果
- 停止正在进行的抓包任务

## 技术栈
- Go 1.23.8
- Gin Web框架 v1.9.1
- gopacket 库用于数据包捕获和分析

## API端点列表

| 接口 | 方法 | 路径 | 描述 |
|------|------|------|------|
| 列出网卡设备 | GET | `/devices` | 获取系统中所有可用的网卡设备名称 |
| 开始抓包任务 | POST | `/capture/start` | 基于指定网卡设备开始HTTP数据包捕获 |
| 获取抓包结果 | GET | `/capture/results/:task_id` | 获取指定抓包任务的捕获结果 |
| 停止抓包任务 | POST | `/capture/stop/:task_id` | 停止指定的抓包任务 |

## API接口详细说明

### 1. 列出网卡设备

**请求**
- 方法: GET
- 路径: `/devices`
- 请求参数: 无

**响应**
- 成功 (200 OK):
  ```json
  {
    "devices": [
      "en0",
      "lo0",
      "en1",
      "bridge0"
    ]
  }
  ```
- 失败 (500 Internal Server Error):
  ```json
  {
    "error": "错误信息"
  }
  ```

### 2. 开始抓包任务

**请求**
- 方法: POST
- 路径: `/capture/start`
- 请求体 (JSON):
  ```json
  {
    "device_name": "en0",            // 必需，网卡设备名称
    "protocols": ["http"],            // 可选，过滤的协议列表，当前仅支持"http"
    "path_filter": "/api",            // 可选，URL路径过滤
    "contains_filter": "username",    // 可选，内容包含过滤
    "snapshot_len": 1024,              // 可选，数据包捕获长度，默认1024
    "promiscuous": false,              // 可选，是否开启混杂模式，默认false
    "timeout": 30                      // 可选，超时时间(秒)，默认30
  }
  ```

**响应**
- 成功 (200 OK):
  ```json
  {
    "task_id": "task_1234567890",    // 任务ID，用于后续查询和停止任务
    "message": "抓包任务已启动",
    "config": { /* 返回请求的配置信息 */ }
  }
  ```
- 失败情况:
  - 400 Bad Request (参数错误或设备不存在):
    ```json
    {"error": "错误信息"}
    ```
  - 500 Internal Server Error (设备打开失败等):
    ```json
    {"error": "错误信息"}
    ```

### 3. 获取抓包结果

**请求**
- 方法: GET
- 路径: `/capture/results/:task_id`
- URL参数: `task_id` - 抓包任务ID

**响应**
- 成功 (200 OK):
  ```json
  {
    "task_id": "task_1234567890",
    "running": true,          // 任务是否正在运行
    "count": 15,              // 捕获的数据包数量
    "packets": [              // 捕获的数据包列表
      {
        "timestamp": "2023-06-01T12:00:00Z",
        "source_ip": "192.168.1.100",
        "dest_ip": "203.0.113.1",
        "source_port": 54321,
        "dest_port": 80,
        "protocol": "HTTP",
        "host": "example.com",
        "path": "/api/users",
        "request_line": "GET /api/users HTTP/1.1",
        "content": "(无内容)"
      }
      // 更多数据包...
    ]
  }
  ```
- 失败 (404 Not Found):
  ```json
  {
    "error": "抓包任务不存在"
  }
  ```

### 4. 停止抓包任务

**请求**
- 方法: POST
- 路径: `/capture/stop/:task_id`
- URL参数: `task_id` - 抓包任务ID

**响应**
- 成功 (200 OK):
  ```json
  {
    "task_id": "task_1234567890",
    "message": "抓包任务已停止",
    "captured_packets": 24     // 总共捕获的数据包数量
  }
  ```
- 失败 (404 Not Found):
  ```json
  {
    "error": "抓包任务不存在"
  }
  ```

## 数据模型

### CaptureConfig (抓包配置)

| 字段名 | 类型 | 必填 | 描述 |
|--------|------|------|------|
| device_name | string | 是 | 网卡设备名称 |
| protocols | string[] | 否 | 支持的协议列表，当前仅支持"http" |
| path_filter | string | 否 | URL路径过滤条件 |
| contains_filter | string | 否 | 内容包含过滤条件 |
| snapshot_len | int32 | 否 | 数据包捕获长度，默认1024 |
| promiscuous | bool | 否 | 是否开启混杂模式，默认false |
| timeout | int | 否 | 超时时间(秒)，默认30 |

### PacketInfo (数据包信息)

| 字段名 | 类型 | 描述 |
|--------|------|------|
| timestamp | time.Time | 数据包捕获时间戳 |
| source_ip | string | 源IP地址 |
| dest_ip | string | 目标IP地址 |
| source_port | int | 源端口号 |
| dest_port | int | 目标端口号 |
| protocol | string | 协议类型，当前为"HTTP" |
| host | string | HTTP请求的Host头 |
| path | string | HTTP请求的路径 |
| request_line | string | HTTP请求行 |
| content | string | HTTP请求内容 |

## 使用示例

### 1. 查询可用网卡设备

```bash
curl http://localhost:8080/devices
```

### 2. 开始抓包任务

```bash
curl -X POST http://localhost:8080/capture/start \
  -H "Content-Type: application/json" \
  -d '{"device_name": "en0", "protocols": ["http"], "path_filter": "/api"}'
```

### 3. 获取抓包结果

```bash
curl http://localhost:8080/capture/results/task_1234567890
```

### 4. 停止抓包任务

```bash
curl -X POST http://localhost:8080/capture/stop/task_1234567890
```

## 运行说明

1. 确保已安装Go环境
2. 安装依赖:
   ```bash
   go mod tidy
   ```
3. 运行程序:
   ```bash
   go run main.go
   ```
4. API服务器将在 http://localhost:8080 启动

## 注意事项

- 运行程序需要足够的权限来捕获网络数据包
- 在macOS上可能需要使用sudo运行
- 在Windows上可能需要以管理员身份运行
- 当前版本仅支持HTTP协议的数据包捕获和分析