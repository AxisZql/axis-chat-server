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
	GroupId       int64      `json:"groupId"`
	Count         int        `json:"count"`
	UserArr       []db.TUser `json:"userArr"`
	OnlineUserIds []int64    `json:"onlineUserIds"`
	Op            int        `json:"op"`
}

type GroupCountMsg struct {
	GroupId int64 `json:"groupId"`
	Count   int   `json:"count"`
	Op      int   `json:"op"`
}

type FriendOnlineMsg struct {
	FriendId   int64  `json:"friendId"` //上线方id
	FriendName string `json:"friendName"`
	Op         int    `json:"op"`
	Belong     int64  `json:"belong"` //发送给消息接收目标的id，信箱id
}

type FriendOfflineMsg struct {
	FriendId int64 `json:"friendId"` // 下线方id
	Op       int   `json:"op"`
	Belong   int64 `json:"belong"`
}

type GroupMsg struct {
	Userid       int64  `json:"userid"`
	GroupId      int64  `json:"groupId"`
	Content      string `json:"content"`
	MessageType  string `json:"messageType"`
	CreateAt     string `json:"createAt"`
	GroupName    string `json:"groupName"`
	FromUsername string `json:"fromUsername"`
	Avatar       string `json:"avatar"`
	Op           int    `json:"op"`
}

type FriendMsg struct {
	Userid       int64  `json:"userid"`
	FriendId     int64  `json:"friendId"`
	Content      string `json:"content"`
	MessageType  string `json:"messageType"`
	CreateAt     string `json:"createAt"`
	FriendName   string `json:"friendName"`
	FromUsername string `json:"fromUsername"`
	Avatar       string `json:"avatar"`
	Op           int    `json:"op"`
	Belong       int64  `json:"belong"` // 消息所属于的信箱id（用户id），在connect层需要根据这个id把消息推送到对应用户
}
