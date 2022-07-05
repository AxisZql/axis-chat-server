package task

import (
	"axisChat/common"
	"time"
)

func (task *Task) fetchRedisQueueMsg() {
	for {
		res, err := common.RedisBRPOP(common.StatusMsgQueue, 10*time.Second)
		if err != nil {
			continue
		}
		if len(res) < 2 {
			continue
		}
		time.Sleep(500 * time.Millisecond) // 因为在触发登陆时redis还没有写入对应用户的serverId所以先睡以下
		task.PushStatusMsg([]byte(res[1]))
	}
}
