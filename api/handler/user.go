package handler

import (
	"axisChat/api/rpc"
	"axisChat/api/utils"
	"axisChat/db"
	"axisChat/proto"
	"axisChat/utils/zlog"
	"context"
	"github.com/gin-gonic/gin"
	"github.com/gin-gonic/gin/binding"
	"time"
)

type registerReq struct {
	Username string `json:"username" binding:"required"`
	Password string `json:"password" binding:"required"`
}

func Register(ctx *gin.Context) {
	var form registerReq
	if err := ctx.ShouldBindBodyWith(&form, binding.JSON); err != nil {
		zlog.Error(err.Error())
		utils.FailWithMsg(ctx, "参数校验失败")
		return
	}
	ins, err := rpc.GetLogicRpcInstance()
	if err != nil {
		utils.ResponseWithCode(ctx, utils.CodeUnknownError, nil, nil)
		return
	}
	client := proto.NewLogicClient(ins.Conn)
	_ctx, cancel := context.WithTimeout(context.Background(), time.Second*5)
	defer cancel()
	reply, err := client.Register(_ctx, &proto.RegisterRequest{
		Username: form.Username,
		Password: form.Password,
	})
	if err != nil {
		zlog.Error(err.Error())
		utils.ResponseWithCode(ctx, utils.CodeUnknownError, nil, nil)
		return
	}
	utils.ResponseWithCode(ctx, int(reply.Code), nil, reply.AccessToken)
}

type loginReq struct {
	Username string `json:"username" binding:"required"`
	Password string `json:"password" binding:"required"`
}

func Login(ctx *gin.Context) {
	var form loginReq
	if err := ctx.ShouldBindBodyWith(&form, binding.JSON); err != nil {
		zlog.Error(err.Error())
		utils.FailWithMsg(ctx, "参数校验失败")
		return
	}
	ins, err := rpc.GetLogicRpcInstance()
	if err != nil {
		utils.ResponseWithCode(ctx, utils.CodeUnknownError, nil, nil)
		return
	}
	client := proto.NewLogicClient(ins.Conn)
	_ctx, cancel := context.WithTimeout(context.Background(), time.Second*5)
	defer cancel()
	reply, err := client.Login(_ctx, &proto.LoginRequest{
		Username: form.Username,
		Password: form.Password,
	})
	if err != nil {
		zlog.Error(err.Error())
		utils.ResponseWithCode(ctx, utils.CodeUnknownError, nil, nil)
		return
	}
	utils.ResponseWithCode(ctx, int(reply.Code), nil, reply.AccessToken)
}

type loginOutReq struct {
	AccessToken string `json:"accessToken" binding:"required"`
}

func LoginOut(ctx *gin.Context) {
	var form loginOutReq
	if err := ctx.ShouldBindBodyWith(&form, binding.JSON); err != nil {
		zlog.Error(err.Error())
		utils.FailWithMsg(ctx, "参数校验失败")
		return
	}
	ins, err := rpc.GetLogicRpcInstance()
	if err != nil {
		utils.ResponseWithCode(ctx, utils.CodeUnknownError, nil, nil)
		return
	}
	client := proto.NewLogicClient(ins.Conn)
	_ctx, cancel := context.WithTimeout(context.Background(), time.Second*5)
	defer cancel()
	_, err = client.LoginOut(_ctx, &proto.LoginOutRequest{
		AccessToken: form.AccessToken,
	})
	if err != nil {
		zlog.Error(err.Error())
		utils.ResponseWithCode(ctx, utils.CodeUnknownError, nil, nil)
		return
	}
	utils.SuccessWithMsg(ctx, nil, nil)
}

type getUserInfoByAccessTokenReq struct {
	AccessToken string `json:"accessToken" binding:"required"`
}

func GetUserInfoByAccessToken(ctx *gin.Context) {
	var form getUserInfoByAccessTokenReq
	if err := ctx.ShouldBindBodyWith(&form, binding.JSON); err != nil {
		zlog.Error(err.Error())
		utils.FailWithMsg(ctx, "参数校验失败")
		return
	}
	ins, err := rpc.GetLogicRpcInstance()
	if err != nil {
		utils.ResponseWithCode(ctx, utils.CodeUnknownError, nil, nil)
		return
	}
	client := proto.NewLogicClient(ins.Conn)
	_ctx, cancel := context.WithTimeout(context.Background(), time.Second*5)
	defer cancel()
	reply, err := client.GetUserInfoByAccessToken(_ctx, &proto.GetUserInfoByAccessTokenRequest{
		AccessToken: form.AccessToken,
	})
	if err != nil {
		zlog.Error(err.Error())
		utils.ResponseWithCode(ctx, utils.CodeUnknownError, nil, nil)
		return
	}
	createAt, _ := time.ParseInLocation(time.RFC3339, reply.User.CreateAt, time.Local)
	data := db.TUser{
		ID:       reply.User.Id,
		Username: reply.User.Username,
		Avatar:   reply.User.Avatar,
		Role:     int(reply.User.Role),
		Status:   int(reply.User.Status),
		Tag:      reply.User.Tag,
		CreateAt: createAt,
	}
	utils.SuccessWithMsg(ctx, nil, data)
}

