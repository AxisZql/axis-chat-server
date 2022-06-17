package task

import (
	"axisChat/common"
	"axisChat/utils/zlog"
	"encoding/json"
	"fmt"
	"github.com/segmentio/kafka-go"
	"math/rand"
	"strconv"
)

/*
*Author: AxisZql
*Date: 2022-6-17 8:58 AM
*Desc: push the kafka topic msg to destination「Push-> pushMsgToConnect」
 */

var pushChannel []chan *kafka.Message

func init() {
	pushChannel = make([]chan *kafka.Message, 2)
}

func (task *Task) GoPush() {
	for i := 0; i < len(pushChannel); i++ {
		// 初始化每个缓冲channel的大小为50
		pushChannel[i] = make(chan *kafka.Message, 50)
		go task.pushMsgToConnect(pushChannel[i])
	}
}

func (task *Task) Push(msg *kafka.Message) {
	// 将msg写入channel缓冲区，触发消息推送机制
	pushChannel[rand.Int()%2] <- msg
}

func (task *Task) pushMsgToConnect(ch chan *kafka.Message) {
	for {
		select {
		case msg := <-ch:
			var payload common.MsgSend
			_ = json.Unmarshal(msg.Value, &payload)
			switch payload.Op {
			case common.OpGroupInfoSend, common.OpGroupOlineUserCountSend, common.OpGroupMsgSend:
				//todo 因为所有Group Msg都有group_id 这一项所以利用common.GroupInfoMsg来提取所有类型消息的groupId
				payload.Msg = new(common.GroupInfoMsg)
				_ = json.Unmarshal(msg.Value, &payload)
				allOnlineUserId, err := common.RedisHGetAll(fmt.Sprintf(common.GroupOnlineUser, payload.Msg.(*common.GroupInfoMsg).GroupId))
				if err != nil {
					zlog.Error(fmt.Sprintf("push Group Info msg get err:%v", err))
					break
				}
				for id := range allOnlineUserId {
					userid, _ := strconv.Atoi(id)
					res, err := common.RedisGetString(fmt.Sprintf(fmt.Sprintf(common.UseridMapServerId, userid)))
					if err != nil {
						zlog.Error(fmt.Sprintf("push Group Info msg when get serverId get err:%v", err))
						continue
					}
					serverId := string(res)
					switch payload.Op {
					case common.OpGroupInfoSend:
						task.pushGroupInfoMsg(serverId, msg)
					case common.OpGroupOlineUserCountSend:
						task.pushGroupCountMsg(serverId, msg)
					case common.OpGroupMsgSend:
						task.pushGroupMsg(serverId, msg)
					}

				}
			case common.OpFriendOnlineSend, common.OPFriendOffOnlineSend, common.OpFriendMsgSend:
				payload.Msg = new(common.FriendOnlineMsg)
				_ = json.Unmarshal(msg.Value, &payload)
				res, err := common.RedisGetString(fmt.Sprintf(common.UseridMapServerId, payload.Msg.(*common.FriendOnlineMsg).FriendId))
				if err != nil {
					zlog.Error(fmt.Sprintf("push signal msg can`t get serverId by friendId err: %v", err))
					break
				}
				serverId := string(res)
				switch payload.Op {
				case common.OpFriendMsgSend:
					task.pushFriendMsg(serverId, msg)
				case common.OpFriendOnlineSend:
					task.pushFriendOnlineMsg(serverId, msg)
				case common.OPFriendOffOnlineSend:
					task.pushFriendOfflineMsg(serverId, msg)
				}
			}
		}
	}
}
