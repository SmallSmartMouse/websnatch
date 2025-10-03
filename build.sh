#!/bin/bash

# 构建脚本 - 用于编译HTTP数据包捕获工具

# 设置为可执行文件时会显示的消息
set -e  # 如果任何命令失败，立即退出

# 检查是否安装了Go
if ! command -v go &> /dev/null
then
    echo "错误: 未找到Go编译器，请先安装Go。"
    exit 1
fi

# 打印当前Go版本信息
echo "使用Go版本: $(go version)"

# 安装依赖
echo "正在安装依赖..."
go mod tidy

# 构建项目
echo "正在构建项目..."
go build -o packet-capture-tool

# 检查是否为Linux系统
if [ "$(uname -s)" = "Linux" ]; then
    echo "正在设置网络权限..."
    sudo setcap cap_net_raw+ep packet-capture-tool
    # 验证
getcap packet-capture-tool
else
    ls -l /dev/bpf*

    # 若显示 crw------- root wheel → 未配置
    # 若显示 crw-rw---- root access_bpf → 已配置
    # 获取当前用户的用户组
    CURRENT_GROUP=$(id -g -n)
    
    # 临时手动修复（重启失效）
    sudo chown root:$CURRENT_GROUP /dev/bpf*
    sudo chmod 660 /dev/bpf*
    echo "非Linux系统，跳过设置网络权限..."
fi

# 检查构建是否成功
if [ -f "./packet-capture-tool" ]; then
    # 创建output目录
    echo "正在创建output目录..."
    mkdir -p output
    
    # 复制可执行文件到output目录
    echo "正在复制可执行文件到output目录..."
    mv ./packet-capture-tool ./output/
    chmod +x ./run.sh
    cp ./run.sh ./output/

    # 复制static目录到output目录
    echo "正在复制静态资源文件到output目录..."
    if [ -d "./static" ]; then
        cp -r ./static ./output/
    else
        echo "警告: 未找到static目录"
    fi
    
    echo "构建成功! 所有文件已复制到output目录。"
    echo "可执行文件位置: ./output/packet-capture-tool"
    echo "使用 './output/packet-capture-tool' 启动应用程序"
else
    echo "构建失败!"
    exit 1
fi