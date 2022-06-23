package connect

import (
	"axisChat/config"
	"axisChat/etcd"
	"axisChat/proto"
	"axisChat/utils/zlog"
	"context"
	"fmt"
	"github.com/google/uuid"
	"github.com/pkg/errors"
	"google.golang.org/grpc"
	"net"
	"runtime"
	"strings"
	"time"
)

type Connect struct {
	ServerId string
}

var (
	DefaultServer *WsServer
)

func New() *Connect {
	return &Connect{}
}

func (c *Connect) Run() {
	conf := config.GetConfig()
	runtime.GOMAXPROCS(conf.ConnectRpc.ConnectBucket.CpuNum)
	c.InitLogicClient()

	// TODO: init the bucket
	buckets := make([]*Bucket, conf.ConnectRpc.ConnectBucket.CpuNum)
	for i := 0; i < conf.ConnectRpc.ConnectBucket.CpuNum; i++ {
		buckets[i] = NewBucket()
	}
	operator := new(DefaultOperator)
	// 初始connect层的服务实例
	DefaultServer = NewServer(buckets, operator)
	// 生成当前服务实例的uuid
	c.ServerId = fmt.Sprintf("%s-%s", "ws", uuid.New().String())
	// 启动connect layer rpc服务
	list := strings.Split(conf.ConnectRpc.ConnectWebsocket.RpcAddress, ";")
	for _, val := range list {
		err := initConnectRpcServer(val, c.ServerId)
		if err != nil {
			panic(err)
		}
	}
	// 启动WebSocket服务
	go c.StartWebSocket(DefaultServer)
}

func initConnectRpcServer(address string, serverId string) (err error) {
	strList := strings.Split(address, "@")
	if len(strList) != 2 {
		err = errors.New("the address is not suitable the rule network@port")
		return
	}
	network, port := strList[0], strList[1]
	lis, err := net.Listen(network, fmt.Sprintf(":%s", port))
	if err != nil {
		return
	}
	s := grpc.NewServer()
	proto.RegisterConnectLayerServer(s, &ServerConnect{})
	go func() {
		if err = s.Serve(lis); err != nil {
			panic(fmt.Sprintf("启动connect layer的%s服务失败,err:%v", address, err))
		}
	}()
	conf := config.GetConfig().ConnectRpc.ConnectWebsocket
	addr := fmt.Sprintf("%s:%s", conf.Host, port)
	registerServer2Etcd(addr, serverId)
	return
}

func registerServer2Etcd(address string, serverId string) {
	conf := config.GetConfig()
	endpoint := strings.Split(conf.Common.Etcd.Address, ";")
	if len(endpoint) == 0 {
		panic("at least one ETCD service is required !")
	}
	// TODO:logic层服务注册到etcd服务上到路径
	serverPathInEtcd := fmt.Sprintf("%s/%s&serverId=%s&address=%s", conf.Common.Etcd.BasePath, conf.Common.Etcd.ServerPathConnect, serverId, address)
	ser, err := etcd.NewServiceRegister(endpoint, serverPathInEtcd, address, 6, 5)
	if err != nil {
		panic(err)
	}
	// 监听并续租对应到chan
	go ser.ListenLeaseRespChan()
}

func (c *Connect) Connect(accessToken, serverId string) (userid int64, err error) {
	// 首先获取实例
	logicRpcInstance.ins, err = serDiscovery.GetServiceByServerId(logicRpcInstance.serverId)
	if err != nil {
		zlog.Error(err.Error())
		return
	}
	logicClient := proto.NewLogicClient(logicRpcInstance.ins.Conn)
	_ctx, cancel := context.WithTimeout(context.Background(), time.Second*5)
	defer cancel()
	reply, err := logicClient.Connect(_ctx, &proto.ConnectRequest{
		AccessToken: accessToken,
		ServerId:    serverId,
	})
	if err != nil {
		zlog.Error(fmt.Sprintf("调用logic层Connect方法错误：err=%v", err))
		return
	}
	userid = reply.Userid
	return
}

func (c *Connect) DisConnect(userid int64) (err error) {
	// 首先获取实例
	logicRpcInstance.ins, err = serDiscovery.GetServiceByServerId(logicRpcInstance.serverId)
	if err != nil {
		zlog.Error(err.Error())
		return
	}
	logicClient := proto.NewLogicClient(logicRpcInstance.ins.Conn)
	_ctx, cancel := context.WithTimeout(context.Background(), time.Second*5)
	defer cancel()
	_, err = logicClient.DisConnect(_ctx, &proto.DisConnectRequest{
		Userid: userid,
	})
	if err != nil {
		zlog.Error(fmt.Sprintf("调用logic层DisConnect方法错误：err=%v", err))
		return
	}
	return
}
