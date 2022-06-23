package common

import "axisChat/db"

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
	GroupId int64      `json:"group_id"`
	Count   int        `json:"count"`
	UserArr []db.TUser `json:"user_arr"`
	Op      int        `json:"op"`
}

type GroupCountMsg struct {
	GroupId int64 `json:"group_id"`
	Count   int   `json:"count"`
	Op      int   `json:"op"`
}

type FriendOnlineMsg struct {
	FriendId   int64  `json:"friend_id"` //上线方id
	FriendName string `json:"friend_name"`
	Op         int    `json:"op"`
}

type FriendOfflineMsg struct {
	FriendId int64 `json:"friend_id"` // 下线方id
	Op       int   `json:"op"`
}

type GroupMsg struct {
	Userid       int64  `json:"userid"`
	GroupId      int64  `json:"group_id"`
	Content      string `json:"content"`
	MessageType  string `json:"message_type"`
	CreateAt     string `json:"create_at"`
	GroupName    string `json:"group_name"`
	FromUsername string `json:"from_username"`
	Op           int    `json:"op"`
}

type FriendMsg struct {
	Userid       int64  `json:"userid"`
	FriendId     int64  `json:"friend_id"`
	Content      string `json:"content"`
	MessageType  string `json:"message_type"`
	CreateAt     string `json:"create_at"`
	FriendName   string `json:"friend_name"`
	FromUsername string `json:"from_username"`
	Op           int    `json:"op"`
}
