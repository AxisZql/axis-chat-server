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

const (
	OpFriendMsgSend           = 0
	OpGroupMsgSend            = 1
	OpGroupOlineUserCountSend = 3
	OpGroupInfoSend           = 4
	OpFriendOnlineSend        = 5
	OPFriendOffOnlineSend     = 6
)

type MsgSend struct {
	Op  int         `json:"op"`
	Msg interface{} `json:"msg"`
}

type GroupInfoMsg struct {
	AffectUserid int64     `json:"affect_userid"` // 触发操作的用户id
	GroupId      int64     `json:"group_id"`
	Count        int       `json:"count"`
	UserArr      []db.User `json:"user_arr"`
}

type GroupCountMsg struct {
	GroupId int64 `json:"group_id"`
	Count   int   `json:"count"`
}

type FriendOnlineMsg struct {
	FriendId   int64  `json:"friend_id"` //上线方id
	FriendName string `json:"friend_name"`
}

type FriendOfflineMsg struct {
	FriendId int64 `json:"friend_id"` // 下线方id
}

type GroupMsg struct {
	Userid       int64  `json:"userid"`
	GroupId      int64  `json:"group_id"`
	Content      string `json:"content"`
	MessageType  string `json:"message_type"`
	CreateAt     int64  `json:"create_at"`
	GroupName    string `json:"group_name"`
	FromUsername string `json:"from_username"`
}

type FriendMsg struct {
	Userid       int64  `json:"userid"`
	FriendId     int64  `json:"friend_id"`
	Content      string `json:"content"`
	MessageType  string `json:"message_type"`
	CreateAt     int64  `json:"create_at"`
	FriendName   string `json:"friend_name"`
	FromUsername string `json:"from_username"`
}

func Push(objectId int64, payload interface{}, op int) (err error) {
	var msg interface{}
	var _type = "friend"
	switch op {
	case OpGroupMsgSend:
		msg = MsgSend{
			Op:  OpGroupMsgSend,
			Msg: payload,
		}
		_type = "group"
	case OpFriendMsgSend:
		msg = MsgSend{
			Op:  OpFriendMsgSend,
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
	case OpFriendOnlineSend:
		msg = MsgSend{
			Op: op,
			Msg: FriendOnlineMsg{
				FriendId:   userid,
				FriendName: username,
			},
		}
	case OPFriendOffOnlineSend:
		msg = MsgSend{
			Op: op,
			Msg: FriendOfflineMsg{
				FriendId: userid,
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

func PushGroupInfo(userid, groupId int64, onlineCount int, userList []db.User) (err error) {
	msg := MsgSend{
		Op: OpGroupInfoSend,
		Msg: GroupInfoMsg{
			AffectUserid: userid,
			GroupId:      groupId,
			Count:        onlineCount,
			UserArr:      userList,
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
	msg := MsgSend{
		Op: OpGroupOlineUserCountSend,
		Msg: GroupCountMsg{
			GroupId: groupId,
			Count:   count,
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
