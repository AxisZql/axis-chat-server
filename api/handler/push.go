package handler

import (
	"axisChat/api/rpc"
	"axisChat/api/utils"
	"axisChat/proto"
	"axisChat/utils/zlog"
	"github.com/gin-gonic/gin"
	"github.com/gin-gonic/gin/binding"
	"time"
)

type pushReq struct {
	Userid       int64  `json:"userid" binding:"required"`
	FriendId     int64  `json:"friendId" binding:"required"`
	Content      string `json:"content" binding:"required"`
	MessageType  string `json:"messageType" binding:"required"`
	FromUserName string `json:"fromUserName" binding:"required"`
	FriendName   string `json:"friendName" binding:"required"`
}

func Push(ctx *gin.Context) {
	var form pushReq
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
	_, err = client.Push(ins.Ctx, &proto.PushRequest{
		Msg: &proto.ChatMessage{
			Userid:       form.Userid,
			FriendId:     form.FriendId,
			Content:      form.Content,
			MessageType:  form.MessageType,
			FromUsername: form.FromUserName,
			FriendName:   form.FriendName,
		},
	})
	if err != nil {
		zlog.Error(err.Error())
		utils.ResponseWithCode(ctx, utils.CodeUnknownError, nil, nil)
		return
	}
	utils.SuccessWithMsg(ctx, nil, nil)
}

type pushRoomReq struct {
	Userid       int64  `json:"userid" binding:"required"`
	GroupId      int64  `json:"groupId" binding:"required"`
	Content      string `json:"content" binding:"required"`
	MessageType  string `json:"messageType" binding:"required"`
	FromUserName string `json:"fromUserName" binding:"required"`
	GroupName    string `json:"groupName" binding:"required"`
}

func PushRoom(ctx *gin.Context) {
	var form pushRoomReq
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
	_, err = client.PushRoom(ins.Ctx, &proto.PushRoomRequest{
		Msg: &proto.ChatMessage{
			Userid:       form.Userid,
			GroupId:      form.GroupId,
			Content:      form.Content,
			MessageType:  form.MessageType,
			FromUsername: form.FromUserName,
			GroupName:    form.GroupName,
			CreateAt:     time.Now().Unix(),
		},
	})
	if err != nil {
		zlog.Error(err.Error())
		utils.ResponseWithCode(ctx, utils.CodeUnknownError, nil, nil)
		return
	}
	utils.SuccessWithMsg(ctx, nil, nil)
}

type pushRoomCountReq struct {
	GroupId int64 `json:"groupId" binding:"required"`
}

func PushRoomCount(ctx *gin.Context) {
	var form pushRoomCountReq
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
	_, err = client.PushRoomCount(ins.Ctx, &proto.PushRoomCountRequest{
		GroupId: form.GroupId,
	})
	if err != nil {
		zlog.Error(err.Error())
		utils.ResponseWithCode(ctx, utils.CodeUnknownError, nil, nil)
		return
	}
	utils.SuccessWithMsg(ctx, nil, nil)
}

type pushRoomInfoReq struct {
	GroupId int64 `json:"groupId" binding:"required"`
}

func PushRoomInfo(ctx *gin.Context) {
	var form pushRoomInfoReq
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
	_, err = client.PushRoomInfo(ins.Ctx, &proto.PushRoomInfoRequest{
		GroupId: form.GroupId,
	})
	if err != nil {
		zlog.Error(err.Error())
		utils.ResponseWithCode(ctx, utils.CodeUnknownError, nil, nil)
		return
	}
	utils.SuccessWithMsg(ctx, nil, nil)
}
