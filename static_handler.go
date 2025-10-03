package main

import (
	"abc/a/util"
	"os"
	"path/filepath"
	"runtime"
)

// GetStaticIndexPath 获取index.html文件的路径
func GetStaticIndexPath() string {
	staticIndexPath := filepath.Join(RootDir(), "static", "index.html")
	if _, err := os.Stat(staticIndexPath); err != nil {
		util.Log.Logger.Warn("未找到static/index.html，将使用根目录下的index.html: %v", err)
		staticIndexPath = filepath.Join(RootDir(), "index.html")
	}
	return staticIndexPath
}

// RootDir 获取当前根目录
func RootDir() string {
	_, f, _, _ := runtime.Caller(0) // 返回当前源文件绝对路径
	return filepath.Dir(f)
}
