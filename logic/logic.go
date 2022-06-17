package logic

import (
	"axisChat/config"
	"axisChat/etcd"
	"axisChat/proto"
	"fmt"
	"github.com/pkg/errors"
	"google.golang.org/grpc"
	"net"
	"strings"
)

/**
*Author:AxisZql
*Date:2022-5-31
*DESC:logic layer server start up
 */

type Logic struct {
	ServerId string
}

func New() *Logic {
	return new(Logic)
}

func (logic *Logic) Run() {
	//logic.ServerId = fmt.Sprintf("logic-%s", uuid.New().String())
	conf := config.GetConfig().LogicRpc.Logic
	logic.ServerId = conf.ServerId
	list := strings.Split(conf.RpcAddress, ";")
	for _, val := range list {
		err := initLogicRpcServer(val, logic.ServerId)
		if err != nil {
			panic(err)
		}
	}
}

func initLogicRpcServer(address string, serverId string) (err error) {
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
	proto.RegisterLogicServer(s, &ServerLogic{})
	go func() {
		if err = s.Serve(lis); err != nil {
			panic(fmt.Sprintf("启动logic layer的%s服务失败,err:%v", address, err))
		}
	}()
	conf := config.GetConfig().LogicRpc.Logic
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
	serverPathInEtcd := fmt.Sprintf("%s/%s&serverId=%s&address=%s", conf.Common.Etcd.BasePath, conf.Common.Etcd.ServerPathLogic, serverId, address)
	ser, err := etcd.NewServiceRegister(endpoint, serverPathInEtcd, address, 6, 5)
	if err != nil {
		panic(err)
	}
	// 监听并续租对应到chan
	go ser.ListenLeaseRespChan()
}