type getUserInfoByUseridReq struct {
	Userid int64 `json:"userid" binding:"required"`
}

func GetUserInfoByUserid(ctx *gin.Context) {
	var form getUserInfoByUseridReq
	if err := ctx.ShouldBindBodyWith(&form, binding.JSON); err != nil {
		zlog.Error(err.Error())
		utils.FailWithMsg(ctx, "参数校验失败")
		return
	}
	ins, err := rpc.GetLogicRpcInstance()
	if err != nil {
		utils.ResponseWithCode(ctx, utils.CodeUnknownError, nil, nil)
		return
	}
	client := proto.NewLogicClient(ins.Conn)
	_ctx, cancel := context.WithTimeout(context.Background(), time.Second*5)
	defer cancel()
	reply, err := client.GetUserInfoByUserid(_ctx, &proto.GetUserInfoByUseridRequest{
		Userid: form.Userid,
	})
	if err != nil {
		zlog.Error(err.Error())
		utils.ResponseWithCode(ctx, utils.CodeUnknownError, nil, nil)
		return
	}
	createAt, _ := time.ParseInLocation(time.RFC3339, reply.User.CreateAt, time.Local)
	data := db.TUser{
		ID:       reply.User.Id,
		Username: reply.User.Username,
		Avatar:   reply.User.Avatar,
		Role:     int(reply.User.Role),
		Status:   int(reply.User.Status),
		Tag:      reply.User.Tag,
		CreateAt: createAt,
	}
	utils.SuccessWithMsg(ctx, nil, data)
}

type updateUserInfo struct {
	Username string `json:"username" `
	Avatar   string `json:"avatar"`
	Role     int    `json:"role"`
	Status   int    `json:"status"`
	Tag      string `json:"tag"`
}

func UpdateUserInfo(ctx *gin.Context) {
	var form updateUserInfo
	if err := ctx.ShouldBindBodyWith(&form, binding.JSON); err != nil {
		zlog.Error(err.Error())
		utils.FailWithMsg(ctx, "参数校验失败")
		return
	}
	userid, ok := ctx.Get("userid")
	if !ok {
		utils.ResponseWithCode(ctx, utils.CodeSessionError, nil, nil)
		return
	}
	ins, err := rpc.GetLogicRpcInstance()
	if err != nil {
		utils.ResponseWithCode(ctx, utils.CodeUnknownError, nil, nil)
		return
	}
	client := proto.NewLogicClient(ins.Conn)
	_ctx, cancel := context.WithTimeout(context.Background(), time.Second*5)
	defer cancel()
	reply, err := client.UpdateUserInfo(_ctx, &proto.UpdateUserInfoRequest{
		User: &proto.User{
			Id:       userid.(int64),
			Username: form.Username,
			Avatar:   form.Avatar,
			Role:     int32(form.Role),
			Status:   int32(form.Status),
			Tag:      form.Tag,
		},
	})
	if err != nil {
		zlog.Error(err.Error())
		utils.ResponseWithCode(ctx, utils.CodeUnknownError, nil, nil)
		return
	}
	createAt, _ := time.ParseInLocation(time.RFC3339, reply.User.CreateAt, time.Local)
	updateAt, _ := time.ParseInLocation(time.RFC3339, reply.User.UpdateAt, time.Local)
	data := db.TUser{
		ID:       reply.User.Id,
		Username: reply.User.Username,
		Avatar:   reply.User.Avatar,
		Role:     int(reply.User.Role),
		Status:   int(reply.User.Status),
		Tag:      reply.User.Tag,
		CreateAt: createAt,
		UpdateAt: updateAt,
	}
	utils.SuccessWithMsg(ctx, nil, data)
}

type updatePasswordReq struct {
	OldPassword  string `json:"oldPassword" binding:"required"`
	NewPassword1 string `json:"newPassword1" binding:"required"`
	NewPassword2 string `json:"newPassword2" binding:"required"`
}

