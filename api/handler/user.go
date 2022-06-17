package handler

import (
	"axisChat/api/rpc"
	"axisChat/api/utils"
	"axisChat/db"
	"axisChat/proto"
	"axisChat/utils/zlog"
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
	reply, err := client.Register(ins.Ctx, &proto.RegisterRequest{
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
	reply, err := client.Login(ins.Ctx, &proto.LoginRequest{
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
	AccessToken string `json:"access_token" binding:"required"`
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
	_, err = client.LoginOut(ins.Ctx, &proto.LoginOutRequest{
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
	AccessToken string `json:"access_token" binding:"required"`
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
	reply, err := client.GetUserInfoByAccessToken(ins.Ctx, &proto.GetUserInfoByAccessTokenRequest{
		AccessToken: form.AccessToken,
	})
	if err != nil {
		zlog.Error(err.Error())
		utils.ResponseWithCode(ctx, utils.CodeUnknownError, nil, nil)
		return
	}
	data := db.User{
		ID:       reply.User.Userid,
		Username: reply.User.Username,
		Avatar:   reply.User.Avatar,
		Role:     int(reply.User.Role),
		Status:   int(reply.User.Status),
		Tag:      reply.User.Tag,
		CreateAt: time.UnixMilli(reply.User.CreateAt).Local(),
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
	reply, err := client.GetUserInfoByUserid(ins.Ctx, &proto.GetUserInfoByUseridRequest{
		Userid: form.Userid,
	})
	if err != nil {
		zlog.Error(err.Error())
		utils.ResponseWithCode(ctx, utils.CodeUnknownError, nil, nil)
		return
	}
	data := db.User{
		ID:       reply.User.Userid,
		Username: reply.User.Username,
		Avatar:   reply.User.Avatar,
		Role:     int(reply.User.Role),
		Status:   int(reply.User.Status),
		Tag:      reply.User.Tag,
		CreateAt: time.UnixMilli(reply.User.CreateAt).Local(),
	}
	utils.SuccessWithMsg(ctx, nil, data)
}

type updateUserInfo struct {
	Userid   int64  `json:"userid" binding:"required"`
	Username string `json:"username" binding:"required"`
	Avatar   string `json:"avatar" binding:"required"`
	Role     int    `json:"role" binding:"required"`
	Status   int    `json:"status" binding:"required"`
	Tag      string `json:"tag" binding:"required"`
}

func UpdateUserInfo(ctx *gin.Context) {
	var form updateUserInfo
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
	reply, err := client.UpdateUserInfo(ins.Ctx, &proto.UpdateUserInfoRequest{
		User: &proto.User{
			Userid:   form.Userid,
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
	data := db.User{
		ID:       reply.User.Userid,
		Username: reply.User.Username,
		Avatar:   reply.User.Avatar,
		Role:     int(reply.User.Role),
		Status:   int(reply.User.Status),
		Tag:      reply.User.Tag,
		CreateAt: time.UnixMilli(reply.User.CreateAt).Local(),
	}
	utils.SuccessWithMsg(ctx, nil, data)
}

type updatePasswordReq struct {
	Userid       int64  `json:"userid" binding:"required"`
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
	ins, err := rpc.GetLogicRpcInstance()
	if err != nil {
		utils.ResponseWithCode(ctx, utils.CodeUnknownError, nil, nil)
		return
	}
	client := proto.NewLogicClient(ins.Conn)
	_, err = client.UpdatePassword(ins.Ctx, &proto.UpdatePasswordRequest{
		Userid:       form.Userid,
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
	reply, err := client.SearchUser(ins.Ctx, &proto.SearchUserRequest{
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
	Userid   int64 `json:"userid" binding:"required"`
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
	ins, err := rpc.GetLogicRpcInstance()
	if err != nil {
		utils.ResponseWithCode(ctx, utils.CodeUnknownError, nil, nil)
		return
	}
	client := proto.NewLogicClient(ins.Conn)
	reply, err := client.GetFriendMsgByPage(ins.Ctx, &proto.GetFriendMsgByPageRequest{
		FriendId: form.FriendId,
		Userid:   form.Userid,
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
	Userid   int64 `json:"userid" binding:"required"`
	FriendId int64 `json:"friendId" binding:"required"`
}

func AddFriend(ctx *gin.Context) {
	var form addFriend
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
	_, err = client.AddFriend(ins.Ctx, &proto.AddFriendRequest{
		FriendId: form.FriendId,
		Userid:   form.Userid,
	})
	if err != nil {
		zlog.Error(err.Error())
		utils.ResponseWithCode(ctx, utils.CodeUnknownError, nil, nil)
		return
	}
	utils.SuccessWithMsg(ctx, nil, nil)
}
