package logic

import (
	"axisChat/common"
	"axisChat/config"
	"axisChat/db"
	"axisChat/logic/proto"
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

type Server struct{}

// Connect 当客户端通过connect层连接服务时，调用该rpc方法通过查看redis中有没有该用户的登陆信息，从而达到鉴权的目的，如果是
// 私聊的话将该用户的上线信息写入其好友的所有MQ中，如果是群聊的话，将该群的人数信息推送到群聊对应的MQ中
// 用户上线利用消息队列主动提醒
func (s *Server) Connect(ctx context.Context, request *proto.ConnectRequest) (reply *proto.ConnectReply, err error) {
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
	var user db.User
	_ = json.Unmarshal(res, &user)
	// TODO：通过serverId查找当前connect的用户在connect层的哪一个服务微服务中
	serverIdKey := fmt.Sprintf(common.UseridMapServerId, user.ID)
	err = common.RedisSetString(serverIdKey, []byte(request.ServerId), 0)
	if err != nil {
		err = errors.New("系统异常")
		return
	}
	// 在通知该用户加入的群聊、所有加好友，其的上线消息
	var groupIdList []int64
	var friendIdList []int64
	db.QueryUserAllGroupId(user.ID, &groupIdList)
	db.QueryUserAllFriendId(user.ID, &friendIdList)
	// 更新该用户加入群聊的所有在redis中信息
	for _, val := range groupIdList {
		err := common.RedisSetSet(fmt.Sprintf(common.GroupOnlineUser, val), fmt.Sprintf("%d", user.ID), user.Username)
		if err != nil {
			err = errors.New("系统异常")
			return
		}
		err = common.RedisInc(fmt.Sprintf(common.GroupOnlineUserCount, val))
		if err != nil {
			err = errors.New("系统异常")
			return
		}
		//往群聊对应topic写入群聊在线人数更新信息 TODO:need to test
		var userList []db.User
		db.QueryGroupAllUser(val, &userList)
		countStr, _ := common.RedisGetString(fmt.Sprintf(common.GroupOnlineUserCount, val))
		count, _ := strconv.Atoi(string(countStr))
		err = PushGroupInfo(user.ID, val, count, userList)
		if err != nil {
			err = errors.New("系统异常")
			return
		}
	}
	// 往好友对应topic写入该用户上线信息
	for _, val := range friendIdList {
		err = PushUserInfo(user.ID, user.Username, val, OpFriendOnlineSend)
		if err != nil {
			err = errors.New("系统异常")
			return
		}
	}
	return &proto.ConnectReply{
		Userid: user.ID,
	}, nil
}

// DisConnect 当connect层检测到对应客户端下线后，更新redis中该用户的在线信息，并在好友和其所在群聊的MQ中写入其下线的信息
// 下也利用MQ主动提醒
func (s *Server) DisConnect(ctx context.Context, request *proto.DisConnectRequest) (reply *empty.Empty, err error) {
	// 在通知该用户加入的群聊、所有加好友，其的上线消息
	var groupIdList []int64
	var friendIdList []int64
	db.QueryUserAllGroupId(request.Userid, &groupIdList)
	db.QueryUserAllFriendId(request.Userid, &friendIdList)
	// 更新该用户加入群聊的所有在redis中信息
	for _, val := range groupIdList {
		err := common.RedisHDel(fmt.Sprintf(common.GroupOnlineUser, val), fmt.Sprintf("%d", request.Userid))
		if err != nil {
			err = errors.New("系统异常")
			return
		}
		err = common.RedisDecr(fmt.Sprintf(common.GroupOnlineUserCount, val))
		if err != nil {
			err = errors.New("系统异常")
			return
		}
		//往群聊对应topic写入群聊在线人数更新信息 TODO:need to test
		var userList []db.User
		db.QueryGroupAllUser(val, &userList)
		countStr, _ := common.RedisGetString(fmt.Sprintf(common.GroupOnlineUserCount, val))
		count, _ := strconv.Atoi(string(countStr))
		// TODO:往该群聊的的topic中生产消息
		err = PushGroupInfo(request.Userid, val, count, userList)
		if err != nil {
			err = errors.New("系统异常")
			return
		}
	}
	// 往好友对应topic写入该用户上线信息
	for _, val := range friendIdList {
		// TODO:往该好友的的topic中生产消息
		err = PushUserInfo(request.Userid, "", val, OPFriendOffOnlineSend)
		if err != nil {
			err = errors.New("系统异常")
			return
		}
	}
	return reply, nil
}

func (s *Server) Register(ctx context.Context, request *proto.RegisterRequest) (reply *proto.RegisterReply, err error) {
	reply.Code = config.FailReplyCode
	user := new(db.User)
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
	return
}

func (s *Server) Login(ctx context.Context, request *proto.LoginRequest) (reply *proto.LoginReply, err error) {
	reply.Code = config.FailReplyCode
	user := new(db.User)
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
		err := common.RedisDelString(oldSessionIdKey)
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

func (s *Server) LoginOut(ctx context.Context, request *proto.LoginOutRequest) (reply *empty.Empty, err error) {
	accessToken := request.AccessToken
	sessionIdKey := fmt.Sprintf(common.SESSION, accessToken)
	res, err := common.RedisGetString(sessionIdKey)
	if err != nil {
		err = errors.New("系统异常")
		return
	}
	var user db.User
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

func (s *Server) GetUserInfoByAccessToken(ctx context.Context, request *proto.GetUserInfoByAccessTokenRequest) (reply *proto.GetUserInfoByAccessTokenReply, err error) {
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
	_ = json.Unmarshal(res, &reply.User)
	reply.Code = config.SuccessReplyCode
	return reply, nil
}

func (s *Server) GetUserInfoByUserid(ctx context.Context, request *proto.GetUserInfoByUseridRequest) (reply *proto.GetUserInfoByUseridReply, err error) {
	reply.Code = config.FailReplyCode
	var user db.User
	db.QueryUserById(request.Userid, &user)
	if user.ID == 0 {
		err = errors.New("不存在该用户")
		return
	}
	reply.User = &proto.User{
		Userid:   user.ID,
		Username: user.Username,
		Avatar:   user.Avatar,
		Role:     int32(user.Role),
		Status:   int32(user.Status),
		Tag:      user.Tag,
		CreateAt: user.CreateAt.Unix(),
		UpdateAt: user.UpdateAt.Unix(),
	}
	reply.Code = config.SuccessReplyCode
	return reply, nil
}

func (s *Server) UpdateUserInfo(ctx context.Context, request *proto.UpdateUserInfoRequest) (reply *proto.UpdateUserInfoReply, err error) {
	reply.Code = config.FailReplyCode
	user := &db.User{
		ID:       request.User.Userid,
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
		Userid:   user.ID,
		Username: user.Username,
		Avatar:   user.Avatar,
		Role:     int32(user.Role),
		Status:   int32(user.Status),
		Tag:      user.Tag,
		CreateAt: user.CreateAt.Unix(),
		UpdateAt: user.UpdateAt.Unix(),
	}
	reply.Code = config.SuccessReplyCode
	return reply, nil
}

func (s *Server) UpdatePassword(ctx context.Context, request *proto.UpdatePasswordRequest) (reply *empty.Empty, err error) {
	var user db.User
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

func (s *Server) SearchUser(ctx context.Context, request *proto.SearchUserRequest) (reply *proto.SearchUserReply, err error) {
	var userList []db.User
	db.SearchUserByUsername(request.Username, &userList)
	for _, val := range userList {
		reply.UserList = append(reply.UserList, &proto.User{
			Userid:   val.ID,
			Username: val.Username,
			Avatar:   val.Avatar,
			Role:     int32(val.Role),
			Status:   int32(val.Status),
			Tag:      val.Tag,
			CreateAt: val.CreateAt.Unix(),
			UpdateAt: val.UpdateAt.Unix(),
		})
	}
	return
}

func (s *Server) SearchGroup(ctx context.Context, request *proto.SearchGroupRequest) (reply *proto.SearchGroupReply, err error) {
	var groupList []db.Group
	db.SearchGroupByGroupName(request.GroupName, &groupList)
	for _, val := range groupList {
		reply.GroupList = append(reply.GroupList, &proto.Group{
			GroupId:   val.ID,
			Userid:    val.ID,
			GroupName: val.GroupName,
			Notice:    val.Notice,
			CreateAt:  val.CreateAt.Unix(),
			UpdateAt:  val.UpdateAt.Unix(),
		})
	}
	return
}

func (s *Server) GetGroupMsgByPage(ctx context.Context, request *proto.GetGroupMsgByPageRequest) (reply *proto.GetGroupMsgByPageReply, err error) {
	var msgList *[]db.Message
	var userList *[]db.User
	reply.Code = config.FailReplyCode
	db.QueryGroupMessageByPage(request.Userid, request.GroupId, msgList, int(request.Current), int(request.PageSize))
	db.QueryGroupAllUser(request.GroupId, userList)
	reply.Code = config.SuccessReplyCode
	for _, val := range *msgList {
		reply.MessageArr = append(reply.MessageArr, &proto.ChatMessage{
			Id:          val.ID,
			Userid:      val.FromA,
			GroupId:     val.ToB,
			Content:     val.Content,
			MessageType: val.MessageType,
			CreateAt:    val.CreateAt.Unix(),
			FriendName:  "", //TODO:need create a views
			GroupName:   "",
		})
	}
	for _, val := range *userList {
		reply.UserArr = append(reply.UserArr, &proto.User{
			Userid:   val.ID,
			Username: val.Username,
			Avatar:   val.Avatar,
			Role:     int32(val.Role),
			Status:   int32(val.Status),
			Tag:      val.Tag,
			CreateAt: val.CreateAt.Unix(),
			UpdateAt: val.UpdateAt.Unix(),
		})
	}
	return
}

func (s *Server) GetFriendMsgByPage(ctx context.Context, request *proto.GetFriendMsgByPageRequest) (reply *proto.GetFriendMsgByPageReply, err error) {
	var msgList *[]db.Message
	reply.Code = config.FailReplyCode
	db.QueryFriendMessageByPage(request.Userid, request.FriendId, msgList, int(request.Current), int(request.PageSize))
	reply.Code = config.SuccessReplyCode
	for _, val := range *msgList {
		reply.MessageArr = append(reply.MessageArr, &proto.ChatMessage{
			Id:          val.ID,
			Userid:      val.FromA,
			GroupId:     val.ToB,
			Content:     val.Content,
			MessageType: val.MessageType,
			CreateAt:    val.CreateAt.Unix(),
			FriendName:  "", //TODO:need create a views
			GroupName:   "",
		})
	}
	return
}

func (s *Server) CreateGroup(ctx context.Context, request *proto.Group) (reply *proto.Group, err error) {
	var oldGroup db.Group
	db.QueryGroupByGroupName(request.GroupName, &oldGroup)
	if oldGroup.ID != 0 {
		err = errors.New("该群聊名称已经被使用")
		return
	}
	newGroup := &db.Group{
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
		GroupId:   newGroup.ID,
		Userid:    newGroup.Userid,
		GroupName: newGroup.GroupName,
		Notice:    newGroup.Notice,
		CreateAt:  newGroup.CreateAt.Unix(),
		UpdateAt:  newGroup.UpdateAt.Unix(),
	}, nil
}

func (s *Server) AddGroup(ctx context.Context, request *proto.AddGroupRequest) (reply *empty.Empty, err error) {
	relation := &db.Relation{
		Type:    "group",
		ObjectA: request.Userid,
		ObjectB: request.GroupId,
	}
	db.CreateRelation(relation)
	if relation.ID == 0 {
		err = errors.New("系统异常，添加群聊失败")
		return
	}
	return
}

func (s *Server) AddFriend(ctx context.Context, request *proto.AddFriendRequest) (reply *empty.Empty, err error) {
	relation := &db.Relation{
		Type:    "group",
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

func (s *Server) Push(ctx context.Context, request *proto.PushRequest) (reply *empty.Empty, err error) {
	payload := FriendMsg{
		Userid:       request.Msg.Userid,
		FriendId:     request.Msg.FriendId,
		Content:      request.Msg.Content,
		MessageType:  request.Msg.MessageType,
		CreateAt:     request.Msg.CreateAt,
		FriendName:   request.Msg.FriendName,
		FromUsername: request.Msg.FromUsername,
	}
	err = Push(request.Msg.FriendId, payload, OpFriendMsgSend)
	if err != nil {
		err = errors.New("系统异常")
		return
	}
	return reply, nil
}

func (s *Server) PushRoom(ctx context.Context, request *proto.PushRoomRequest) (reply *empty.Empty, err error) {
	payload := GroupMsg{
		Userid:       request.Msg.Userid,
		GroupId:      request.Msg.GroupId,
		Content:      request.Msg.Content,
		MessageType:  request.Msg.MessageType,
		CreateAt:     request.Msg.CreateAt,
		GroupName:    request.Msg.GroupName,
		FromUsername: request.Msg.FromUsername,
	}
	err = Push(request.Msg.GroupId, payload, OpGroupMsgSend)
	if err != nil {
		err = errors.New("系统异常")
		return
	}
	return reply, nil
}

func (s *Server) PushRoomCount(ctx context.Context, request *proto.PushRoomCountRequest) (reply *empty.Empty, err error) {
	countStr, _ := common.RedisGetString(fmt.Sprintf(common.GroupOnlineUserCount, request.GroupId))
	count, _ := strconv.Atoi(string(countStr))
	// TODO:往该群聊的的topic中生产消息
	err = PushGroupCount(request.GroupId, count)
	if err != nil {
		err = errors.New("系统异常")
		return
	}
	return reply, nil
}

func (s *Server) PushRoomInfo(ctx context.Context, request *proto.PushRoomInfoRequest) (reply *empty.Empty, err error) {
	var userList []db.User
	db.QueryGroupAllUser(request.GroupId, &userList)
	countStr, _ := common.RedisGetString(fmt.Sprintf(common.GroupOnlineUserCount, request.GroupId))
	count, _ := strconv.Atoi(string(countStr))
	// TODO:往该群聊的的topic中生产消息
	err = PushGroupInfo(0, request.GroupId, count, userList)
	if err != nil {
		err = errors.New("系统异常")
		return
	}
	return reply, nil
}
