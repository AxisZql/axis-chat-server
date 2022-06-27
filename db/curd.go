package db

import (
	"axisChat/utils/zlog"
	"fmt"
	"gorm.io/gorm"
)

func QueryUserById(id int64, user *TUser) {
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

func QueryUserByUserName(username string, user *TUser) {
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

func QueryUserAllGroupId(userid int64) (groupIdList []int64) {
	db := GetDb()
	var relationList []TRelation
	r := db.Where("object_a = ? AND type = ?", userid, "group").Find(&relationList)
	if r.Error != nil && r.Error != gorm.ErrRecordNotFound {
		zlog.Error(r.Error.Error())
		return
	}
	if r.Error != nil {
		zlog.Info(r.Error.Error())
	}
	groupIdList = make([]int64, 0)
	for _, val := range relationList {
		// todo slice 扩容后，其不一定指向原来的地址
		groupIdList = append(groupIdList, val.ObjectB)
	}
	return groupIdList
}

func QueryUserAllFriendId(userid int64) (friendIdList []int64) {
	db := GetDb()
	var relationList []TRelation
	r := db.Where("object_a = ? AND type = ?", userid, "friend").Find(&relationList)
	if r.Error != nil && r.Error != gorm.ErrRecordNotFound {
		zlog.Error(r.Error.Error())
		return
	}
	if r.Error != nil {
		zlog.Info(r.Error.Error())
	}
	friendIdList = make([]int64, 0)
	for _, val := range relationList {
		friendIdList = append(friendIdList, val.ObjectB)
	}
	return friendIdList
}

func SearchUserByUsername(query string, userList *[]TUser) {
	db := GetDb()
	r := db.Select([]string{"id", "username", "avatar", "role", "status", "tag", "create_at", "update_at"}).Where(fmt.Sprintf("username like %q", "%"+query+"%")).Find(userList)
	if r.Error != nil && r.Error != gorm.ErrRecordNotFound {
		zlog.Error(r.Error.Error())
		return
	}
	if r.Error != nil {
		zlog.Info(r.Error.Error())
	}
}

func CreateUser(user *TUser) {
	db := GetDb()
	r := db.Model(&TUser{}).Create(user)
	if r.Error != nil {
		zlog.Error(r.Error.Error())
		return
	}
}

func UpdateUser(user *TUser) {
	db := GetDb()
	// 使用struct更新时Updates方法只会更新非零值字段
	r := db.Updates(user).Find(user)
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
	r := db.Model(&TUser{}).Where("id = ?", userid).Update("password", encPwd)
	if r.Error != nil && r.Error != gorm.ErrRecordNotFound {
		zlog.Error(r.Error.Error())
		return r.Error
	}
	if r.Error != nil {
		zlog.Info(r.Error.Error())
	}
	return nil
}

func QueryGroupById(id int64, group *TGroup) {
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

func QueryGroupByGroupName(groupName string, group *TGroup) {
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

func SearchGroupByGroupName(query string, groupList *[]TGroup) {
	db := GetDb()
	r := db.Where(fmt.Sprintf("group_name like %q", "%"+query+"%")).Find(groupList)
	if r.Error != nil && r.Error != gorm.ErrRecordNotFound {
		zlog.Error(r.Error.Error())
		return
	}
	if r.Error != nil {
		zlog.Info(r.Error.Error())
	}
}

func QueryGroupAllUser(groupId int64, userList *[]TUser) {
	db := GetDb()
	var relationList []TRelation
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
	db.Select([]string{"id", "username", "avatar", "role", "status", "tag", "create_at", "update_at"}).Where("id IN ?", allUserId).Find(userList)
	if r.Error != nil && r.Error != gorm.ErrRecordNotFound {
		zlog.Error(r.Error.Error())
		return
	}
	if r.Error != nil {
		zlog.Info(r.Error.Error())
	}
}

func CreateGroup(group *TGroup) {
	db := GetDb()
	r := db.Model(&TGroup{}).Create(group)
	if r.Error != nil {
		zlog.Error(r.Error.Error())
		return
	}
}

func UpdateGroup(group *TGroup) {
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

func QueryFriendMessageByPage(userid, friendId int64, msgList *[]VFriendMessage, current int, pageSize int) {
	if current <= 0 || pageSize <= 0 {
		current = 1
		pageSize = 16
	}
	db := GetDb()
	r := db.Table("v_friend_message").Where("((userid = ? AND friend_id = ?) OR (friend_id = ? AND userid = ?)) AND belong = ?", userid, friendId, userid, friendId, userid).Limit(pageSize).Offset((current - 1) * pageSize).Find(msgList)
	if r.Error != nil && r.Error != gorm.ErrRecordNotFound {
		zlog.Error(r.Error.Error())
		return
	}
	if r.Error != nil {
		zlog.Info(r.Error.Error())
	}
}

func QueryGroupMessageByPage(groupId int64, msgList *[]VGroupMessage, current int, pageSize int) {
	if current <= 0 || pageSize <= 0 {
		current = 1
		pageSize = 16
	}
	db := GetDb()
	r := db.Table("v_group_message").Where("group_id = ? AND belong = ?", groupId, groupId).Limit(pageSize).Offset((current - 1) * pageSize).Find(msgList)
	if r.Error != nil && r.Error != gorm.ErrRecordNotFound {
		zlog.Error(r.Error.Error())
		return
	}
	if r.Error != nil {
		zlog.Info(r.Error.Error())
	}
}

func CreateRelation(relation *TRelation) {
	db := GetDb()
	r := db.Where("object_a = ? AND object_b = ? AND type = ?", relation.ObjectA, relation.ObjectB, relation.Type).First(relation)
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

func SaveMsgInBatches(msg []TMessage) {
	if msg == nil || len(msg) == 0 {
		return
	}
	db := GetDb()
	r := db.CreateInBatches(msg, 100) // 每次更新写入100条数据
	if r.Error != nil {
		zlog.Error(r.Error.Error())
		return
	}
}
