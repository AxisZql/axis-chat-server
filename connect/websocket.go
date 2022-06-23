package connect

import (
	"axisChat/config"
	"axisChat/utils/zlog"
	"github.com/gorilla/websocket"
	"net/http"
)

func (c *Connect) StartWebSocket(ws *WsServer) {
	conf := config.GetConfig().ConnectRpc.ConnectWebsocket
	http.HandleFunc("/ws", func(writer http.ResponseWriter, request *http.Request) {
		upGrader := websocket.Upgrader{
			ReadBufferSize:  DefaultServer.Options.ReadBufferSize,
			WriteBufferSize: DefaultServer.Options.WriteBufferSize,
		}
		// 支持跨域
		upGrader.CheckOrigin = func(r *http.Request) bool {
			return true
		}
		conn, err := upGrader.Upgrade(writer, request, nil)
		if err != nil {
			zlog.Error(err.Error())
			return
		}
		ch := NewChannel(DefaultServer.Options.BroadcastSize)
		ch.Conn = conn
		// 前端只需要发送携带凭证的数据包过来，验证通过后会调用logic layer的rpc服务，在redis中写入该用户的登陆状态，进而消费其信箱中的消息
		go ws.readPump(ch, c)
		// 后端通过websocket接口推送消息给对应客户端，推送成功后，该消息成功消费，在kafka中提交消费偏移量
		go ws.writePump(ch)
	})
	err := http.ListenAndServe(conf.Bind, nil)
	if err != nil {
		panic(err)
	}
}
