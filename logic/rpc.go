package logic

import (
	"axisChat/common"
	"axisChat/config"
	"axisChat/db"
	"axisChat/proto"
	"axisChat/utils"
	"axisChat/utils/zlog"
	"context"
	"encoding/json"
	"fmt"
	"github.com/golang/protobuf/ptypes/empty"
	"github.com/pkg/errors"
	"strconv"
	"time"
)

/**
*Author:AxisZql
*Date:2022-5-31
*DESC:implement the rpc server interface
 */

type ServerLogic struct{}

// Connect 当客户端通过connect层连接服务时，调用该rpc方法通过查看redis中有没有该用户的登陆信息，从而达到鉴权的目的，如果是
// 私聊的话将该用户的上线信息写入其好友的所有MQ中，如果是群聊的话，将该群的人数信息推送到群聊对应的MQ中
// 用户上线利用消息队列主动提醒
func (s *ServerLogic) Connect(ctx context.Context, request *proto.ConnectRequest) (reply *proto.ConnectReply, err error) {
	reply = new(proto.ConnectReply)
	sessionIdKey := fmt.Sprintf(common.SESSION, request.AccessToken)
	res, err := common.RedisGetString(sessionIdKey)
	if err != nil {
		err = errors.New("系统异常")
		return
	}
	if string(res) == "" {
		err = errors.New("不合法的access_token")
		return
	}
	var user db.TUser
	_ = json.Unmarshal(res, &user)
	// TODO：通过serverId查找当前connect的用户在connect层的哪一个服务微服务中
	serverIdKey := fmt.Sprintf(common.UseridMapServerId, user.ID)
	err = common.RedisSetString(serverIdKey, []byte(request.ServerId), 0)
	if err != nil {
		err = errors.New("系统异常")
		return
	}
	// 在通知该用户加入的群聊、所有加好友，其的上线消息
	groupIdList := db.QueryUserAllGroupId(user.ID)
	friendIdList := db.QueryUserAllFriendId(user.ID)

	// 更改对应用户的在线状态为在线
	if err = common.RedisHSet(common.AllOnlineUser, fmt.Sprintf("%d", user.ID), "on"); err != nil {
		err = errors.New("系统异常")
		return
	}

	// 更新该用户加入群聊的所有在redis中信息
	for _, val := range groupIdList {
		var has bool
		// 群聊在线用户hash table 存放userid和对应用户所在serverId的serverId
		has, err = common.RedisIsNotExistHSet(fmt.Sprintf(common.GroupOnlineUser, val), fmt.Sprintf("%d", user.ID), request.ServerId)
		if err != nil {
			err = errors.New("系统异常")
			return
		}
		// 之前不在线的用户上线时才更新群聊在线人数
		if !has {
			// 更新hash table中对应群聊项的在线人数
			err = common.RedisHINCRBY(common.GroupOnlineUserCount, fmt.Sprintf("%d", val), 1)
			if err != nil {
				err = errors.New("系统异常")
				return
			}
		}
		//往群聊对应topic写入群聊在线人数更新信息 TODO:need to test
		var userList []db.TUser
		db.QueryGroupAllUser(val, &userList)
		countStr, _ := common.RedisHGet(common.GroupOnlineUserCount, fmt.Sprintf("%d", val))
		count, _ := strconv.Atoi(countStr)
		var onlineUserIdList []int64
		// 获取当前群聊的所有在线用户的id
		var userMap map[string]string
		userMap, err = common.RedisHGetAll(fmt.Sprintf(common.GroupOnlineUser, val))
		if err != nil {
			err = errors.New("系统异常")
			return
		}
		for k := range userMap {
			id, _ := strconv.Atoi(k)
			onlineUserIdList = append(onlineUserIdList, int64(id))
		}
		err = PushGroupInfo(user.ID, val, count, userList, onlineUserIdList)
		if err != nil {
			err = errors.New("系统异常")
			return
		}
	}
	// 在redis中写入当前用户的加入的所有群聊id
	b, _ := json.Marshal(groupIdList)
	err = common.RedisSetString(fmt.Sprintf(common.UserGroupList, user.ID), b, 0)
	if err != nil {
		err = errors.New("系统异常")
		return
	}
	// 往好友对应topic写入该用户上线信息
	for _, val := range friendIdList {
		err = PushUserInfo(user.ID, user.Username, val, common.OpFriendOnlineSend)
		if err != nil {
			err = errors.New("系统异常")
			return
		}
	}

	// 在redis中写入当前用户所有好友的id
	b, _ = json.Marshal(friendIdList)
	err = common.RedisSetString(fmt.Sprintf(common.UserFriendList, user.ID), b, 0)
	if err != nil {
		err = errors.New("系统异常")
		return
	}

	return &proto.ConnectReply{
		Userid: user.ID,
	}, nil
}

