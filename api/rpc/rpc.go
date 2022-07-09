package rpc

import (
	"axisChat/config"
	"axisChat/etcd"
	"axisChat/utils/zlog"
	"fmt"
	"strings"
)

type LogicRpcInstance struct {
	serverId string
}

var (
	serDiscovery     *etcd.ServiceDiscovery
	logicRpcInstance = &LogicRpcInstance{}
)

// InitLogicClient 初始化获取logic层的rpc服务客户端
func InitLogicClient() {
	conf := config.GetConfig()
	etcdAddrList := strings.Split(conf.Common.Etcd.Address, ";")
	serDiscovery = etcd.NewServiceDiscovery(etcdAddrList)
	//TODO：从etcd中 获取所有logic layer 的服务地址列表，并监听其改变
	err := serDiscovery.WatchService(fmt.Sprintf("%s/%s", conf.Common.Etcd.BasePath, conf.Common.Etcd.ServerPathLogic))
	if err != nil {
		panic(err)
	}

	// 轮询获取对应serverId下的logic layer rpc服务实例,这获取实例是为了确保在程序开始运行之前有可用的logic layer rpc实例
	_, err = serDiscovery.GetServiceByServerId(conf.LogicRpc.Logic.ServerId)
	if err != nil {
		zlog.Error(err.Error())
	}
	logicRpcInstance.serverId = conf.LogicRpc.Logic.ServerId
}

func GetLogicRpcInstance() (ins *etcd.Instance, err error) {
	ins, err = serDiscovery.GetServiceByServerId(logicRpcInstance.serverId)
	if err != nil {
		zlog.Error(err.Error())
	}
	return
}
