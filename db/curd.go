package db

import (
	"axisChat/utils/zlog"
	"fmt"
	"gorm.io/gorm"
)

func QueryUserById(id int64, user *User) {
	db := GetDb()
	r := db.Where("id = ?", id).First(user)
	if r.Error != nil && r.Error != gorm.ErrRecordNotFound {
		zlog.Error(r.Error.Error())
		return
	}
	if r.Error != nil {
		zlog.Info(r.Error.Error())
	}
}

func QueryUserByUserName(username string, user *User) {
	db := GetDb()
	r := db.Where("username = ?", username).First(user)
	if r.Error != nil && r.Error != gorm.ErrRecordNotFound {
		zlog.Error(r.Error.Error())
		return
	}
	if r.Error != nil {
		zlog.Info(r.Error.Error())
	}
}

func QueryUserAllGroupId(userid int64, groupIdList *[]int64) {
	db := GetDb()
	var relationList []Relation
	r := db.Where("object_a = ? AND type = ?", userid, "group").First(&relationList)
	if r.Error != nil && r.Error != gorm.ErrRecordNotFound {
		zlog.Error(r.Error.Error())
		return
	}
	if r.Error != nil {
		zlog.Info(r.Error.Error())
	}
	for _, val := range relationList {
		*groupIdList = append(*groupIdList, val.ObjectB)
	}
}

func QueryUserAllFriendId(userid int64, friendIdList *[]int64) {
	db := GetDb()
	var relationList []Relation
	r := db.Where("object_a = ? AND type = ?", userid, "friend").First(&friendIdList)
	if r.Error != nil && r.Error != gorm.ErrRecordNotFound {
		zlog.Error(r.Error.Error())
		return
	}
	if r.Error != nil {
		zlog.Info(r.Error.Error())
	}
	for _, val := range relationList {
		*friendIdList = append(*friendIdList, val.ObjectB)
	}
}

func SearchUserByUsername(query string, userList *[]User) {
	db := GetDb()
	r := db.Where(fmt.Sprintf("username like %q", "%"+query+"%")).First(userList)
	if r.Error != nil && r.Error != gorm.ErrRecordNotFound {
		zlog.Error(r.Error.Error())
		return
	}
	if r.Error != nil {
		zlog.Info(r.Error.Error())
	}
}

func CreateUser(user *User) {
	db := GetDb()
	r := db.Model(&User{}).Create(user)
	if r.Error != nil {
		zlog.Error(r.Error.Error())
		return
	}
}

func UpdateUser(user *User) {
	db := GetDb()
	// 使用struct更新时Updates方法只会更新非零值字段
	r := db.Updates(user)
	if r.Error != nil && r.Error != gorm.ErrRecordNotFound {
		zlog.Error(r.Error.Error())
		return
	}
	if r.Error != nil {
		zlog.Info(r.Error.Error())
	}
}

func UpdateUserPassword(userid int64, encPwd string) error {
	db := GetDb()
	r := db.Model(&User{}).Where("id = ?", userid).Update("password", encPwd)
	if r.Error != nil && r.Error != gorm.ErrRecordNotFound {
		zlog.Error(r.Error.Error())
		return r.Error
	}
	if r.Error != nil {
		zlog.Info(r.Error.Error())
	}
	return nil
}

func QueryGroupById(id int64, group *Group) {
	db := GetDb()
	r := db.Where("id = ?", id).First(group)
	if r.Error != nil && r.Error != gorm.ErrRecordNotFound {
		zlog.Error(r.Error.Error())
		return
	}
	if r.Error != nil {
		zlog.Info(r.Error.Error())
	}
}

func QueryGroupByGroupName(groupName string, group *Group) {
	db := GetDb()
	r := db.Where("group_name = ?", groupName).First(group)
	if r.Error != nil && r.Error != gorm.ErrRecordNotFound {
		zlog.Error(r.Error.Error())
		return
	}
	if r.Error != nil {
		zlog.Info(r.Error.Error())
	}
}

func SearchGroupByGroupName(query string, groupList *[]Group) {
	db := GetDb()
	r := db.Where(fmt.Sprintf("group_name like %q", "%"+query+"%")).First(groupList)
	if r.Error != nil && r.Error != gorm.ErrRecordNotFound {
		zlog.Error(r.Error.Error())
		return
	}
	if r.Error != nil {
		zlog.Info(r.Error.Error())
	}
}

func QueryGroupAllUser(groupId int64, userList *[]User) {
	db := GetDb()
	var relationList []Relation
	r := db.Where("type = ? AND object_b = ?", "group", groupId).Find(&relationList)
	if r.Error != nil && r.Error != gorm.ErrRecordNotFound {
		zlog.Error(r.Error.Error())
		return
	}
	if r.Error != nil {
		return
	}
	var allUserId []int64
	for _, val := range relationList {
		allUserId = append(allUserId, val.ObjectA)
	}
	db.Where("object_a IN ? AND type = ?", allUserId, "group").Find(userList)
	if r.Error != nil && r.Error != gorm.ErrRecordNotFound {
		zlog.Error(r.Error.Error())
		return
	}
	if r.Error != nil {
		zlog.Info(r.Error.Error())
	}
}

func CreateGroup(group *Group) {
	db := GetDb()
	r := db.Model(&Group{}).Create(group)
	if r.Error != nil {
		zlog.Error(r.Error.Error())
		return
	}
}

func UpdateGroup(group *Group) {
	db := GetDb()
	// 使用struct更新时Updates方法只会更新非零值字段
	r := db.Updates(group)
	if r.Error != nil && r.Error != gorm.ErrRecordNotFound {
		zlog.Error(r.Error.Error())
		return
	}
	if r.Error != nil {
		zlog.Info(r.Error.Error())
	}
}

func QueryFriendMessageByPage(userid, friendId int64, msgList *[]Message, current int, pageSize int) {
	if current <= 0 || pageSize <= 0 {
		current = 1
		pageSize = 16
	}
	db := GetDb()
	r := db.Where("from_a = ? AND to_b = ? AND type = ?", userid, friendId, "friend").Limit(pageSize).Offset((current - 1) * pageSize).Order("snow_id ASC").Find(msgList)
	if r.Error != nil && r.Error != gorm.ErrRecordNotFound {
		zlog.Error(r.Error.Error())
		return
	}
	if r.Error != nil {
		zlog.Info(r.Error.Error())
	}
}

func QueryGroupMessageByPage(userid, groupId int64, msgList *[]Message, current int, pageSize int) {
	if current <= 0 || pageSize <= 0 {
		current = 1
		pageSize = 16
	}
	db := GetDb()
	r := db.Where("from_a = ? AND to_b = ? AND type = ?", userid, groupId, "group").Limit(pageSize).Offset((current - 1) * pageSize).Order("snow_id ASC").Find(msgList)
	if r.Error != nil && r.Error != gorm.ErrRecordNotFound {
		zlog.Error(r.Error.Error())
		return
	}
	if r.Error != nil {
		zlog.Info(r.Error.Error())
	}
}

func CreateRelation(relation *Relation) {
	db := GetDb()
	r := db.Where("object_a = ? AND object_b = ? OR type = ?", relation.ObjectA, relation.ObjectB, relation.Type).First(relation)
	if r.Error != nil && r.Error != gorm.ErrRecordNotFound {
		zlog.Error(r.Error.Error())
		return
	}
	if r.Error != nil {
		zlog.Info(r.Error.Error())
		r = db.Create(relation)
		if r.Error != nil {
			zlog.Error(r.Error.Error())
			return
		}
	}
}