// DisConnect 当connect层检测到对应客户端下线后，更新redis中该用户的在线信息，并在好友和其所在群聊的MQ中写入其下线的信息
// 下也利用MQ主动提醒
func (s *ServerLogic) DisConnect(ctx context.Context, request *proto.DisConnectRequest) (reply *empty.Empty, err error) {
	reply = new(empty.Empty)
	// 在通知该用户加入的群聊、所有加好友，其的上线消息

	groupIdList := db.QueryUserAllGroupId(request.Userid)
	friendIdList := db.QueryUserAllFriendId(request.Userid)

	// 更改对应用户的在线状态为下线,todo:不采取删除而是采取字段更改的原因是，删除对应field 频繁涉及空间的申请和释放
	if err = common.RedisHSet(common.AllOnlineUser, fmt.Sprintf("%d", request.Userid), "off"); err != nil {
		err = errors.New("系统异常")
		return
	}

	// 更新该用户加入群聊的所有在redis中信息
	for _, val := range groupIdList {
		var has bool
		has, err = common.RedisHDel(fmt.Sprintf(common.GroupOnlineUser, val), fmt.Sprintf("%d", request.Userid))
		if err != nil {
			err = errors.New("系统异常")
			return
		}
		if has {
			err = common.RedisHINCRBY(common.GroupOnlineUserCount, fmt.Sprintf("%d", val), -1)
			if err != nil {
				err = errors.New("系统异常")
				return
			}
		}
		//往群聊对应topic写入群聊在线人数更新信息 TODO:need to test
		var userList []db.TUser
		db.QueryGroupAllUser(val, &userList)
		countStr, _ := common.RedisHGet(common.GroupOnlineUserCount, fmt.Sprintf("%d", val))
		count, _ := strconv.Atoi(countStr)
		var onlineUserIdList []int64
		// 获取当前群聊的所有在线用户的id
		var userMap map[string]string
		userMap, err = common.RedisHGetAll(fmt.Sprintf(common.GroupOnlineUser, val))
		if err != nil {
			err = errors.New("系统异常")
			return
		}
		for k := range userMap {
			id, _ := strconv.Atoi(k)
			onlineUserIdList = append(onlineUserIdList, int64(id))
		}

		// TODO:往该群聊的的topic中生产消息
		err = PushGroupInfo(request.Userid, val, count, userList, onlineUserIdList)
		if err != nil {
			err = errors.New("系统异常")
			return
		}
	}
	// 往好友对应topic写入该用户上线信息
	for _, val := range friendIdList {
		// TODO:往该好友的的topic中生产消息
		err = PushUserInfo(request.Userid, "", val, common.OPFriendOffOnlineSend)
		if err != nil {
			err = errors.New("系统异常")
			return
		}
	}
	return reply, nil
}

