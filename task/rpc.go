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

// InitConnectRpcClient 初始化获取connect层的rpc服务客户端
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

func (task *Task) pushGroupInfoMsg(serverId string, msg *common.GroupInfoMsg) {
	var err error
	connectRpcInstance.ins, err = serDiscovery.GetServiceByServerId(serverId)
	if err != nil {
		zlog.Error(err.Error())
		return
	}

	var userArr []*proto.User
	for _, u := range msg.UserArr {
		t := &proto.User{
			Id:       u.ID,
			Username: u.Username,
			Avatar:   u.Avatar,
			Role:     int32(u.Role),
			Status:   int32(u.Status),
			Tag:      u.Tag,
			CreateAt: u.CreateAt.Format(time.RFC3339),
			UpdateAt: u.UpdateAt.Format(time.RFC3339),
		}
		userArr = append(userArr, t)
	}

	connectClient := proto.NewConnectLayerClient(connectRpcInstance.ins.Conn)
	_ctx, cancel := context.WithTimeout(context.Background(), time.Second*5)
	defer cancel()
	_, err = connectClient.PushGroupInfoMsg(_ctx, &proto.PushGroupInfoMsgReq{
		Msg: &proto.PushGroupInfoMsgReq_Msg{
			GroupId:       msg.GroupId,
			Count:         int32(msg.Count),
			UserArr:       userArr,
			Op:            int32(msg.Op),
			OnlineUserIds: msg.OnlineUserIds,
		},
	})
	if err != nil {
		zlog.Error(err.Error())
	}
}

func (task *Task) pushGroupCountMsg(serverId string, msg *common.GroupCountMsg) {
	var err error
	connectRpcInstance.ins, err = serDiscovery.GetServiceByServerId(serverId)
	if err != nil {
		zlog.Error(err.Error())
		return
	}

	connectClient := proto.NewConnectLayerClient(connectRpcInstance.ins.Conn)
	_ctx, cancel := context.WithTimeout(context.Background(), time.Second*5)
	defer cancel()
	_, err = connectClient.PushGroupCountMsg(_ctx, &proto.PushGroupCountMsgReq{
		Msg: &proto.PushGroupCountMsgReq_Msg{
			GroupId: msg.GroupId,
			Count:   int32(msg.Count),
			Op:      int32(msg.Op),
		},
	})
	if err != nil {
		zlog.Error(err.Error())
	}
}

func (task *Task) pushFriendOnlineMsg(serverId string, msg *common.FriendOnlineMsg) {
	var err error
	connectRpcInstance.ins, err = serDiscovery.GetServiceByServerId(serverId)
	if err != nil {
		zlog.Error(err.Error())
		return
	}

	connectClient := proto.NewConnectLayerClient(connectRpcInstance.ins.Conn)
	_ctx, cancel := context.WithTimeout(context.Background(), time.Second*5)
	defer cancel()
	_, err = connectClient.PushFriendOnlineMsg(_ctx, &proto.PushFriendOnlineMsgReq{
		Msg: &proto.PushFriendOnlineMsgReq_Msg{
			FriendId:   msg.FriendId,
			FriendName: msg.FriendName,
			Op:         int32(msg.Op),
			Belong:     msg.Belong,
		},
	})
	if err != nil {
		zlog.Error(err.Error())
	}
}

func (task *Task) pushFriendOfflineMsg(serverId string, msg *common.FriendOfflineMsg) {
	var err error
	connectRpcInstance.ins, err = serDiscovery.GetServiceByServerId(serverId)
	if err != nil {
		zlog.Error(err.Error())
		return
	}
	connectClient := proto.NewConnectLayerClient(connectRpcInstance.ins.Conn)
	_ctx, cancel := context.WithTimeout(context.Background(), time.Second*5)
	defer cancel()
	_, err = connectClient.PushFriendOfflineMsg(_ctx, &proto.PushFriendOfflineMsgReq{
		Msg: &proto.PushFriendOfflineMsgReq_Msg{
			Op:       int32(msg.Op),
			FriendId: msg.FriendId,
			Belong:   msg.Belong,
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
