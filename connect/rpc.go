package connect

import (
	"axisChat/config"
	"axisChat/etcd"
	"axisChat/proto"
	"axisChat/utils/zlog"
	"context"
	"encoding/json"
	"fmt"
	"github.com/golang/protobuf/ptypes/empty"
	"github.com/pkg/errors"
	"github.com/segmentio/kafka-go"
	"strings"
	"time"
)

type LogicRpcInstance struct {
	ins      *etcd.Instance
	serverId string
}

var (
	serDiscovery     *etcd.ServiceDiscovery
	logicRpcInstance = &LogicRpcInstance{}
)

// InitLogicClient 初始化获取logic层的rpc服务客户端
func (c *Connect) InitLogicClient() {
	conf := config.GetConfig()
	etcdAddrList := strings.Split(conf.Common.Etcd.Address, ";")
	serDiscovery = etcd.NewServiceDiscovery(etcdAddrList)
	//TODO：从etcd中 获取所有logic layer 的服务地址列表，并监听其改变
	err := serDiscovery.WatchService(fmt.Sprintf("%s/%s", conf.Common.Etcd.BasePath, conf.Common.Etcd.ServerPathLogic))
	if err != nil {
		panic(err)
	}

	// 轮询获取对应serverId下的logic layer rpc服务实例
	logicRpcInstance.ins, err = serDiscovery.GetServiceByServerId(conf.LogicRpc.Logic.ServerId)
	if err != nil {
		panic(err.Error())
	}
	logicRpcInstance.serverId = conf.LogicRpc.Logic.ServerId
}

type ServerConnect struct{}

func (sc *ServerConnect) PushGroupInfoMsg(ctx context.Context, req *proto.PushGroupInfoMsgReq) (reply *empty.Empty, err error) {
	reply = new(empty.Empty)
	if req == nil {
		err = errors.New("req *proto.PushGroupInfoMsgReq == nil")
		zlog.Error(err.Error())
		return
	}

	header := make([]kafka.Header, 0)
	for _, val := range req.KafkaInfo.Headers {
		header = append(header, kafka.Header{
			Key:   val.Key,
			Value: val.Value,
		})
	}

	msgBody, _ := json.Marshal(req.Msg)
	_time, _ := time.ParseInLocation(time.RFC3339, req.KafkaInfo.Time, time.Local)
	kafkaMsg := kafka.Message{
		Topic:         req.KafkaInfo.Topic,
		Partition:     int(req.KafkaInfo.Partition),
		Offset:        req.KafkaInfo.Offset,
		HighWaterMark: req.KafkaInfo.High_WaterMark,
		Key:           req.KafkaInfo.Key,
		Value:         msgBody,
		Headers:       header,
		Time:          _time,
	}
	// todo 遍历bucket推送群聊消息，因为同一个群的成员可以分散在不同的bucket中，故不同bucket中可能有同一个groupNode
	for _, bucket := range DefaultServer.Buckets {
		bucket.BroadcastRoom(kafkaMsg)
	}
	return
}

func (sc *ServerConnect) PushGroupCountMsg(ctx context.Context, req *proto.PushGroupCountMsgReq) (reply *empty.Empty, err error) {
	reply = new(empty.Empty)
	if req == nil {
		err = errors.New("req *proto.PushGroupCountMsgReq == nil")
		zlog.Error(err.Error())
		return
	}
	header := make([]kafka.Header, 0)
	for _, val := range req.KafkaInfo.Headers {
		header = append(header, kafka.Header{
			Key:   val.Key,
			Value: val.Value,
		})
	}

	msgBody, _ := json.Marshal(req.Msg)
	_time, _ := time.ParseInLocation(time.RFC3339, req.KafkaInfo.Time, time.Local)
	kafkaMsg := kafka.Message{
		Topic:         req.KafkaInfo.Topic,
		Partition:     int(req.KafkaInfo.Partition),
		Offset:        req.KafkaInfo.Offset,
		HighWaterMark: req.KafkaInfo.High_WaterMark,
		Key:           req.KafkaInfo.Key,
		Value:         msgBody,
		Headers:       header,
		Time:          _time,
	}
	// todo 遍历bucket推送群聊消息，因为同一个群的成员可以分散在不同的bucket中，故不同bucket中可能有同一个groupNode
	for _, bucket := range DefaultServer.Buckets {
		bucket.BroadcastRoom(kafkaMsg)
	}
	return
}
func (sc *ServerConnect) PushFriendOnlineMsg(ctx context.Context, req *proto.PushFriendOnlineMsgReq) (reply *empty.Empty, err error) {
	reply = new(empty.Empty)
	// TODO:因为参数不一样所以分开写
	if req == nil {
		err = errors.New("req *proto.PushFriendOnlineMsgReq == nil")
		zlog.Error(err.Error())
		return
	}
	header := make([]kafka.Header, 0)
	for _, val := range req.KafkaInfo.Headers {
		header = append(header, kafka.Header{
			Key:   val.Key,
			Value: val.Value,
		})
	}

	msgBody, _ := json.Marshal(req.Msg)
	_time, _ := time.ParseInLocation(time.RFC3339, req.KafkaInfo.Time, time.Local)
	kafkaMsg := kafka.Message{
		Topic:         req.KafkaInfo.Topic,
		Partition:     int(req.KafkaInfo.Partition),
		Offset:        req.KafkaInfo.Offset,
		HighWaterMark: req.KafkaInfo.High_WaterMark,
		Key:           req.KafkaInfo.Key,
		Value:         msgBody,
		Headers:       header,
		Time:          _time,
	}
	// todo 获取目标推送用户所在的令牌桶
	bucket := DefaultServer.Bucket(req.Msg.FriendId)
	ch := bucket.GetChannel(req.Msg.FriendId)
	if ch != nil {
		ch.Push(kafkaMsg) // 向目标用户的消息发送缓冲区中写入消息
	} else {
		err = errors.New(fmt.Sprintf("bucket.GetChannel(req.Msg.FriendId=%d) get nil", req.Msg.FriendId))
		zlog.Error(err.Error())
	}
	return
}