func (s *ServerLogic) Register(ctx context.Context, request *proto.RegisterRequest) (reply *proto.RegisterReply, err error) {
	reply = new(proto.RegisterReply)
	reply.Code = config.FailReplyCode
	user := new(db.TUser)
	db.QueryUserByUserName(request.Username, user)
	if user.ID > 0 {
		err = errors.New("当前用户已经存在，请直接登陆")
		return
	}
	user.Username = request.Username
	pwd := utils.EncPassword(request.Password)
	if pwd == "" {
		// TODO：这样做的目的时不把详细错误返回给下游服务
		err = errors.New("系统异常")
		return
	}
	user.Password = pwd
	user.Avatar = config.GetConfig().Api.Api.DefaultAvatar
	db.CreateUser(user)
	if user.ID == 0 {
		err = errors.New("系统异常")
		return
	}
	// set token
	accessToken := utils.GetRandomToken(32)
	sessionKey := fmt.Sprintf(common.SESSION, accessToken)
	body, _ := json.Marshal(user)
	err = common.RedisSetString(sessionKey, body, config.RedisBaseValidTime*time.Second)
	if err != nil {
		zlog.Error(fmt.Sprintf("set key:%s,get err:%v", sessionKey, err))
		return
	}
	reply.Code = config.SuccessReplyCode
	reply.AccessToken = accessToken
	reply.Userid = user.ID
	return
}

func (s *ServerLogic) Login(ctx context.Context, request *proto.LoginRequest) (reply *proto.LoginReply, err error) {
	reply = new(proto.LoginReply)
	reply.Code = config.FailReplyCode
	user := new(db.TUser)
	db.QueryUserByUserName(request.Username, user)
	if user.ID == 0 {
		err = errors.New("不存在该用户名")
		return
	}
	if !utils.VerifyPwd(user.Password, request.Password) {
		err = errors.New("账号密码错误")
		return
	}
	// 在redis中记录下对应用户的登陆token的key
	onlineUserKey := fmt.Sprintf(common.UseridMapToken, user.ID)
	newAccessToken := utils.GetRandomToken(32)
	sessionIdKey := fmt.Sprintf(common.SESSION, newAccessToken)

	// 如果当前用户已经登陆过，则删除覆盖原来的登陆信息
	token, err := common.RedisGetString(onlineUserKey)
	if err != nil {
		err = errors.New("系统异常")
		return
	}
	if string(token) != "" {
		oldSessionIdKey := fmt.Sprintf(common.SESSION, string(token))
		err = common.RedisDelString(oldSessionIdKey)
		if err != nil {
			err = errors.New(fmt.Sprintf("登出用户失败！access_token=%s", string(token)))
			return
		}
	}
	body, _ := json.Marshal(user)
	err = common.RedisSetString(sessionIdKey, body, config.RedisBaseValidTime*time.Second)
	if err != nil {
		err = errors.New("系统异常")
		return
	}
	err = common.RedisSetString(onlineUserKey, []byte(newAccessToken), config.RedisBaseValidTime*time.Second)
	if err != nil {
		err = errors.New("系统异常")
		return
	}
	reply.Code = config.SuccessReplyCode
	reply.AccessToken = newAccessToken
	return
}

func (s *ServerLogic) LoginOut(ctx context.Context, request *proto.LoginOutRequest) (reply *empty.Empty, err error) {
	reply = new(empty.Empty)
	accessToken := request.AccessToken
	sessionIdKey := fmt.Sprintf(common.SESSION, accessToken)
	res, err := common.RedisGetString(sessionIdKey)
	if err != nil {
		err = errors.New("系统异常")
		return
	}
	var user db.TUser
	_ = json.Unmarshal(res, &user)
	if user.ID == 0 {
		err = errors.New(fmt.Sprintf("获取该用户的信息失败，access_token=%s", accessToken))
		return
	}
	err = common.RedisDelString(fmt.Sprintf(common.UseridMapToken, user.ID))
	if err != nil {
		err = errors.New("系统异常")
		return
	}
	err = common.RedisDelString(fmt.Sprintf(common.SESSION, accessToken))
	if err != nil {
		err = errors.New("系统异常")
		return
	}
	return
}

