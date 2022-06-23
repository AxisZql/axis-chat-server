package handler

import (
	"axisChat/api/rpc"
	"axisChat/api/utils"
	"axisChat/proto"
	"axisChat/utils/zlog"
	"context"
	"github.com/gin-gonic/gin"
	"github.com/gin-gonic/gin/binding"
	"time"
)

type pushReq struct {
	FriendId     int64  `json:"friendId" binding:"required"`
	Content      string `json:"content" binding:"required"`
	MessageType  string `json:"messageType" binding:"required"`
	FromUsername string `json:"fromUsername" binding:"required"`
	FriendName   string `json:"friendName" binding:"required"`
}

func Push(ctx *gin.Context) {
	var form pushReq
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
	_, err = client.Push(_ctx, &proto.PushRequest{
		Msg: &proto.ChatMessage{
			Userid:       userid.(int64),
			FriendId:     form.FriendId,
			Content:      form.Content,
			MessageType:  form.MessageType,
			FromUsername: form.FromUsername,
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
	GroupId      int64  `json:"groupId" binding:"required"`
	Content      string `json:"content" binding:"required"`
	MessageType  string `json:"messageType" binding:"required"`
	FromUsername string `json:"fromUsername" binding:"required"`
	GroupName    string `json:"groupName" binding:"required"`
}

func PushRoom(ctx *gin.Context) {
	var form pushRoomReq
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
	_, err = client.PushRoom(_ctx, &proto.PushRoomRequest{
		Msg: &proto.ChatMessage{
			Userid:       userid.(int64),
			GroupId:      form.GroupId,
			Content:      form.Content,
			MessageType:  form.MessageType,
			FromUsername: form.FromUsername,
			GroupName:    form.GroupName,
			CreateAt:     time.Now().Format(time.RFC3339),
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
	_ctx, cancel := context.WithTimeout(context.Background(), time.Second*5)
	defer cancel()
	_, err = client.PushRoomCount(_ctx, &proto.PushRoomCountRequest{
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
	_ctx, cancel := context.WithTimeout(context.Background(), time.Second*5)
	defer cancel()
	_, err = client.PushRoomInfo(_ctx, &proto.PushRoomInfoRequest{
		GroupId: form.GroupId,
	})
	if err != nil {
		zlog.Error(err.Error())
		utils.ResponseWithCode(ctx, utils.CodeUnknownError, nil, nil)
		return
	}
	utils.SuccessWithMsg(ctx, nil, nil)
}
