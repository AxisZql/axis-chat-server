package main

import (
	"axisChat/api"
	"axisChat/common"
	"axisChat/config"
	"axisChat/connect"
	"axisChat/db"
	"axisChat/logic"
	"axisChat/task"
	"axisChat/utils/zlog"
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
		// 初始化db
		db.InitDb()
		// 初始化redis
		common.InitRedis()
		Execute()
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
		logic.New().Run()
	case "connect_websocket":
		// connect层websocket方式的服务
		connect.New().Run()
	case "connect_tcp":
		// connect层tcp方式的服务
		zlog.Warn("the tcp delivery mode is not currently written")
	case "task":
		// task层服务
		task.New().Run()
	case "api":
		// RESTFUL API 层服务
		api.New().Run()
	default:
		fmt.Println("Wrong running mode!")
		return
	}
	zlog.Info(fmt.Sprintf("run %s module done!", module))
	fmt.Println(`         _          _      _              _        _             _                _            _
	       / /\      /_/\    /\ \           /\ \     / /\         /\ \              /\ \         _\ \
	      / /  \     \ \ \   \ \_\          \ \ \   / /  \       /  \ \            /  \ \       /\__ \
	     / / /\ \     \ \ \__/ / /          /\ \_\ / / /\ \__ __/ /\ \ \          / /\ \ \     / /_ \_\
	    / / /\ \ \     \ \__ \/_/          / /\/_// / /\ \___/___/ /\ \ \        / / /\ \ \   / / /\/_/
	   / / /  \ \ \     \/_/\__/\         / / /   \ \ \ \/___\___\/ / / /       / / /  \ \_\ / / /
	  / / /___/ /\ \     _/\/__\ \       / / /     \ \ \           / / /       / / / _ / / // / /
	 / / /_____/ /\ \   / _/_/\ \ \     / / /  _    \ \ \         / / /    _  / / / /\ \/ // / / ____
	/ /_________/\ \ \ / / /   \ \ \___/ / /__/_/\__/ / /         \ \ \__/\_\/ / /__\ \ \// /_/_/ ___/\
	/ / /_       __\ \_/ / /    /_/ /\__\/_/___\ \/___/ /           \ \___\/ / / /____\ \ /_______/\__\/
	\_\___\     /____/_\/_/     \_\/\/_________/\_____\/             \/___/_/\/________\_\\_______\/
	                                                                                                    `)
	quit := make(chan os.Signal)
	// 收到系统的中断信号会往quit channel种写入信息然后退出（优雅退出）
	signal.Notify(quit, syscall.SIGHUP, syscall.SIGINT, syscall.SIGTERM, syscall.SIGQUIT)
	<-quit
	fmt.Println("Server exiting")
}

func main() {
	if err := rootCmd.Execute(); err != nil {
		zlog.Error(fmt.Sprintf("服务启动失败:%v", err))
		os.Exit(1)
	}
}