func (s *ServerLogic) AfterLogin(ctx context.Context, request *proto.AfterLoginReq) (reply *proto.AfterLoginReply, err error) {
	reply = new(proto.AfterLoginReply)
	var (
		friendIdList    []int64
		groupIdList     []int64
		allOnlineUserId []int64
		res             []byte
	)
	res, err = common.RedisGetString(fmt.Sprintf(common.UserFriendList, request.Userid))
	if err != nil {
		err = errors.New("系统异常")
		return
	}
	_ = json.Unmarshal(res, &friendIdList)
	res, err = common.RedisGetString(fmt.Sprintf(common.UserGroupList, request.Userid))
	if err != nil {
		err = errors.New("系统异常")
		return
	}
	_ = json.Unmarshal(res, &groupIdList)
	var userMap map[string]string
	userMap, err = common.RedisHGetAll(common.AllOnlineUser)
	if err != nil {
		err = errors.New("系统异常")
		return
	}
	for K := range userMap {
		id, _ := strconv.Atoi(K)
		allOnlineUserId = append(allOnlineUserId, int64(id))
	}

	// 录入所有好友的最近聊天记录
	for _, id := range friendIdList {
		var userInfo db.TUser
		db.QueryUserById(id, &userInfo)
		msgList := make([]db.VFriendMessage, 0)
		db.QueryFriendMessageByPage(request.Userid, id, &msgList, 1, 16)
		tmp := &proto.FriendData{
			Userid:   userInfo.ID,
			Username: userInfo.Username,
			Tag:      userInfo.Tag,
			Role:     int32(userInfo.Role),
			Status:   int32(userInfo.Status),
			Avatar:   userInfo.Avatar,
			CreateAt: userInfo.CreateAt.Format(time.RFC3339),
		}
		for _, m := range msgList {
			mtmp := &proto.ChatMessage{
				Id:           m.ID,
				Userid:       m.Userid,
				FriendId:     m.FriendId,
				Content:      m.Content,
				MessageType:  m.MessageType,
				CreateAt:     m.CreateAt.Format(time.RFC3339),
				FromUsername: m.FromUsername,
				FriendName:   m.FriendName,
				Avatar:       m.Avatar,
				SnowId:       m.SnowId,
			}
			tmp.Messages = append(tmp.Messages, mtmp)
		}
		reply.FriendData = append(reply.FriendData, tmp)
	}

	// 录入所有群聊的聊天记录
	for _, id := range groupIdList {
		var groupInfo db.TGroup
		db.QueryGroupById(id, &groupInfo)
		msgList := make([]db.VGroupMessage, 0)
		db.QueryGroupMessageByPage(id, &msgList, 1, 16)
		tmp := &proto.GroupData{
			GroupId:   groupInfo.ID,
			GroupName: groupInfo.GroupName,
			Notice:    groupInfo.Notice,
			Userid:    groupInfo.ID,
			CreateAt:  groupInfo.CreateAt.Format(time.RFC3339),
		}
		for _, m := range msgList {
			mtmp := &proto.ChatMessage{
				Id:           m.ID,
				Userid:       m.Userid,
				GroupId:      m.GroupId,
				Content:      m.Content,
				MessageType:  m.MessageType,
				CreateAt:     m.CreateAt.Format(time.RFC3339),
				FromUsername: m.FromUsername,
				GroupName:    m.GroupName,
				Avatar:       m.Avatar,
				SnowId:       m.SnowId,
			}
			tmp.Messages = append(tmp.Messages, mtmp)
		}
		reply.GroupData = append(reply.GroupData, tmp)
	}

	for _, id := range allOnlineUserId {
		var userInfo db.TUser
		db.QueryUserById(id, &userInfo)
		tmp := &proto.UserData{
			Userid:   userInfo.ID,
			Avatar:   userInfo.Avatar,
			Role:     int32(userInfo.Role),
			Status:   int32(userInfo.Status),
			Tag:      userInfo.Tag,
			Username: userInfo.Username,
			CreateAt: userInfo.CreateAt.Format(time.RFC3339),
		}
		reply.UserData = append(reply.UserData, tmp)
	}
	return reply, nil
}

