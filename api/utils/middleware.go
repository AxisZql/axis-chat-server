package utils

import (
	"axisChat/api/rpc"
	"axisChat/proto"
	"axisChat/utils/zlog"
	"context"
	"github.com/gin-gonic/gin"
	"github.com/gin-gonic/gin/binding"
	"net/http"
	"time"
)

/*
*Author: AxisZql
*Date: 2022-6-17 1:38 PM
*Desc: the middleware define
 */

// Cors 跨域中间件
func Cors() gin.HandlerFunc {
	return func(ctx *gin.Context) {
		method := ctx.Request.Method
		ctx.Header("Access-Control-Allow-Origin", "*")
		ctx.Header("Access-Control-Allow-Headers", "Content-Type,AccessToken,X-CSRF-Token,Authorization,Token")
		ctx.Header("Access-Control-Expose-Headers", "Content-Length,Access-Control-Allow-Origin,Access-Control-Allow-Headers,Content-Type")
		ctx.Header("Access-Control-Allow-Credentials", "true")

		if method == "OPTIONS" {
			ctx.AbortWithStatus(http.StatusNoContent)
		}
	}
}

type accessTokenForm struct {
	AccessToken string `json:"accessToken" form:"accessToken" binding:"required"`
}

// CheckSession 会话合法性验证中间件
func CheckSession() gin.HandlerFunc {
	return func(ctx *gin.Context) {
		var form accessTokenForm
		// 同时支持json和form两种身份验证方式
		if err := ctx.ShouldBindBodyWith(&form, binding.JSON); err != nil {
			if err = ctx.ShouldBind(&form); err != nil {
				ctx.Abort()
				ResponseWithCode(ctx, CodeSessionError, nil, nil)
				return
			}
		}
		ins, err := rpc.GetLogicRpcInstance()
		if err != nil {
			ctx.Abort()
			ResponseWithCode(ctx, CodeUnknownError, nil, nil)
			return
		}
		client := proto.NewLogicClient(ins.Conn)
		_ctx, cancel := context.WithTimeout(context.Background(), time.Second*5)
		defer cancel()
		reply, err := client.GetUserInfoByAccessToken(_ctx, &proto.GetUserInfoByAccessTokenRequest{
			AccessToken: form.AccessToken,
		})
		if err != nil {
			ctx.Abort()
			zlog.Error(err.Error())
			ResponseWithCode(ctx, CodeSessionError, nil, nil)
			return
		}
		ctx.Set("userid", reply.User.Id)
		ctx.Set("role", reply.User.Role)
		ctx.Next()
		return
	}
}
