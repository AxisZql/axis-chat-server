package task

import (
	"axisChat/common"
	"axisChat/config"
	"axisChat/etcd"
	"axisChat/proto"
	"axisChat/utils"
	"axisChat/utils/zlog"
	"context"
	"encoding/json"
	"fmt"
	"github.com/segmentio/kafka-go"
	"strings"
	"time"
)

type ConnectRpcInstance struct {
	ins *etcd.Instance
}

var (
	serDiscovery       *etcd.ServiceDiscovery
	connectRpcInstance = &ConnectRpcInstance{}
)

// InitConnectRpcClient 初始化获取logic层的rpc服务客户端
func (task *Task) InitConnectRpcClient() {
	conf := config.GetConfig()
	etcdAddrList := strings.Split(conf.Common.Etcd.Address, ";")
	serDiscovery = etcd.NewServiceDiscovery(etcdAddrList)
	//TODO：从etcd中 获取所有logic layer 的服务地址列表，并监听其改变
	err := serDiscovery.WatchService(fmt.Sprintf("%s/%s", conf.Common.Etcd.BasePath, conf.Common.Etcd.ServerPathConnect))
	if err != nil {
		panic(err)
	}
}

//todo 因为一个群聊可以在多个serverId中,所以往connect层推送消息时，可以通过redis获取对应群聊所有在线成员所在serverId，然后给这些serverId推送该群聊消息

func (task *Task) pushGroupInfoMsg(serverId string, msg *kafka.Message) {
	var err error
	connectRpcInstance.ins, err = serDiscovery.GetServiceByServerId(serverId)
	if err != nil {
		zlog.Error(err.Error())
		return
	}
	var payload common.MsgSend
	payload.Msg = new(proto.PushGroupInfoMsgReq_Msg)
	_ = json.Unmarshal(msg.Value, &payload)
	connectClient := proto.NewConnectLayerClient(connectRpcInstance.ins.Conn)
	_ctx, cancel := context.WithTimeout(context.Background(), time.Second*5)
	defer cancel()

	headers := make([]*proto.KafkaMsgInfo_Header, 0)
	for _, val := range msg.Headers {
		headers = append(headers, &proto.KafkaMsgInfo_Header{
			Key:   val.Key,
			Value: msg.Value,
		})
	}
	_, err = connectClient.PushGroupInfoMsg(_ctx, &proto.PushGroupInfoMsgReq{
		Msg: payload.Msg.(*proto.PushGroupInfoMsgReq_Msg),
		KafkaInfo: &proto.KafkaMsgInfo{
			Topic:          msg.Topic,
			Partition:      int32(msg.Partition),
			Offset:         msg.Offset,
			High_WaterMark: msg.HighWaterMark,
			Key:            msg.Key,
			Value:          msg.Value,
			Headers:        headers,
			Time:           msg.Time.Format(time.RFC3339),
		},
	})
	if err != nil {
		zlog.Error(err.Error())
	}
}

func (task *Task) pushGroupCountMsg(serverId string, msg *kafka.Message) {
	var err error
	connectRpcInstance.ins, err = serDiscovery.GetServiceByServerId(serverId)
	if err != nil {
		zlog.Error(err.Error())
		return
	}
	var payload common.MsgSend
	payload.Msg = new(proto.PushGroupCountMsgReq_Msg)
	connectClient := proto.NewConnectLayerClient(connectRpcInstance.ins.Conn)
	_ctx, cancel := context.WithTimeout(context.Background(), time.Second*5)
	defer cancel()
	_, err = connectClient.PushGroupCountMsg(_ctx, &proto.PushGroupCountMsgReq{
		Msg: payload.Msg.(*proto.PushGroupCountMsgReq_Msg),
		KafkaInfo: &proto.KafkaMsgInfo{
			Topic:     msg.Topic,
			Partition: int32(msg.Partition),
			Offset:    msg.Offset,
		},
	})
	if err != nil {
		zlog.Error(err.Error())
	}
}

