package api

import (
	"axisChat/api/router"
	"axisChat/api/rpc"
	"axisChat/config"
	"axisChat/utils/zlog"
	"fmt"
	"github.com/gin-gonic/gin"
	"net/http"
)

type Chat struct{}

func New() *Chat {
	return &Chat{}
}

func (chat *Chat) Run() {
	// init logic layer rpc server client
	rpc.InitLogicClient()

	r := router.Register()
	conf := config.GetConfig()
	// 设置gin的运行模式「生产环境不要使用DEBUG模式」
	runMode := config.GetGinRunMode()
	gin.SetMode(runMode)

	server := &http.Server{
		Addr:    fmt.Sprintf(":%d", conf.Api.Api.Port),
		Handler: r,
	}

	go func() {
		err := server.ListenAndServe()
		if err != nil {
			zlog.Error(fmt.Sprintf("start up the API layer server at :%d is failure!", conf.Api.Api.Port))
		}
	}()
}
