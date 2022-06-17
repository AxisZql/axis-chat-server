package utils

import (
	"github.com/gin-gonic/gin"
	"net/http"
)

const (
	CodeSuccess      = 0
	CodeFail         = 1
	CodeUnknownError = -1
	CodeSessionError = 40100
)

var MsgCodeMap = map[int]string{
	CodeSuccess:      "success",
	CodeFail:         "fail",
	CodeUnknownError: "unknown error",
	CodeSessionError: "invalid session",
}

func SuccessWithMsg(ctx *gin.Context, msg interface{}, data interface{}) {
	ResponseWithCode(ctx, CodeSuccess, msg, data)
}

func FailWithMsg(ctx *gin.Context, msg interface{}) {
	ResponseWithCode(ctx, CodeFail, msg, nil)
}

func ResponseWithCode(ctx *gin.Context, code int, msg interface{}, data interface{}) {
	if msg == nil {
		if val, ok := MsgCodeMap[code]; ok {
			msg = val
		} else {
			msg = MsgCodeMap[-1]
		}
	}
	ctx.AbortWithStatusJSON(http.StatusOK, gin.H{
		"code": code,
		"msg":  msg,
		"data": data,
	})
}
