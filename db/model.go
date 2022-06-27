package db

import (
	"gorm.io/gorm"
	"time"
)

/**
*Author: AxisZql
*Date: 2022-5-31
*DESC: DB models define
 */

type TUser struct {
	ID       int64          `json:"id,omitempty" gorm:"primaryKey"`
	Username string         `json:"username,omitempty" gorm:"type:varchar(32);index;comment:'用户名'"`
	Password string         `json:"password,omitempty" gorm:"type:varchar(256);not null;comment:'密码'"`
	Avatar   string         `json:"avatar,omitempty" gorm:"type:varchar(256);not null;comment:'头像'"`
	Role     int            `json:"role,omitempty" gorm:"type:tinyint;default:1;not null;comment:'权限 1用户 2管理员 3超级管理员'"`
	Status   int            `json:"status,omitempty" gorm:"type:tinyint;default:1;not null;comment:'权限 1正常 2封禁'"`
	Tag      string         `json:"tag,omitempty" gorm:"type:varchar(128);comment:'用户标签'"`
	CreateAt time.Time      `json:"createAt,omitempty" gorm:"type:datetime;default:current_timestamp;not null;comment:'创建时间'"`
	UpdateAt time.Time      `json:"updateAt,omitempty" gorm:"type:datetime;autoUpdateTime;not null;comment:'修改时间'"`
	DeleteAt gorm.DeletedAt // gorm 软删除
}

type TGroup struct {
	ID        int64          `json:"id,omitempty" gorm:"primaryKey"`
	Userid    int64          `json:"userid,omitempty" gorm:"type:bigint;not null;comment:'群聊创建者id'"`
	GroupName string         `json:"groupName,omitempty" gorm:"type:varchar(32);index;comment:'群聊名称'"`
	Notice    string         `json:"notice,omitempty" gorm:"type:varchar(1024);comment:'群聊公告'"`
	CreateAt  time.Time      `json:"createAt,omitempty" gorm:"type:datetime;default:current_timestamp;not null;comment:'创建时间'"`
	UpdateAt  time.Time      `json:"updateAt,omitempty" gorm:"type:datetime;autoUpdateTime;not null;comment:'修改时间'"`
	DeleteAt  gorm.DeletedAt // gorm 软删除
}

type TRelation struct {
	ID       int64          `json:"id,omitempty" gorm:"primaryKey"`
	Type     string         `json:"type,omitempty" gorm:"type:varchar(16);not null;comment:'关系类型，friend、group'"`
	ObjectA  int64          `json:"objectA,omitempty" gorm:"type:bigint;not null;comment:'关系对象A，用户id'"`
	ObjectB  int64          `json:"objectB,omitempty" gorm:"type:bigint;not null;comment:'关系对象B，用户id或群聊id'"` //如果type=group，那么此项必须为群聊id
	CreateAt time.Time      `json:"createAt,omitempty" gorm:"type:datetime;default:current_timestamp;not null;comment:'创建时间'"`
	UpdateAt time.Time      `json:"updateAt,omitempty" gorm:"type:datetime;autoUpdateTime;not null;comment:'修改时间'"`
	DeleteAt gorm.DeletedAt // gorm 软删除
}

type TMessage struct {
	ID          int64          `json:"id,omitempty" gorm:"primaryKey"`
	Belong      int64          `json:"belong" gorm:"type:bigint;not null;comment:'信箱所有者id「userid or groupId」'"`
	SnowID      string         `json:"snowId,omitempty" gorm:"type:varchar(512);uniqueIndex;comment:'消息雪花id'"`
	Type        string         `json:"type,omitempty" gorm:"type:varchar(16);not null;comment:'消息类型，friend、group'"`
	Content     string         `json:"content,omitempty" gorm:"type:varchar(1024);not null;comment:'消息内容'"`
	FromA       int64          `json:"fromA,omitempty" gorm:"type:bigint;not null;comment:'消息发送方，用户id'"`
	ToB         int64          `json:"toB,omitempty" gorm:"type:bigint;not null;comment:'消息接收方，用户id或群聊id'"` //如果type=group，那么此项必须为群聊id
	MessageType string         `json:"messageType,omitempty" gorm:"type:varchar(16);not null;comment:'消息类型，图片或者文字'"`
	CreateAt    time.Time      `json:"createAt,omitempty" gorm:"type:datetime;default:current_timestamp;not null;comment:'创建时间'"`
	UpdateAt    time.Time      `json:"updateAt,omitempty" gorm:"type:datetime;autoUpdateTime;not null;comment:'修改时间'"`
	DeleteAt    gorm.DeletedAt // gorm 软删除
}

// db views model

type VGroupMessage struct {
	ID           int64     `json:"id"`
	Belong       int64     `json:"belong"`
	Userid       int64     `json:"userid"`
	GroupId      int64     `json:"groupId"`
	Content      string    `json:"content"`
	MessageType  string    `json:"messageType"`
	GroupName    string    `json:"groupName"`
	FromUsername string    `json:"fromUsername"`
	Avatar       string    `json:"avatar"`
	SnowId       string    `json:"snowId"`
	CreateAt     time.Time `json:"createAt"`
}

type VFriendMessage struct {
	ID           int64     `json:"id"`
	Belong       int64     `json:"belong"`
	Userid       int64     `json:"userid"`
	FriendId     int64     `json:"friendId"`
	Content      string    `json:"content"`
	MessageType  string    `json:"messageType"`
	FriendName   string    `json:"friendName"`
	FromUsername string    `json:"fromUsername"`
	Avatar       string    `json:"avatar"`
	SnowId       string    `json:"snowId"`
	CreateAt     time.Time `json:"createAt"`
}
