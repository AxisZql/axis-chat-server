package logic

import (
	"axisChat/common"
	"axisChat/db"
	"encoding/json"
	"github.com/pkg/errors"
)

/**
*Author:AxisZql
*Date:2022-5-31
*DESC:将消息推送到对应到消息队列中
 */

func Push(objectId int64, payload interface{}, op int) (err error) {
	var msg interface{}
	var _type = "friend"
	switch op {
	case common.OpGroupMsgSend:
		msg = common.MsgSend{
			Op:  common.OpGroupMsgSend,
			Msg: payload,
		}
		_type = "group"
	case common.OpFriendMsgSend:
		msg = common.MsgSend{
			Op:  common.OpFriendMsgSend,
			Msg: payload,
		}
	}
	body, _ := json.Marshal(&msg)
	err = common.TopicProduce(objectId, _type, body)
	if err != nil {
		err = errors.New("系统异常")
		return
	}
	return
}

func PushUserInfo(userid int64, username string, friendId int64, op int) (err error) {
	var msg interface{}
	switch op {
	case common.OpFriendOnlineSend:
		msg = common.MsgSend{
			Op: op,
			Msg: common.FriendOnlineMsg{
				FriendId:   userid,
				FriendName: username,
				Op:         common.OpFriendOnlineSend,
			},
		}
	case common.OPFriendOffOnlineSend:
		msg = common.MsgSend{
			Op: op,
			Msg: common.FriendOfflineMsg{
				FriendId: userid,
				Op:       common.OPFriendOffOnlineSend,
			},
		}
	}
	body, _ := json.Marshal(&msg)
	// TODO:往该好友的的topic中生产消息
	err = common.TopicProduce(friendId, "friend", body)
	if err != nil {
		err = errors.New("系统异常")
		return
	}
	return
}

func PushGroupInfo(userid, groupId int64, onlineCount int, userList []db.TUser) (err error) {
	msg := common.MsgSend{
		Op: common.OpGroupInfoSend,
		Msg: common.GroupInfoMsg{
			GroupId: groupId,
			Count:   onlineCount,
			UserArr: userList,
			Op:      common.OpGroupInfoSend,
		},
	}
	body, _ := json.Marshal(&msg)
	// TODO:往该群聊的的topic中生产消息
	err = common.TopicProduce(groupId, "group", body)
	if err != nil {
		err = errors.New("系统异常")
		return
	}
	return
}

func PushGroupCount(groupId int64, count int) (err error) {
	msg := common.MsgSend{
		Op: common.OpGroupOlineUserCountSend,
		Msg: common.GroupCountMsg{
			GroupId: groupId,
			Count:   count,
			Op:      common.OpGroupOlineUserCountSend,
		},
	}
	body, _ := json.Marshal(&msg)
	// TODO:往该群聊的的topic中生产消息
	err = common.TopicProduce(groupId, "group", body)
	if err != nil {
		err = errors.New("系统异常")
		return
	}
	return
}
