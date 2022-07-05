package handler

import (
	"axisChat/api/rpc"
	"axisChat/api/utils"
	"axisChat/config"
	"axisChat/proto"
	"axisChat/utils/zlog"
	"context"
	"crypto/md5"
	"fmt"
	"github.com/gin-gonic/gin"
	"github.com/gin-gonic/gin/binding"
	"io/ioutil"
	"mime/multipart"
	"os"
	"strings"
	"time"
)

type pushReq struct {
	FriendId     int64  `json:"friendId" binding:"required"`
	Content      string `json:"content" binding:"required"`
	MessageType  string `json:"messageType" binding:"required"`
	FromUsername string `json:"fromUsername" binding:"required"`
	FriendName   string `json:"friendName" binding:"required"`
	Avatar       string `json:"avatar" binding:"required"`
	Watermark    int64  `json:"watermark" binding:"required"`
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
			Avatar:       form.Avatar,
			CreateAt:     time.Now().Format(time.RFC3339),
			Watermark:    form.Watermark,
		},
	})
	if err != nil {
		zlog.Error(err.Error())
		utils.ResponseWithCode(ctx, utils.CodeUnknownError, nil, nil)
		return
	}
	utils.SuccessWithMsg(ctx, nil, "success")
}

type pushRoomReq struct {
	GroupId      int64  `json:"groupId" binding:"required"`
	Content      string `json:"content" binding:"required"`
	MessageType  string `json:"messageType" binding:"required"`
	FromUsername string `json:"fromUsername" binding:"required"`
	GroupName    string `json:"groupName" binding:"required"`
	Avatar       string `json:"avatar" binding:"required"`
	Watermark    int64  `json:"watermark" binding:"required"`
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
			Avatar:       form.Avatar,
			Watermark:    form.Watermark,
		},
	})
	if err != nil {
		zlog.Error(err.Error())
		utils.ResponseWithCode(ctx, utils.CodeUnknownError, nil, nil)
		return
	}
	utils.SuccessWithMsg(ctx, nil, "success")
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

type reqPushImgMsg struct {
	// 公共部分
	Content      *multipart.FileHeader `form:"content" binding:"required"`
	Ty           string                `form:"type" binding:"required"`   //图片对象是group还是friend
	FromId       int64                 `form:"fromId" binding:"required"` //如果是群聊则是groupId，否则是图片发起方的id
	FromUsername string                `form:"fromUsername" binding:"required"`
	Avatar       string                `form:"avatar" binding:"required"`
	AccessToken  string                `form:"accessToken" binding:"required"`
	Watermark    int64                 `form:"watermark" binding:"required"`

	// 群聊消息独有
	GroupId   int64  `form:"groupId"`
	GroupName string `form:"groupName"`

	// 私聊消息独有
	FriendId   int64  `form:"friendId"`
	FriendName string `form:"friendName"`
}

func PushImgMsg(ctx *gin.Context) {
	var form reqPushImgMsg
	if err := ctx.ShouldBind(&form); err != nil {
		zlog.Error(err.Error())
		utils.FailWithMsg(ctx, "参数校验失败")
		return
	}
	// 先验证发送方身份合法性
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
		utils.ResponseWithCode(ctx, utils.CodeSessionError, nil, nil)
		return
	}

	// 存放图片
	f, _ := form.Content.Open()
	extendName := strings.Split(form.Content.Filename, ".")
	if len(extendName) != 2 && extendName[1] != "png" && extendName[1] != "gif" && extendName[1] != "jpg" {
		utils.FailWithMsg(ctx, "不支持的图片格式;仅支持png|gif|jpg格式")
		return
	}
	defer f.Close()
	fileData, err2 := ioutil.ReadAll(f)
	if err2 != nil {
		zlog.Error(err2.Error())
		utils.FailWithMsg(ctx, "系统异常")
		return
	}
	conf := config.GetConfig()
	filePath := conf.Api.Api.ChatImgDir + form.Ty + "/" + fmt.Sprintf("%d/", form.FromId)
	err = os.MkdirAll(filePath, os.ModePerm)
	if err != nil {
		zlog.Error(fmt.Sprintf("创建聊天图片存放目录失败:%v", err))
		utils.FailWithMsg(ctx, "系统异常")
		return
	}

	fileMD5 := fmt.Sprintf("%x", md5.Sum(fileData))
	fileName := fileMD5 + "." + extendName[1]

	filePath = filePath + fileName
	err = ctx.SaveUploadedFile(form.Content, filePath)
	if err != nil {
		zlog.Error(err.Error())
		utils.FailWithMsg(ctx, "系统异常")
		return
	}
	//example: https://localhost:8090/images/group/1/8dwekdkjfl.png
	imgUrl := fmt.Sprintf("%s/images/%s/%d/%s", conf.Api.Api.Host, form.Ty, form.FromId, fileName)

	// 将图片消息写入Kafka对应的topic
	switch form.Ty {
	case "group":
		defer cancel()
		_, err = client.PushRoom(_ctx, &proto.PushRoomRequest{
			Msg: &proto.ChatMessage{
				Userid:       reply.User.Id,
				GroupId:      form.GroupId,
				Content:      imgUrl,
				MessageType:  "image",
				FromUsername: form.FromUsername,
				GroupName:    form.GroupName,
				CreateAt:     time.Now().Format(time.RFC3339),
				Avatar:       form.Avatar,
				Watermark:    form.Watermark,
			},
		})
		if err != nil {
			zlog.Error(err.Error())
			utils.ResponseWithCode(ctx, utils.CodeUnknownError, nil, nil)
			return
		}
	case "friend":
		_, err = client.Push(_ctx, &proto.PushRequest{
			Msg: &proto.ChatMessage{
				Userid:       reply.User.Id,
				FriendId:     form.FriendId,
				Content:      imgUrl,
				MessageType:  "image",
				FromUsername: form.FromUsername,
				FriendName:   form.FriendName,
				Avatar:       form.Avatar,
				CreateAt:     time.Now().Format(time.RFC3339),
				Watermark:    form.Watermark,
			},
		})
		if err != nil {
			zlog.Error(err.Error())
			utils.ResponseWithCode(ctx, utils.CodeUnknownError, nil, nil)
			return
		}
	}
	utils.SuccessWithMsg(ctx, nil, imgUrl)
}
