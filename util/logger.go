package util

import (
	"fmt"
	"os"
	"path/filepath"
	"time"
)

// LogLevel 定义日志级别
type LogLevel int

const (
	DEBUG LogLevel = iota
	INFO
	WARN
	ERROR
	FATAL
)

// Logger 结构体用于日志记录
type Logger struct {
	Level         LogLevel
	ConsoleOutput bool
	FileOutput    bool
	LogFilePath   string
}

// NewLogger 创建一个新的Logger实例
func NewLogger(level LogLevel, consoleOutput bool, fileOutput bool, logFilePath string) *Logger {
	// 如果需要文件输出但文件路径为空，使用默认路径
	if fileOutput && logFilePath == "" {
		// 确保日志目录存在
		logDir := "../log"
		if _, err := os.Stat(logDir); os.IsNotExist(err) {
			os.MkdirAll(logDir, 0755)
		}
		// 生成日期格式的日志文件名
		currentDate := time.Now().Format("2006-01-02")
		logFilePath = filepath.Join(logDir, fmt.Sprintf("app-%s.log", currentDate))
	}

	return &Logger{
		Level:         level,
		ConsoleOutput: consoleOutput,
		FileOutput:    fileOutput,
		LogFilePath:   logFilePath,
	}
}

// 日志级别对应的字符串
func (l *Logger) levelString(level LogLevel) string {
	levels := []string{"DEBUG", "INFO", "WARN", "ERROR", "FATAL"}
	return levels[level]
}

// 写入日志
func (l *Logger) log(level LogLevel, format string, args ...interface{}) {
	// 如果当前日志级别低于设置的级别，不输出
	if level < l.Level {
		return
	}

	// 格式化日志消息
	now := time.Now().Format("2006-01-02 15:04:05.000")
	message := fmt.Sprintf(format, args...)
	logLine := fmt.Sprintf("[%s] [%s] %s\n", now, l.levelString(level), message)

	// 控制台输出
	if l.ConsoleOutput {
		fmt.Print(logLine)
	}

	// 文件输出
	if l.FileOutput && l.LogFilePath != "" {
		file, err := os.OpenFile(l.LogFilePath, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0644)
		if err != nil {
			// 如果文件写入失败，至少在控制台输出错误
			fmt.Printf("Failed to write log to file: %v\n", err)
			return
		}
		defer file.Close()
		file.WriteString(logLine)
	}
}

// Debug 记录调试日志
func (l *Logger) Debug(format string, args ...interface{}) {
	l.log(DEBUG, format, args...)
}

// Info 记录信息日志
func (l *Logger) Info(format string, args ...interface{}) {
	l.log(INFO, format, args...)
}

// Warn 记录警告日志
func (l *Logger) Warn(format string, args ...interface{}) {
	l.log(WARN, format, args...)
}

// Error 记录错误日志
func (l *Logger) Error(format string, args ...interface{}) {
	l.log(ERROR, format, args...)
}

// Fatal 记录致命错误并退出程序
func (l *Logger) Fatal(format string, args ...interface{}) {
	l.log(FATAL, format, args...)
	os.Exit(1)
}

// 创建全局日志实例
type Logging struct {
	Logger *Logger
}

var Log *Logging

// InitLogger 初始化全局日志实例
func InitLogger() {
	Log = &Logging{
		Logger: NewLogger(INFO, true, true, ""), // 默认日志级别为INFO，同时输出到控制台和文件
	}
}
