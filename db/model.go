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

type User struct {
	ID       int64          `json:"id,omitempty" gorm:"primaryKey"`
	Username string         `json:"username,omitempty" gorm:"type:varchar(32);index;comment:'用户名'"`
	Password string         `json:"password,omitempty" gorm:"type:varchar(256);not null;comment:'密码'"`
	Avatar   string         `json:"avatar,omitempty" gorm:"type:varchar(256);not null;comment:'头像'"`
	Role     int            `json:"role,omitempty" gorm:"type:tinyint;default:0;not null;comment:'权限 0用户 1管理员 2超级管理员'"`
	Status   int            `json:"status,omitempty" gorm:"type:tinyint;default:0;not null;comment:'权限 0正常 1封禁'"`
	Tag      string         `json:"tag,omitempty" gorm:"type:varchar(128);comment:'用户标签'"`
	CreateAt time.Time      `json:"create_at,omitempty" gorm:"type:datetime;default:current_timestamp;not null;comment:'创建时间'"`
	UpdateAt time.Time      `json:"update_at,omitempty" gorm:"type:datetime;autoUpdateTime;not null;comment:'修改时间'"`
	DeleteAt gorm.DeletedAt // gorm 软删除
}

type Group struct {
	ID        int64          `json:"id,omitempty" gorm:"primaryKey"`
	Userid    int64          `json:"userid,omitempty" gorm:"type:bigint;not null;comment:'群聊创建者id'"`
	GroupName string         `json:"group_name,omitempty" gorm:"type:'varchar(32);index;comment:'群聊名称'"`
	Notice    string         `json:"notice,omitempty" gorm:"type:varchar(1024);comment:'群聊公告'"`
	CreateAt  time.Time      `json:"create_at,omitempty" gorm:"type:datetime;default:current_timestamp;not null;comment:'创建时间'"`
	UpdateAt  time.Time      `json:"update_at,omitempty" gorm:"type:datetime;autoUpdateTime;not null;comment:'修改时间'"`
	DeleteAt  gorm.DeletedAt // gorm 软删除
}

type Relation struct {
	ID       int64          `json:"id,omitempty" gorm:"primaryKey"`
	Type     string         `json:"type,omitempty" gorm:"type:varchar(16);not null;comment:'关系类型，friend、group'"`
	ObjectA  int64          `json:"object_a,omitempty" gorm:"type:bigint;not null;comment:'关系对象A，用户id'"`
	ObjectB  int64          `json:"object_b,omitempty" gorm:"type:bigint;not null;comment:'关系对象B，用户id或群聊id'"` //如果type=group，那么此项必须为群聊id
	CreateAt time.Time      `json:"create_at,omitempty" gorm:"type:datetime;default:current_timestamp;not null;comment:'创建时间'"`
	UpdateAt time.Time      `json:"update_at,omitempty" gorm:"type:datetime;autoUpdateTime;not null;comment:'修改时间'"`
	DeleteAt gorm.DeletedAt // gorm 软删除
}

type Message struct {
	ID          int64          `json:"id,omitempty" gorm:"primaryKey"`
	SnowID      string         `json:"snow_id,omitempty" gorm:"varchar(256);uniqueIndex;comment:'消息雪花id'"`
	Type        string         `json:"type,omitempty" gorm:"type:varchar(16);not null;comment:'消息类型，friend、group'"`
	Content     string         `json:"content,omitempty" gorm:"type:varchar(1024);not null;comment:'消息内容'"`
	FromA       int64          `json:"from_a,omitempty" gorm:"type:bigint;not null;comment:'消息发送方，用户id'"`
	ToB         int64          `json:"to_b,omitempty" gorm:"type:bigint;not null;comment:'消息接收方，用户id或群聊id'"` //如果type=group，那么此项必须为群聊id
	MessageType string         `json:"message_type,omitempty" gorm:"type:varchar(16);not null;comment:'消息类型，图片或者文字'"`
	CreateAt    time.Time      `json:"create_at,omitempty" gorm:"type:datetime;default:current_timestamp;not null;comment:'创建时间'"`
	UpdateAt    time.Time      `json:"update_at,omitempty" gorm:"type:datetime;autoUpdateTime;not null;comment:'修改时间'"`
	DeleteAt    gorm.DeletedAt // gorm 软删除
}
