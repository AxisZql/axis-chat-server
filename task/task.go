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
	runtime.GOMAXPROCS(runtime.NumCPU())
	// 初始化connectRcp服务客户端
	task.InitConnectRpcClient()
	// 初始化消息kafka消息消费reader
	go task.UpdateOnlineObjTrigger() //弥补redis只能在对象上线时才触发的缺点
	go task.startTopic()             // 开始监听对应topic，并开始消息推送
	go task.fetchRedisQueueMsg()     // 拉取redis队列中的状态消息

	task.GoPush() // 开始向connect层进行消息推送
}
