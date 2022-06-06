package connect

import (
	"axisChat/utils"
	"fmt"
	"time"
)

type WsServer struct {
	Buckets   []*Bucket
	bucketNum uint32
	Options   wsServerOptions
	Operator  Operator //连接操作和断开操作interface
}

type wsServerOptions struct {
	WriteWait       time.Duration // 写操作超时时间
	PongWait        time.Duration // 心跳响应包接收等待时间
	PingPeriod      time.Duration // 心跳包检测周期
	MaxMessageSize  int           // 最大消息限制
	ReadBufferSize  int           // 读消息时的缓冲大小
	WriteBufferSize int           // 写消息时的缓冲大小
	BroadcastSize   int           // 一个channel最多可以缓冲多少个消息推送请求
}

const (
	defaultWriteWait       = 10 * time.Second
	defaultPongWait        = 60 * time.Second
	defaultPingPeriod      = 54 * time.Second
	defaultMaxMessageSize  = 512
	defaultReadBufferSize  = 1024
	defaultWriteBufferSize = 1024
	defaultBroadcastSize   = 512
)

func NewServer(buckets []*Bucket, op Operator, opts ...WsServerOption) (ws *WsServer) {
	options := wsServerOptions{
		WriteWait:       defaultWriteWait,
		PongWait:        defaultPongWait,
		PingPeriod:      defaultPingPeriod,
		MaxMessageSize:  defaultMaxMessageSize,
		ReadBufferSize:  defaultReadBufferSize,
		WriteBufferSize: defaultWriteBufferSize,
		BroadcastSize:   defaultBroadcastSize,
	}
	for _, o := range opts {
		o.apply(&options)
	}
	ws.Buckets = buckets
	ws.bucketNum = uint32(len(buckets))
	ws.Options = options
	ws.Operator = op
	return
}

// Bucket 通过cityHash把不同Channel均匀分散到同一个ServerId服务的不同Bucket中
//因此在一个serverId服务中同一个群聊的节点（groupNode）可能在同时存在不同的bucket中
func (ws *WsServer) Bucket(userid int64) *Bucket {
	useridStr := fmt.Sprintf("%d", userid)
	idx := utils.CityHash32([]byte(useridStr), uint32(len(useridStr))) % ws.bucketNum
	return ws.Buckets[idx]
}

type WsServerOption interface {
	apply(options *wsServerOptions)
}
type wsServerOptionFunc func(options *wsServerOptions)

func (f wsServerOptionFunc) apply(options *wsServerOptions) {
	f(options)
}

func WsWithMaxMessageSize(size int) WsServerOption {
	return wsServerOptionFunc(func(options *wsServerOptions) {
		options.MaxMessageSize = size
	})
}
func WsWithReadBufferSize(size int) WsServerOption {
	return wsServerOptionFunc(func(options *wsServerOptions) {
		options.ReadBufferSize = size
	})
}

func WsWithWriteBufferSize(size int) WsServerOption {
	return wsServerOptionFunc(func(options *wsServerOptions) {
		options.WriteBufferSize = size
	})
}
