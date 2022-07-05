package task

import (
	"axisChat/common"
	"axisChat/utils/zlog"
	"encoding/json"
	"fmt"
	"github.com/segmentio/kafka-go"
	"math/rand"
)

/*
*Author: AxisZql
*Date: 2022-6-17 8:58 AM
*Desc: push the kafka topic msg to destination「Push-> pushMsgToConnect」
 */

var (
	pushChannel          []chan *kafka.Message
	pushStatusMsgChannel []chan []byte
)

func init() {
	pushChannel = make([]chan *kafka.Message, 2)
	pushStatusMsgChannel = make([]chan []byte, 2)
}

func (task *Task) GoPush() {
	for i := 0; i < len(pushChannel); i++ {
		// 初始化每个缓冲channel的大小为50
		pushChannel[i] = make(chan *kafka.Message, 50)
		pushStatusMsgChannel[i] = make(chan []byte, 50)
		go task.pushMsgToConnect(pushChannel[i])
		go task.pushStatusMsgToConnect(pushStatusMsgChannel[i])
	}
}

func (task *Task) Push(msg *kafka.Message) {
	// 将msg写入channel缓冲区，触发消息推送机制
	pushChannel[rand.Int()%2] <- msg
}

func (task *Task) PushStatusMsg(msg []byte) {
	pushStatusMsgChannel[rand.Int()%2] <- msg
}

func (task *Task) pushStatusMsgToConnect(ch chan []byte) {
	for {
		select {
		case msg := <-ch:
			var payload common.MsgSend
			_ = json.Unmarshal(msg, &payload)
			switch payload.Op {
			case common.OpGroupInfoSend, common.OpGroupOlineUserCountSend:
				//todo 因为所有Group Msg都有group_id 这一项所以利用common.GroupInfoMsg来提取所有类型消息的groupId
				payload.Msg = new(common.GroupInfoMsg)
				_ = json.Unmarshal(msg, &payload)
				allOnlineUserId, err := common.RedisHGetAll(fmt.Sprintf(common.GroupOnlineUser, payload.Msg.(*common.GroupInfoMsg).GroupId))
				if err != nil {
					zlog.Error(fmt.Sprintf("push Group Info msg get err:%v", err))
					break
				}
				// 可以通过限制同一个serverId推送一次消息来解决
				serverIdMap := make(map[string]struct{})
				for _, serverId := range allOnlineUserId {
					serverIdMap[serverId] = struct{}{}
				}
				for serverId := range serverIdMap {
					if serverId == "" {
						continue
					}
					switch payload.Op {
					case common.OpGroupInfoSend:
						task.pushGroupInfoMsg(serverId, payload.Msg.(*common.GroupInfoMsg))
					case common.OpGroupOlineUserCountSend:
						payload.Msg = new(common.GroupCountMsg)
						_ = json.Unmarshal(msg, &payload)
						task.pushGroupCountMsg(serverId, payload.Msg.(*common.GroupCountMsg))
					}
				}
			case common.OpFriendOnlineSend, common.OPFriendOffOnlineSend:
				payload.Msg = new(common.FriendOnlineMsg)
				_ = json.Unmarshal(msg, &payload)
				res, err := common.RedisGetString(fmt.Sprintf(common.UseridMapServerId, payload.Msg.(*common.FriendOnlineMsg).Belong))
				if err != nil {
					zlog.Error(fmt.Sprintf("push signal msg can`t get serverId by friendId err: %v", err))
					break
				}
				serverId := string(res)
				if serverId == "" {
					return
				}
				switch payload.Op {
				case common.OpFriendOnlineSend:
					task.pushFriendOnlineMsg(serverId, payload.Msg.(*common.FriendOnlineMsg))
				case common.OPFriendOffOnlineSend:
					payload.Msg = new(common.FriendOfflineMsg)
					_ = json.Unmarshal(msg, &payload)
					task.pushFriendOfflineMsg(serverId, payload.Msg.(*common.FriendOfflineMsg))
				}
			}
		}
	}
}

func (task *Task) pushMsgToConnect(ch chan *kafka.Message) {
	for {
		select {
		case msg := <-ch:
			var payload common.MsgSend
			_ = json.Unmarshal(msg.Value, &payload)
			switch payload.Op {
			case common.OpGroupMsgSend:
				//todo 因为所有Group Msg都有group_id 这一项所以利用common.GroupInfoMsg来提取所有类型消息的groupId
				payload.Msg = new(common.GroupInfoMsg)
				_ = json.Unmarshal(msg.Value, &payload)
				allOnlineUserId, err := common.RedisHGetAll(fmt.Sprintf(common.GroupOnlineUser, payload.Msg.(*common.GroupInfoMsg).GroupId))
				if err != nil {
					zlog.Error(fmt.Sprintf("push Group Info msg get err:%v", err))
					break
				}
				// 可以通过限制同一个serverId推送一次消息来解决
				serverIdMap := make(map[string]struct{})
				for _, serverId := range allOnlineUserId {
					serverIdMap[serverId] = struct{}{}
				}
				for serverId := range serverIdMap {
					task.pushGroupMsg(serverId, msg)
				}
			case common.OpFriendMsgSend:
				payload.Msg = new(common.FriendOnlineMsg)
				_ = json.Unmarshal(msg.Value, &payload)
				res, err := common.RedisGetString(fmt.Sprintf(common.UseridMapServerId, payload.Msg.(*common.FriendOnlineMsg).Belong))
				if err != nil {
					zlog.Error(fmt.Sprintf("push signal msg can`t get serverId by friendId err: %v", err))
					break
				}
				serverId := string(res)
				task.pushFriendMsg(serverId, msg)
			}
		}
	}
}