func (sc *ServerConnect) PushFriendOfflineMsg(ctx context.Context, req *proto.PushFriendOfflineMsgReq) (reply *empty.Empty, err error) {
	reply = new(empty.Empty)
	if req == nil {
		err = errors.New("req *proto.PushFriendOfflineMsgReq == nil")
		zlog.Error(err.Error())
		return
	}
	header := make([]kafka.Header, 0)
	for _, val := range req.KafkaInfo.Headers {
		header = append(header, kafka.Header{
			Key:   val.Key,
			Value: val.Value,
		})
	}

	msgBody, _ := json.Marshal(req.Msg)
	_time, _ := time.ParseInLocation(time.RFC3339, req.KafkaInfo.Time, time.Local)
	kafkaMsg := kafka.Message{
		Topic:         req.KafkaInfo.Topic,
		Partition:     int(req.KafkaInfo.Partition),
		Offset:        req.KafkaInfo.Offset,
		HighWaterMark: req.KafkaInfo.High_WaterMark,
		Key:           req.KafkaInfo.Key,
		Value:         msgBody,
		Headers:       header,
		Time:          _time,
	}
	// todo 获取目标推送用户所在的令牌桶
	bucket := DefaultServer.Bucket(req.Msg.FriendId)
	ch := bucket.GetChannel(req.Msg.FriendId)
	if ch != nil {
		ch.Push(kafkaMsg) // 向目标用户的消息发送缓冲区中写入消息
	} else {
		err = errors.New(fmt.Sprintf("bucket.GetChannel(req.Msg.FriendId=%d) get nil", req.Msg.FriendId))
		zlog.Error(err.Error())
	}
	return
}

func (sc *ServerConnect) PushGroupMsg(ctx context.Context, req *proto.PushGroupMsgReq) (reply *empty.Empty, err error) {
	reply = new(empty.Empty)
	if req == nil {
		err = errors.New("req *proto.PushGroupMsgReq == nil")
		zlog.Error(err.Error())
		return
	}
	header := make([]kafka.Header, 0)
	for _, val := range req.KafkaInfo.Headers {
		header = append(header, kafka.Header{
			Key:   val.Key,
			Value: val.Value,
		})
	}

	msgBody, _ := json.Marshal(req.Msg)
	_time, _ := time.ParseInLocation(time.RFC3339, req.KafkaInfo.Time, time.Local)
	kafkaMsg := kafka.Message{
		Topic:         req.KafkaInfo.Topic,
		Partition:     int(req.KafkaInfo.Partition),
		Offset:        req.KafkaInfo.Offset,
		HighWaterMark: req.KafkaInfo.High_WaterMark,
		Key:           req.KafkaInfo.Key,
		Value:         msgBody,
		Headers:       header,
		Time:          _time,
	}
	// todo 遍历bucket推送群聊消息，因为同一个群的成员可以分散在不同的bucket中，故不同bucket中可能有同一个groupNode
	for _, bucket := range DefaultServer.Buckets {
		bucket.BroadcastRoom(kafkaMsg)
	}
	return
}

func (sc *ServerConnect) PushFriendMsg(ctx context.Context, req *proto.PushFriendMsgReq) (reply *empty.Empty, err error) {
	reply = new(empty.Empty)
	if req == nil {
		zlog.Error("req *proto.PushFriendMsgReq == nil")
		return
	}
	header := make([]kafka.Header, 0)
	for _, val := range req.KafkaInfo.Headers {
		header = append(header, kafka.Header{
			Key:   val.Key,
			Value: val.Value,
		})
	}

	msgBody, _ := json.Marshal(req.Msg)
	_time, _ := time.ParseInLocation(time.RFC3339, req.KafkaInfo.Time, time.Local)
	kafkaMsg := kafka.Message{
		Topic:         req.KafkaInfo.Topic,
		Partition:     int(req.KafkaInfo.Partition),
		Offset:        req.KafkaInfo.Offset,
		HighWaterMark: req.KafkaInfo.High_WaterMark,
		Key:           req.KafkaInfo.Key,
		Value:         msgBody,
		Headers:       header,
		Time:          _time,
	}
	// todo 获取目标推送用户所在的令牌桶
	bucket := DefaultServer.Bucket(req.Msg.FriendId)
	ch := bucket.GetChannel(req.Msg.FriendId)
	if ch != nil {
		ch.Push(kafkaMsg) // 向目标用户的消息发送缓冲区中写入消息
	} else {
		err = errors.New(fmt.Sprintf("bucket.GetChannel(req.Msg.FriendId=%d) get nil", req.Msg.FriendId))
		zlog.Error(err.Error())
	}
	return
}
