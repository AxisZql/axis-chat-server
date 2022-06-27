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

type searchGroupReq struct {
	GroupName string `json:"groupName" binding:"required"`
}

func SearchGroup(ctx *gin.Context) {
	var form searchGroupReq
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
	reply, err := client.SearchGroup(_ctx, &proto.SearchGroupRequest{
		GroupName: form.GroupName,
	})
	if err != nil {
		zlog.Error(err.Error())
		utils.ResponseWithCode(ctx, utils.CodeUnknownError, nil, nil)
		return
	}
	utils.SuccessWithMsg(ctx, nil, reply)
}

type getGroupMsgByPageReq struct {
	GroupId  int64 `json:"groupId" binding:"required"`
	Current  int64 `json:"current" binding:"required"`
	PageSize int64 `json:"pageSize" binding:"required"`
}

func GetGroupMsgByPage(ctx *gin.Context) {
	var form getGroupMsgByPageReq
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
	reply, err := client.GetGroupMsgByPage(_ctx, &proto.GetGroupMsgByPageRequest{
		GroupId:  form.GroupId,
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

type createGroupReq struct {
	GroupName string `json:"groupName" binding:"required"`
	Notice    string `json:"notice" binding:"required"`
}

func CreateGroup(ctx *gin.Context) {
	var form createGroupReq
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
	reply, err := client.CreateGroup(_ctx, &proto.Group{
		Userid:    userid.(int64),
		GroupName: form.GroupName,
		Notice:    form.Notice,
	})
	if err != nil {
		zlog.Error(err.Error())
		utils.ResponseWithCode(ctx, utils.CodeUnknownError, nil, nil)
		return
	}
	// 添加完毕群聊后接着加入创建的群聊
	_, err = client.AddGroup(_ctx, &proto.AddGroupRequest{
		GroupId: reply.Id,
		Userid:  userid.(int64),
	})
	if err != nil {
		zlog.Error(err.Error())
		utils.ResponseWithCode(ctx, utils.CodeUnknownError, nil, nil)
		return
	}
	utils.SuccessWithMsg(ctx, nil, reply)
}

type addGroup struct {
	GroupId int64 `json:"groupId" binding:"required"`
}

func AddGroup(ctx *gin.Context) {
	var form addGroup
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
	_, err = client.AddGroup(_ctx, &proto.AddGroupRequest{
		GroupId: form.GroupId,
		Userid:  userid.(int64),
	})
	if err != nil {
		zlog.Error(err.Error())
		utils.ResponseWithCode(ctx, utils.CodeUnknownError, nil, nil)
		return
	}
	utils.SuccessWithMsg(ctx, nil, nil)
}