func (s *ServerLogic) GetUserInfoByAccessToken(ctx context.Context, request *proto.GetUserInfoByAccessTokenRequest) (reply *proto.GetUserInfoByAccessTokenReply, err error) {
	reply = new(proto.GetUserInfoByAccessTokenReply)
	reply.Code = config.FailReplyCode
	sessionIdKey := fmt.Sprintf(common.SESSION, request.AccessToken)
	res, err := common.RedisGetString(sessionIdKey)
	if err != nil {
		err = errors.New("系统异常")
		return
	}
	if string(res) == "" {
		err = errors.New("不合法的access_token")
		return
	}
	err = json.Unmarshal(res, &reply.User)
	if err != nil {
		zlog.Error(err.Error())
	}
	reply.Code = config.SuccessReplyCode
	return reply, nil
}

func (s *ServerLogic) GetUserInfoByUserid(ctx context.Context, request *proto.GetUserInfoByUseridRequest) (reply *proto.GetUserInfoByUseridReply, err error) {
	reply = new(proto.GetUserInfoByUseridReply)
	reply.Code = config.FailReplyCode
	var user db.TUser
	db.QueryUserById(request.Userid, &user)
	if user.ID == 0 {
		err = errors.New("不存在该用户")
		return
	}
	reply.User = &proto.User{
		Id:       user.ID,
		Username: user.Username,
		Avatar:   user.Avatar,
		Role:     int32(user.Role),
		Status:   int32(user.Status),
		Tag:      user.Tag,
		CreateAt: user.CreateAt.Format(time.RFC3339),
		UpdateAt: user.UpdateAt.Format(time.RFC3339),
	}
	reply.Code = config.SuccessReplyCode
	return reply, nil
}

func (s *ServerLogic) UpdateUserInfo(ctx context.Context, request *proto.UpdateUserInfoRequest) (reply *proto.UpdateUserInfoReply, err error) {
	reply = new(proto.UpdateUserInfoReply)
	reply.Code = config.FailReplyCode
	user := &db.TUser{
		ID:       request.User.Id,
		Username: request.User.Username,
		Avatar:   request.User.Avatar,
		Role:     int(request.User.Role),
		Status:   int(request.User.Status),
		Tag:      request.User.Tag,
	}
	db.UpdateUser(user)
	if user.ID == 0 {
		err = errors.New("不存在该用户")
		return
	}
	reply.User = &proto.User{
		Id:       user.ID,
		Username: user.Username,
		Avatar:   user.Avatar,
		Role:     int32(user.Role),
		Status:   int32(user.Status),
		Tag:      user.Tag,
		CreateAt: user.CreateAt.Format(time.RFC3339),
		UpdateAt: user.UpdateAt.Format(time.RFC3339),
	}
	reply.Code = config.SuccessReplyCode
	return reply, nil
}

func (s *ServerLogic) UpdatePassword(ctx context.Context, request *proto.UpdatePasswordRequest) (reply *empty.Empty, err error) {
	reply = new(empty.Empty)
	var user db.TUser
	if request.NewPassword2 != request.NewPassword1 {
		err = errors.New("输入的新密码不一致")
		return
	}
	db.QueryUserById(request.Userid, &user)
	if user.ID == 0 {
		err = errors.New("不存在该用户")
		return
	}
	if !utils.VerifyPwd(user.Password, request.OldPassword) {
		err = errors.New("密码不正确")
		return
	}
	pwd := utils.EncPassword(request.NewPassword1)
	if pwd == "" {
		err = errors.New("系统异常")
		return
	}
	err = db.UpdateUserPassword(user.ID, pwd)
	if err != nil {
		err = errors.New("系统异常")
		return
	}
	return
}