func (task *Task) pushFriendOnlineMsg(serverId string, msg *kafka.Message) {
	var err error
	connectRpcInstance.ins, err = serDiscovery.GetServiceByServerId(serverId)
	if err != nil {
		zlog.Error(err.Error())
		return
	}
	var payload common.MsgSend
	payload.Msg = new(proto.PushFriendOnlineMsgReq_Msg)
	connectClient := proto.NewConnectLayerClient(connectRpcInstance.ins.Conn)
	_ctx, cancel := context.WithTimeout(context.Background(), time.Second*5)
	defer cancel()
	_, err = connectClient.PushFriendOnlineMsg(_ctx, &proto.PushFriendOnlineMsgReq{
		Msg: payload.Msg.(*proto.PushFriendOnlineMsgReq_Msg),
		KafkaInfo: &proto.KafkaMsgInfo{
			Topic:     msg.Topic,
			Partition: int32(msg.Partition),
			Offset:    msg.Offset,
		},
	})
	if err != nil {
		zlog.Error(err.Error())
	}
}

func (task *Task) pushFriendOfflineMsg(serverId string, msg *kafka.Message) {
	var err error
	connectRpcInstance.ins, err = serDiscovery.GetServiceByServerId(serverId)
	if err != nil {
		zlog.Error(err.Error())
		return
	}
	var payload common.MsgSend
	payload.Msg = new(proto.PushFriendOfflineMsgReq_Msg)
	connectClient := proto.NewConnectLayerClient(connectRpcInstance.ins.Conn)
	_ctx, cancel := context.WithTimeout(context.Background(), time.Second*5)
	defer cancel()
	_, err = connectClient.PushFriendOfflineMsg(_ctx, &proto.PushFriendOfflineMsgReq{
		Msg: payload.Msg.(*proto.PushFriendOfflineMsgReq_Msg),
		KafkaInfo: &proto.KafkaMsgInfo{
			Topic:     msg.Topic,
			Partition: int32(msg.Partition),
			Offset:    msg.Offset,
		},
	})
	if err != nil {
		zlog.Error(err.Error())
	}
}

func (task *Task) pushGroupMsg(serverId string, msg *kafka.Message) {
	var err error
	connectRpcInstance.ins, err = serDiscovery.GetServiceByServerId(serverId)
	if err != nil {
		zlog.Error(err.Error())
		return
	}
	var payload common.MsgSend
	payload.Msg = new(proto.PushGroupMsgReq_Msg)
	_ = json.Unmarshal(msg.Value, &payload)

	// todo 为保证消息持久化到db后到时序性，采用分布式系统中常用的snow flake ID的方法
	payload.Msg.(*proto.PushGroupMsgReq_Msg).SnowId = utils.GetSnowflakeId()

	connectClient := proto.NewConnectLayerClient(connectRpcInstance.ins.Conn)
	_ctx, cancel := context.WithTimeout(context.Background(), time.Second*5)
	defer cancel()
	_, err = connectClient.PushGroupMsg(_ctx, &proto.PushGroupMsgReq{
		Msg: payload.Msg.(*proto.PushGroupMsgReq_Msg),
		KafkaInfo: &proto.KafkaMsgInfo{
			Topic:     msg.Topic,
			Partition: int32(msg.Partition),
			Offset:    msg.Offset,
		},
	})
	if err != nil {
		zlog.Error(err.Error())
	}
}

func (task *Task) pushFriendMsg(serverId string, msg *kafka.Message) {
	var err error
	connectRpcInstance.ins, err = serDiscovery.GetServiceByServerId(serverId)
	if err != nil {
		zlog.Error(err.Error())
		return
	}
	var payload common.MsgSend
	payload.Msg = new(proto.PushFriendMsgReq_Msg)
	_ = json.Unmarshal(msg.Value, &payload)

	// todo 为保证消息持久化到db后到时序性，采用分布式系统中常用的snow flake ID的方法
	payload.Msg.(*proto.PushFriendMsgReq_Msg).SnowId = utils.GetSnowflakeId()

	connectClient := proto.NewConnectLayerClient(connectRpcInstance.ins.Conn)
	_ctx, cancel := context.WithTimeout(context.Background(), time.Second*5)
	defer cancel()
	_, err = connectClient.PushFriendMsg(_ctx, &proto.PushFriendMsgReq{
		Msg: payload.Msg.(*proto.PushFriendMsgReq_Msg),
		KafkaInfo: &proto.KafkaMsgInfo{
			Topic:     msg.Topic,
			Partition: int32(msg.Partition),
			Offset:    msg.Offset,
		},
	})
	if err != nil {
		zlog.Error(err.Error())
	}
}
