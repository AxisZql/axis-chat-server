package handler

import (
	"axisChat/api/rpc"
	"axisChat/api/utils"
	"axisChat/proto"
	"axisChat/utils/zlog"
	"github.com/gin-gonic/gin"
	"github.com/gin-gonic/gin/binding"
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
	reply, err := client.SearchGroup(ins.Ctx, &proto.SearchGroupRequest{
		GroupName: form.GroupName,
	})
	if err != nil {
		zlog.Error(err.Error())
		utils.ResponseWithCode(ctx, utils.CodeUnknownError, nil, nil)
		return
	}
	utils.SuccessWithMsg(ctx, nil, reply.GroupList)
}

type getGroupMsgByPageReq struct {
	GroupId  int64 `json:"groupId" binding:"required"`
	Userid   int64 `json:"userid" binding:"required"`
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
	reply, err := client.GetGroupMsgByPage(ins.Ctx, &proto.GetGroupMsgByPageRequest{
		GroupId:  form.GroupId,
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

type createGroupReq struct {
	Userid    int64  `json:"userid" binding:"required"`
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
	ins, err := rpc.GetLogicRpcInstance()
	if err != nil {
		utils.ResponseWithCode(ctx, utils.CodeUnknownError, nil, nil)
		return
	}
	client := proto.NewLogicClient(ins.Conn)
	reply, err := client.CreateGroup(ins.Ctx, &proto.Group{
		Userid:    form.Userid,
		GroupName: form.GroupName,
		Notice:    form.Notice,
	})
	if err != nil {
		zlog.Error(err.Error())
		utils.ResponseWithCode(ctx, utils.CodeUnknownError, nil, nil)
		return
	}
	utils.SuccessWithMsg(ctx, nil, reply)
}

type addGroup struct {
	Userid  int64 `json:"userid" binding:"required"`
	GroupId int64 `json:"groupId" binding:"required"`
}

func AddGroup(ctx *gin.Context) {
	var form addGroup
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
	_, err = client.AddGroup(ins.Ctx, &proto.AddGroupRequest{
		GroupId: form.GroupId,
		Userid:  form.Userid,
	})
	if err != nil {
		zlog.Error(err.Error())
		utils.ResponseWithCode(ctx, utils.CodeUnknownError, nil, nil)
		return
	}
	utils.SuccessWithMsg(ctx, nil, nil)
}