func UpdatePassword(ctx *gin.Context) {
	var form updatePasswordReq
	if err := ctx.ShouldBindBodyWith(&form, binding.JSON); err != nil {
		zlog.Error(err.Error())
		utils.FailWithMsg(ctx, "参数校验失败")
		return
	}
	userid, ok := ctx.Get("userid")
	if !ok {
		utils.ResponseWithCode(ctx, utils.CodeSessionError, nil, nil)
		return
	}
	ins, err := rpc.GetLogicRpcInstance()
	if err != nil {
		utils.ResponseWithCode(ctx, utils.CodeUnknownError, nil, nil)
		return
	}
	client := proto.NewLogicClient(ins.Conn)
	_ctx, cancel := context.WithTimeout(context.Background(), time.Second*5)
	defer cancel()
	_, err = client.UpdatePassword(_ctx, &proto.UpdatePasswordRequest{
		Userid:       userid.(int64),
		OldPassword:  form.OldPassword,
		NewPassword1: form.NewPassword1,
		NewPassword2: form.NewPassword2,
	})
	if err != nil {
		zlog.Error(err.Error())
		utils.ResponseWithCode(ctx, utils.CodeUnknownError, nil, nil)
		return
	}
	utils.SuccessWithMsg(ctx, nil, nil)
}

type searchUserReq struct {
	Username string `json:"username" binding:"required"`
}

func SearchUser(ctx *gin.Context) {
	var form searchUserReq
	if err := ctx.ShouldBindBodyWith(&form, binding.JSON); err != nil {
		zlog.Error(err.Error())
		utils.FailWithMsg(ctx, "参数校验失败")
		return
	}
	ins, err := rpc.GetLogicRpcInstance()
	if err != nil {
		utils.ResponseWithCode(ctx, utils.CodeUnknownError, nil, nil)
		return
	}
	client := proto.NewLogicClient(ins.Conn)
	_ctx, cancel := context.WithTimeout(context.Background(), time.Second*5)
	defer cancel()
	reply, err := client.SearchUser(_ctx, &proto.SearchUserRequest{
		Username: form.Username,
	})
	if err != nil {
		zlog.Error(err.Error())
		utils.ResponseWithCode(ctx, utils.CodeUnknownError, nil, nil)
		return
	}
	utils.SuccessWithMsg(ctx, nil, reply)
}

type getFriendMsgByPageReq struct {
	FriendId int64 `json:"friendId" binding:"required"`
	Current  int64 `json:"current" binding:"required"`
	PageSize int64 `json:"pageSize" binding:"required"`
}

func GetFriendMsgByPage(ctx *gin.Context) {
	var form getFriendMsgByPageReq
	if err := ctx.ShouldBindBodyWith(&form, binding.JSON); err != nil {
		zlog.Error(err.Error())
		utils.FailWithMsg(ctx, "参数校验失败")
		return
	}
	userid, ok := ctx.Get("userid")
	if !ok {
		utils.ResponseWithCode(ctx, utils.CodeSessionError, nil, nil)
		return
	}
	ins, err := rpc.GetLogicRpcInstance()
	if err != nil {
		utils.ResponseWithCode(ctx, utils.CodeUnknownError, nil, nil)
		return
	}
	client := proto.NewLogicClient(ins.Conn)
	_ctx, cancel := context.WithTimeout(context.Background(), time.Second*5)
	defer cancel()
	reply, err := client.GetFriendMsgByPage(_ctx, &proto.GetFriendMsgByPageRequest{
		FriendId: form.FriendId,
		Userid:   userid.(int64),
		Current:  form.Current,
		PageSize: form.PageSize,
	})
	if err != nil {
		zlog.Error(err.Error())
		utils.ResponseWithCode(ctx, utils.CodeUnknownError, nil, nil)
		return
	}
	utils.SuccessWithMsg(ctx, nil, reply)
}

type addFriend struct {
	FriendId int64 `json:"friendId" binding:"required"`
}

func AddFriend(ctx *gin.Context) {
	var form addFriend
	if err := ctx.ShouldBindBodyWith(&form, binding.JSON); err != nil {
		zlog.Error(err.Error())
		utils.FailWithMsg(ctx, "参数校验失败")
		return
	}
	userid, ok := ctx.Get("userid")
	if !ok {
		utils.ResponseWithCode(ctx, utils.CodeSessionError, nil, nil)
		return
	}
	ins, err := rpc.GetLogicRpcInstance()
	if err != nil {
		utils.ResponseWithCode(ctx, utils.CodeUnknownError, nil, nil)
		return
	}
	client := proto.NewLogicClient(ins.Conn)
	_ctx, cancel := context.WithTimeout(context.Background(), time.Second*5)
	defer cancel()
	_, err = client.AddFriend(_ctx, &proto.AddFriendRequest{
		FriendId: form.FriendId,
		Userid:   userid.(int64),
	})
	if err != nil {
		zlog.Error(err.Error())
		utils.ResponseWithCode(ctx, utils.CodeUnknownError, nil, nil)
		return
	}
	utils.SuccessWithMsg(ctx, nil, nil)
}
