package logic

import (
	"axisChat/common"
	"axisChat/db"
	"axisChat/utils/zlog"
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
		err = common.TopicProduce(objectId, _type, body) //重试一遍
		if err != nil {
			zlog.Error(err.Error())
			err = errors.New("推送消息-异常")
			return
		}
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
				Belong:     friendId, //消息接收发id
			},
		}
	case common.OPFriendOffOnlineSend:
		msg = common.MsgSend{
			Op: op,
			Msg: common.FriendOfflineMsg{
				FriendId: userid,
				Op:       common.OPFriendOffOnlineSend,
				Belong:   friendId,
			},
		}
	}
	body, _ := json.Marshal(&msg)
	err = common.RedisLPUSH(common.StatusMsgQueue, body)
	return
}

func PushGroupInfo(userid, groupId int64, onlineCount int, userList []db.TUser, onlineUserId []int64) (err error) {
	msg := common.MsgSend{
		Op: common.OpGroupInfoSend,
		Msg: common.GroupInfoMsg{
			GroupId:       groupId,
			Count:         onlineCount,
			UserArr:       userList,
			OnlineUserIds: onlineUserId,
			Op:            common.OpGroupInfoSend,
		},
	}
	body, _ := json.Marshal(&msg)
	err = common.RedisLPUSH(common.StatusMsgQueue, body)
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
	err = common.RedisLPUSH(common.StatusMsgQueue, body)
	return
}
