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
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
	"math/rand"
	"strings"
	"sync"
	"time"
)

type LogicRpcClient struct {
	mutex  sync.RWMutex
	Client proto.LogicClient
	cancel context.CancelFunc
	ctx    context.Context
	conn   *grpc.ClientConn
	ser    *etcd.ServiceDiscovery
}

var (
	logicRpcClient *LogicRpcClient
	once           sync.Once
)

// InitLogicClient 初始化获取logic层的rpc服务客户端
func (c *Connect) InitLogicClient() {
	conf := config.GetConfig()
	etcdAddrList := strings.Split(conf.Common.Etcd.Address, ";")
	logicRpcClient.ser = etcd.NewServiceDiscovery(etcdAddrList)
	//TODO：从etcd中 获取所有logic layer 的服务地址列表，并监听其改变
	err := logicRpcClient.ser.WatchService(fmt.Sprintf("%s/%s", conf.Common.Etcd.BasePath, conf.Common.Etcd.ServerPathLogic))
	rand.Seed(time.Now().Unix())
	if err != nil {
		panic(err)
	}
	once.Do(func() {
		logicRpcList := logicRpcClient.ser.GetServices()
		idx := rand.Intn(len(logicRpcList))
		addr := logicRpcList[idx]
		if len(logicRpcList) == 0 {
			err = logicRpcClient.ser.Close()
			if err != nil {
				zlog.Error(err.Error())
			}
			panic(errors.New("logic layer 没有可用服务"))
		}
		logicRpcClient.conn, err = grpc.Dial(addr, grpc.WithTransportCredentials(insecure.NewCredentials()))
		if err != nil {
			panic(err)
		}
		logicRpcClient.Client = proto.NewLogicClient(logicRpcClient.conn)
		logicRpcClient.ctx, logicRpcClient.cancel = context.WithTimeout(context.Background(), time.Second*5)
		// todo:实时监听logic层的服务变化，随机选择logic层某个可用的服务
		go logicRpcClient.randomChoiceLogicRpcCanUseList()
	})
	if logicRpcClient.conn == nil {
		panic("初始化logic层服务客户端失败")
	}
}

func (lc *LogicRpcClient) randomChoiceLogicRpcCanUseList() {
	defer func() {
		lc.cancel()
		err := lc.conn.Close()
		if err != nil {
			zlog.Error(err.Error())
		}
		err = lc.ser.Close()
		if err != nil {
			zlog.Error(err.Error())
		}
	}()
	for {
		select {
		case <-time.Tick(10 * time.Second):
			// todo 持有读锁读goroutine可用进行读和写操作
			lc.mutex.RLock()
			logicRpcList := lc.ser.GetServices()
			if len(logicRpcList) == 0 {
				panic(errors.New("logic layer 没有可用服务"))
			}
			idx := rand.Intn(len(logicRpcList))
			addr := logicRpcList[idx]
			var err error
			oldConn := logicRpcClient.conn
			oldSer := logicRpcClient.ser
			oldCancel := logicRpcClient.cancel
			logicRpcClient.conn, err = grpc.Dial(addr, grpc.WithTransportCredentials(insecure.NewCredentials()))
			if err != nil {
				panic(err)
			}
			oldCancel()
			err = oldConn.Close()
			if err != nil {
				zlog.Error(err.Error())
			}
			err = oldSer.Close()
			if err != nil {
				zlog.Error(err.Error())
			}
			lc.mutex.RUnlock()
		}
	}
}

type ServerConnect struct{}

func (sc *ServerConnect) PushGroupInfoMsg(ctx context.Context, req *proto.PushGroupInfoMsgReq) (reply *empty.Empty, err error) {
	if req == nil {
		err = errors.New("req *proto.PushGroupInfoMsgReq == nil")
		zlog.Error(err.Error())
		return
	}
	msgBody, _ := json.Marshal(req.Msg)
	kafkaMsg := kafka.Message{
		Topic:     req.KafkaInfo.Topic,
		Partition: int(req.KafkaInfo.Partition),
		Offset:    req.KafkaInfo.Offset,
		Value:     msgBody,
	}
	// todo 遍历bucket推送群聊消息，因为同一个群的成员可以分散在不同的bucket中，故不同bucket中可能有同一个groupNode
	for _, bucket := range DefaultServer.Buckets {
		bucket.BroadcastRoom(kafkaMsg)
	}
	return
}

func (sc *ServerConnect) PushGroupCountMsg(ctx context.Context, req *proto.PushGroupCountMsgReq) (reply *empty.Empty, err error) {
	if req == nil {
		err = errors.New("req *proto.PushGroupCountMsgReq == nil")
		zlog.Error(err.Error())
		return
	}
	msgBody, _ := json.Marshal(req.Msg)
	kafkaMsg := kafka.Message{
		Topic:     req.KafkaInfo.Topic,
		Partition: int(req.KafkaInfo.Partition),
		Offset:    req.KafkaInfo.Offset,
		Value:     msgBody,
	}
	// todo 遍历bucket推送群聊消息，因为同一个群的成员可以分散在不同的bucket中，故不同bucket中可能有同一个groupNode
	for _, bucket := range DefaultServer.Buckets {
		bucket.BroadcastRoom(kafkaMsg)
	}
	return
}
func (sc *ServerConnect) PushFriendOnlineMsg(ctx context.Context, req *proto.PushFriendOnlineMsgReq) (reply *empty.Empty, err error) {
	// TODO:因为参数不一样所以分开写
	if req == nil {
		err = errors.New("req *proto.PushFriendOnlineMsgReq == nil")
		zlog.Error(err.Error())
		return
	}
	msgBody, _ := json.Marshal(req.Msg)
	kafkaMsg := kafka.Message{
		Topic:     req.KafkaInfo.Topic,
		Partition: int(req.KafkaInfo.Partition),
		Offset:    req.KafkaInfo.Offset,
		Value:     msgBody,
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
	if req == nil {
		err = errors.New("req *proto.PushFriendOfflineMsgReq == nil")
		zlog.Error(err.Error())
		return
	}
	msgBody, _ := json.Marshal(req.Msg)
	kafkaMsg := kafka.Message{
		Topic:     req.KafkaInfo.Topic,
		Partition: int(req.KafkaInfo.Partition),
		Offset:    req.KafkaInfo.Offset,
		Value:     msgBody,
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
	if req == nil {
		err = errors.New("req *proto.PushGroupMsgReq == nil")
		zlog.Error(err.Error())
		return
	}
	msgBody, _ := json.Marshal(req.Msg)
	kafkaMsg := kafka.Message{
		Topic:     req.KafkaInfo.Topic,
		Partition: int(req.KafkaInfo.Partition),
		Offset:    req.KafkaInfo.Offset,
		Value:     msgBody,
	}
	// todo 遍历bucket推送群聊消息，因为同一个群的成员可以分散在不同的bucket中，故不同bucket中可能有同一个groupNode
	for _, bucket := range DefaultServer.Buckets {
		bucket.BroadcastRoom(kafkaMsg)
	}
	return
}

func (sc *ServerConnect) PushFriendMsg(ctx context.Context, req *proto.PushFriendMsgReq) (reply *empty.Empty, err error) {
	if req == nil {
		zlog.Error("req *proto.PushFriendMsgReq == nil")
		return
	}
	msgBody, _ := json.Marshal(req.Msg)
	kafkaMsg := kafka.Message{
		Topic:     req.KafkaInfo.Topic,
		Partition: int(req.KafkaInfo.Partition),
		Offset:    req.KafkaInfo.Offset,
		Value:     msgBody,
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
