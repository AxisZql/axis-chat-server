package main

import (
	"axisChat/config"
	"fmt"
	"github.com/spf13/cobra"
	"os"
	"os/signal"
	"syscall"
)

var rootCmd = &cobra.Command{
	Use:   "AXIS-CHAT",
	Short: "AXIS-CHAT COMMAND",
	Long:  "This is the backend service of AXIS-CHAT",
	Run: func(cmd *cobra.Command, args []string) {
		// 初始化全局配置文件
		config.InitConfig()
	},
}

// 运行模式，决定当前启动的是api、logic、task、connect层服务中的哪一个
var module string

func init() {
	// 通过获取命令行标志位来设置启动的模式
	rootCmd.PersistentFlags().StringVarP(&module, "module", "m", "logic", "set the class of service to run")
}

func Execute() {
	switch module {
	case "logic":
		// 逻辑层服务

	case "connect_websocket":
		// connect层websocket方式的服务

	case "connect_tcp":
		// connect层tcp方式的服务

	case "task":
		// task层服务

	case "api":
		// RESTFUL API 层服务

	default:
		fmt.Println("Wrong running mode!")
		return
	}
	fmt.Println(fmt.Sprintf("run %s module done!", module))
	quit := make(chan os.Signal)
	// 收到系统的中断信号会往quit channel种写入信息然后退出（优雅退出）
	signal.Notify(quit, syscall.SIGHUP, syscall.SIGINT, syscall.SIGTERM, syscall.SIGQUIT)
	<-quit
	fmt.Println("Server exiting")
}
