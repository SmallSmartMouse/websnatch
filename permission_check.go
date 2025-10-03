// Package main
// @time    :2025/9/2319:26
package main

import (
	"abc/a/util"
	"os/exec"
	"strings"

	"github.com/google/gopacket/pcap"
)

// promptUserForPermission 根据是否是本机请求决定是否打开系统设置
func promptUserForPermission(isLocalRequest bool) {
	util.Log.Logger.Info("提示用户授予网络监控权限，是否本机请求: %v", isLocalRequest)
	cmd := exec.Command("osascript", "-e", `display dialog "本程序需要\"网络监控\"权限才能抓包。\n\n请按以下步骤操作：\n\n1. 点击下方\"前往设置\"按钮，系统将自动打开隐私设置\n2. 在左侧菜单中选择\"网络监控\"选项\n3. 点击右侧的\"+\"按钮，添加本程序\n4. 勾选本程序旁边的复选框以授予权限\n5. 授予权限后，请重新运行本程序" buttons {"前往设置", "取消"} default button 1 with icon caution`)
	result, _ := cmd.CombinedOutput()
	buttonPressed := string(result)

	// 如果是本机请求且用户点击了"前往设置"按钮，则打开系统设置面板
	if isLocalRequest && strings.Contains(buttonPressed, "button returned:前往设置") {
		util.Log.Logger.Info("用户点击了前往设置按钮，正在打开系统设置面板")
		exec.Command("open", "x-apple.systempreferences:com.apple.preference.security?Privacy_NetworkCapture").Run()
	} else {
		util.Log.Logger.Info("用户取消了权限设置")
	}
}
func hasNetworkCapturePermission() bool {
	devices, err := pcap.FindAllDevs()
	if err != nil {
		util.Log.Logger.Error("获取设备列表失败: %v", err)
		return false
	}
	for _, dev := range devices {
		if len(dev.Addresses) > 0 {
			handle, err := pcap.OpenLive(dev.Name, 1600, true, pcap.BlockForever)
			if err == nil {
				handle.Close()
				util.Log.Logger.Info("已获得网络监控权限")
				return true
			}
		}
	}
	util.Log.Logger.Warn("未获得网络监控权限")
	return false
}
func test(requestIP string) bool {
	if !hasNetworkCapturePermission() {
		isLocalRequest := requestIP == "127.0.0.1" || requestIP == "localhost" || requestIP == util.GetLocalIP()
		util.Log.Logger.Warn("网络监控权限检查失败，请求IP: %s，是否本机请求: %v", requestIP, isLocalRequest)
		promptUserForPermission(isLocalRequest)
		return false
	}
	util.Log.Logger.Info("已获得网络监控权限，开始抓包...")
	return true
}
