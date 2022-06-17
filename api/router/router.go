package router

import (
	"axisChat/api/handler"
	"axisChat/api/utils"
	"github.com/gin-gonic/gin"
)

func Register() *gin.Engine {
	r := gin.Default()
	r.Use(utils.Cors())

	// 注册路由
	initUserRouter(r)
	initGroupRouter(r)
	initPushRouter(r)
	r.NoRoute(func(ctx *gin.Context) {
		utils.FailWithMsg(ctx, "not support the url")
	})
	return r
}

func initUserRouter(r *gin.Engine) {
	userRouter := r.Group("/user")
	userRouter.POST("/register", handler.Register)
	userRouter.POST("/login", handler.Login)
	userRouter.POST("/check-auth", handler.GetUserInfoByAccessToken)
	userRouter.Use(utils.CheckSession())
	{
		userRouter.GET("/login-out", handler.LoginOut)
		userRouter.GET("/info", handler.GetUserInfoByUserid)
		userRouter.POST("/update-info", handler.UpdateUserInfo)
		userRouter.POST("/update-password", handler.UpdatePassword)
		userRouter.GET("/search", handler.SearchUser)
		userRouter.GET("/chat-history", handler.GetFriendMsgByPage)
		userRouter.POST("/add-friend", handler.AddFriend)
	}

}

func initGroupRouter(r *gin.Engine) {
	groupRouter := r.Group("/group")
	groupRouter.Use(utils.CheckSession())
	{
		groupRouter.GET("/search", handler.SearchGroup)
		groupRouter.GET("/chat-history", handler.GetGroupMsgByPage)
		groupRouter.POST("/create", handler.CreateGroup)
		groupRouter.POST("/add-group", handler.AddGroup)
	}
}

func initPushRouter(r *gin.Engine) {
	pushRouter := r.Group("/push")
	pushRouter.Use(utils.CheckSession())
	{
		pushRouter.POST("/push-friend", handler.Push)
		pushRouter.POST("/push-group", handler.PushRoom)
		pushRouter.POST("/push-group-count", handler.PushRoomCount)
		pushRouter.POST("/push-group-info", handler.PushRoomInfo)
	}
}