func (s *ServerLogic) SearchUser(ctx context.Context, request *proto.SearchUserRequest) (reply *proto.SearchUserReply, err error) {
	reply = new(proto.SearchUserReply)
	var userList []db.TUser
	db.SearchUserByUsername(request.Username, &userList)
	for _, val := range userList {
		reply.UserList = append(reply.UserList, &proto.User{
			Id:       val.ID,
			Username: val.Username,
			Avatar:   val.Avatar,
			Role:     int32(val.Role),
			Status:   int32(val.Status),
			Tag:      val.Tag,
			CreateAt: val.CreateAt.Format(time.RFC3339),
			UpdateAt: val.UpdateAt.Format(time.RFC3339),
		})
	}
	return
}

func (s *ServerLogic) SearchGroup(ctx context.Context, request *proto.SearchGroupRequest) (reply *proto.SearchGroupReply, err error) {
	reply = new(proto.SearchGroupReply)
	var groupList []db.TGroup
	db.SearchGroupByGroupName(request.GroupName, &groupList)
	for _, val := range groupList {
		reply.GroupList = append(reply.GroupList, &proto.Group{
			Id:        val.ID,
			Userid:    val.ID,
			GroupName: val.GroupName,
			Notice:    val.Notice,
			CreateAt:  val.CreateAt.Format(time.RFC3339),
			UpdateAt:  val.UpdateAt.Format(time.RFC3339),
		})
	}
	return
}

func (s *ServerLogic) GetGroupMsgByPage(ctx context.Context, request *proto.GetGroupMsgByPageRequest) (reply *proto.GetGroupMsgByPageReply, err error) {
	reply = new(proto.GetGroupMsgByPageReply)
	var msgList []db.VGroupMessage
	var userList []db.TUser
	reply.Code = config.FailReplyCode
	db.QueryGroupMessageByPage(request.GroupId, &msgList, int(request.Current), int(request.PageSize))
	db.QueryGroupAllUser(request.GroupId, &userList)
	reply.Code = config.SuccessReplyCode
	for _, val := range msgList {
		reply.MessageArr = append(reply.MessageArr, &proto.ChatMessage{
			Id:           val.ID,
			Userid:       val.Userid,
			GroupId:      val.GroupId,
			Content:      val.Content,
			MessageType:  val.MessageType,
			CreateAt:     val.CreateAt.Format(time.RFC3339),
			FromUsername: val.FromUsername,
			GroupName:    val.GroupName,
			Avatar:       val.Avatar,
			SnowId:       val.SnowId,
		})
	}
	for _, val := range userList {
		reply.UserArr = append(reply.UserArr, &proto.User{
			Id:       val.ID,
			Username: val.Username,
			Avatar:   val.Avatar,
			Role:     int32(val.Role),
			Status:   int32(val.Status),
			Tag:      val.Tag,
			CreateAt: val.CreateAt.Format(time.RFC3339),
			UpdateAt: val.UpdateAt.Format(time.RFC3339),
		})
	}
	return
}

func (s *ServerLogic) GetFriendMsgByPage(ctx context.Context, request *proto.GetFriendMsgByPageRequest) (reply *proto.GetFriendMsgByPageReply, err error) {
	reply = new(proto.GetFriendMsgByPageReply)
	var msgList []db.VFriendMessage
	reply.Code = config.FailReplyCode
	db.QueryFriendMessageByPage(request.Userid, request.FriendId, &msgList, int(request.Current), int(request.PageSize))
	reply.Code = config.SuccessReplyCode
	for _, val := range msgList {
		reply.MessageArr = append(reply.MessageArr, &proto.ChatMessage{
			Id:           val.ID,
			Userid:       val.Userid,
			FriendId:     val.FriendId,
			Content:      val.Content,
			MessageType:  val.MessageType,
			CreateAt:     val.CreateAt.Format(time.RFC3339),
			FriendName:   val.FriendName,
			Avatar:       val.Avatar,
			FromUsername: val.FromUsername,
			SnowId:       val.SnowId,
		})
	}
	return
}

