package task

import "runtime"

/**
*Author:AxisZql
*Date:2022-6-14 2:18 PM
*Desc:start up task layer
 */

type Task struct{}

func New() *Task {
	return &Task{}
}

func (task *Task) Run() {
	// 配置connect层最多使用的核心数为4
	runtime.GOMAXPROCS(4)
	// 初始化connectRcp服务客户端
	task.InitConnectRpcClient()
	// 初始化消息kafka消息消费reader
	task.UpdateOnlineObjTrigger() // 定时更新所有在线对象数据
	go task.startTopic()          // 开始监听对应topic，并开始消息推送

	task.GoPush() // 开始向connect层进行消息推送
}
