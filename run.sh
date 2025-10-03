#!/bin/bash

# 运行脚本 - 用于启动HTTP数据包捕获工具

# 设置为可执行文件时会显示的消息
set -e  # 如果任何命令失败，立即退出

# 应用程序名称和路径
APP_NAME="packet-capture-tool"
OUTPUT_DIR="output"
APP_PATH="$OUTPUT_DIR/$APP_NAME"

# 检查应用程序是否存在
if [ ! -f "$APP_PATH" ]; then
    # 如果在output目录中找不到，检查当前目录
    if [ ! -f "./$APP_NAME" ]; then
        echo "错误: 未找到可执行文件 '$APP_NAME'。"
        echo "请先运行 './build.sh' 来构建应用程序。"
        exit 1
    else
        echo "警告: 在output目录中未找到应用程序，但在当前目录中找到了。"
        echo "建议运行 './build.sh' 来更新到最新版本并组织到output目录。"
        APP_PATH="./$APP_NAME"
    fi
fi

# 设置执行权限（如果尚未设置）
chmod +x "$APP_PATH"


# 显示启动信息
echo "正在启动HTTP数据包捕获工具..."
echo "----------------------------------"
echo "访问以下地址打开Web界面:"
echo "  - http://localhost:8080"
echo "  - 或者通过服务器的IP地址访问（启动后会显示）"
echo "----------------------------------"

# 启动应用程序 - 在output目录中运行以确保正确加载静态文件
cd "$OUTPUT_DIR" && STATIC_DIR="." ./$APP_NAME

# 如果应用程序意外退出，显示信息
if [ $? -ne 0 ]; then
    echo "----------------------------------"
    echo "应用程序意外退出。"
    echo "可能的原因:"
    echo "1. 需要管理员/root权限来捕获网络数据包"
    echo "2. 端口8080已被占用"
    echo "3. 缺少必要的依赖"
    echo "----------------------------------"
    exit 1
fi