func (s *ServerLogic) CreateGroup(ctx context.Context, request *proto.Group) (reply *proto.Group, err error) {
	reply = new(proto.Group)
	var oldGroup db.TGroup
	db.QueryGroupByGroupName(request.GroupName, &oldGroup)
	if oldGroup.ID != 0 {
		err = errors.New("该群聊名称已经被使用")
		return
	}
	newGroup := &db.TGroup{
		Userid:    request.Userid,
		GroupName: request.GroupName,
		Notice:    request.GroupName,
	}
	db.CreateGroup(newGroup)
	if newGroup.ID == 0 {
		err = errors.New("系统异常，创建群聊失败")
		return
	}
	return &proto.Group{
		Id:        newGroup.ID,
		Userid:    newGroup.Userid,
		GroupName: newGroup.GroupName,
		Notice:    newGroup.Notice,
		CreateAt:  newGroup.CreateAt.Format(time.RFC3339),
		UpdateAt:  newGroup.UpdateAt.Format(time.RFC3339),
	}, nil
}

func (s *ServerLogic) AddGroup(ctx context.Context, request *proto.AddGroupRequest) (reply *empty.Empty, err error) {
	reply = new(empty.Empty)
	// 获取当前用户的serverId
	var serverIdByte []byte
	serverIdByte, err = common.RedisGetString(fmt.Sprintf(common.UseridMapServerId, request.Userid))
	if err != nil {
		err = errors.New("系统异常，添加群聊失败")
		return
	}
	var user db.TUser
	db.QueryUserById(request.Userid, &user)
	relation := &db.TRelation{
		Type:    "group",
		ObjectA: request.Userid,
		ObjectB: request.GroupId,
	}
	db.CreateRelation(relation)
	if relation.ID == 0 {
		err = errors.New("系统异常，添加群聊失败")
		return
	}

	// 向群聊的topic推送新群聊状态变更的消息
	groupIdList := db.QueryUserAllGroupId(user.ID)

	// 更新该用户加入群聊的所有在redis中信息
	var has bool
	has, err = common.RedisIsNotExistHSet(fmt.Sprintf(common.GroupOnlineUser, request.GroupId), fmt.Sprintf("%d", user.ID), string(serverIdByte))
	if err != nil {
		err = errors.New("系统异常")
		return
	}
	// 之前不在线的用户上线时才更新群聊在线人数
	if !has {
		// 更新hash table中对应群聊项的在线人数
		err = common.RedisHINCRBY(common.GroupOnlineUserCount, fmt.Sprintf("%d", request.GroupId), 1)
		if err != nil {
			err = errors.New("系统异常")
			return
		}
	}
	//往群聊对应topic写入群聊在线人数更新信息 TODO:need to test
	var userList []db.TUser
	db.QueryGroupAllUser(request.GroupId, &userList)
	countStr, _ := common.RedisHGet(common.GroupOnlineUserCount, fmt.Sprintf("%d", request.GroupId))
	count, _ := strconv.Atoi(countStr)
	var onlineUserIdList []int64
	// 获取当前群聊的所有在线用户的id
	var userMap map[string]string
	userMap, err = common.RedisHGetAll(fmt.Sprintf(common.GroupOnlineUser, request.GroupId))
	if err != nil {
		err = errors.New("系统异常")
		return
	}
	for k := range userMap {
		id, _ := strconv.Atoi(k)
		onlineUserIdList = append(onlineUserIdList, int64(id))
	}
	err = PushGroupInfo(user.ID, request.GroupId, count, userList, onlineUserIdList)
	if err != nil {
		err = errors.New("系统异常")
		return
	}
	// 在redis中写入当前用户的加入的所有群聊id
	b, _ := json.Marshal(groupIdList)
	err = common.RedisSetString(fmt.Sprintf(common.UserGroupList, user.ID), b, 0)
	if err != nil {
		err = errors.New("系统异常")
		return
	}
	return
}

func (s *ServerLogic) AddFriend(ctx context.Context, request *proto.AddFriendRequest) (reply *empty.Empty, err error) {
	reply = new(empty.Empty)
	relation := &db.TRelation{
		Type:    "friend",
		ObjectA: request.Userid,
		ObjectB: request.FriendId,
	}
	db.CreateRelation(relation)
	if relation.ID == 0 {
		err = errors.New("系统异常，添加群聊失败")
		return
	}
	return
}

func (s *ServerLogic) Push(ctx context.Context, request *proto.PushRequest) (reply *empty.Empty, err error) {
	reply = new(empty.Empty)
	payload := common.FriendMsg{
		Userid:       request.Msg.Userid,
		FriendId:     request.Msg.FriendId,
		Content:      request.Msg.Content,
		MessageType:  request.Msg.MessageType,
		CreateAt:     request.Msg.CreateAt,
		FriendName:   request.Msg.FriendName,
		FromUsername: request.Msg.FromUsername,
		Avatar:       request.Msg.Avatar,
		Op:           common.OpFriendMsgSend,
		Watermark:    request.Msg.Watermark,
	}
	// 由于采用写扩散的机制，所以要同时往发送和接收方的topic中写入消息
	payload.Belong = request.Msg.FriendId
	err = Push(request.Msg.FriendId, payload, common.OpFriendMsgSend)
	if err != nil {
		err = errors.New("系统异常")
		return
	}
	payload.Belong = request.Msg.Userid
	err = Push(request.Msg.Userid, payload, common.OpFriendMsgSend)
	if err != nil {
		err = errors.New("系统异常")
		return
	}
	return reply, nil
}

func (s *ServerLogic) PushRoom(ctx context.Context, request *proto.PushRoomRequest) (reply *empty.Empty, err error) {
	reply = new(empty.Empty)
	payload := common.GroupMsg{
		Userid:       request.Msg.Userid,
		GroupId:      request.Msg.GroupId,
		Content:      request.Msg.Content,
		MessageType:  request.Msg.MessageType,
		CreateAt:     request.Msg.CreateAt,
		GroupName:    request.Msg.GroupName,
		FromUsername: request.Msg.FromUsername,
		Avatar:       request.Msg.Avatar,
		Op:           common.OpGroupMsgSend,
		Watermark:    request.Msg.Watermark,
	}
	err = Push(request.Msg.GroupId, payload, common.OpGroupMsgSend)
	if err != nil {
		err = errors.New("系统异常")
		return
	}
	return reply, nil
}

func (s *ServerLogic) PushRoomCount(ctx context.Context, request *proto.PushRoomCountRequest) (reply *empty.Empty, err error) {
	reply = new(empty.Empty)
	countStr, _ := common.RedisHGet(common.GroupOnlineUserCount, fmt.Sprintf("%d", request.GroupId))
	count, _ := strconv.Atoi(countStr)
	// TODO:往该群聊的的topic中生产消息
	err = PushGroupCount(request.GroupId, count)
	if err != nil {
		err = errors.New("系统异常")
		return
	}
	return reply, nil
}

func (s *ServerLogic) PushRoomInfo(ctx context.Context, request *proto.PushRoomInfoRequest) (reply *empty.Empty, err error) {
	reply = new(empty.Empty)
	var userList []db.TUser
	db.QueryGroupAllUser(request.GroupId, &userList)
	countStr, _ := common.RedisHGet(common.GroupOnlineUserCount, fmt.Sprintf("%d", request.GroupId))
	count, _ := strconv.Atoi(countStr)
	var onlineUserIdList []int64
	// 获取当前群聊的所有在线用户的id
	var userMap map[string]string
	userMap, err = common.RedisHGetAll(fmt.Sprintf(common.GroupOnlineUser, request.GroupId))
	if err != nil {
		err = errors.New("系统异常")
		return
	}
	for k := range userMap {
		id, _ := strconv.Atoi(k)
		onlineUserIdList = append(onlineUserIdList, int64(id))
	}
	// TODO:往该群聊的的topic中生产消息
	err = PushGroupInfo(0, request.GroupId, count, userList, onlineUserIdList)
	if err != nil {
		err = errors.New("系统异常")
		return
	}
	return reply, nil
}
