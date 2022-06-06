package connect

import (
	"axisChat/common"
	"axisChat/utils/zlog"
	"encoding/json"
	"fmt"
	"github.com/segmentio/kafka-go"
	"sync"
	"sync/atomic"
)

/**
*Author: AxisZql
*Date: 2022-6-3
*DESC: connect layer 令牌桶限流算法
 */

type Bucket struct {
	mutex      sync.RWMutex
	socketMap  map[int64]*Channel // userid和conn之间的映射
	routineIdx uint64
	routineNum uint64
	routines   []chan kafka.Message // 所有要进行消息推送的消息在channel队列中排队
	GroupNode  map[int64]*GroupNode
}

type bucketOptions struct {
	SocketSize    int // 每个bucket最多负责的连接数目
	RoutineAmount int // 请求队列的数目
	RoutineSize   int // 每个请求队列中最多包含的请求数目
}

const (
	defaultSocketSize    = 1024
	defaultRoutineAmount = 32
	defaultRoutineSize   = 20
	NoGroup              = -1
)

type BucketOption interface {
	apply(options *bucketOptions)
}

type bucketOptionFunc func(options *bucketOptions)

func (f bucketOptionFunc) apply(options *bucketOptions) {
	f(options)
}

func BucketWithSocketSize(size int) BucketOption {
	return bucketOptionFunc(func(options *bucketOptions) {
		options.SocketSize = size
	})
}

func BucketWithRoutineAmount(size int) BucketOption {
	return bucketOptionFunc(func(options *bucketOptions) {
		options.RoutineAmount = size
	})
}

func BucketWithRoutineSize(size int) BucketOption {
	return bucketOptionFunc(func(options *bucketOptions) {
		options.RoutineSize = size
	})
}

func NewBucket(option ...BucketOption) (b *Bucket) {
	options := bucketOptions{
		SocketSize:    defaultSocketSize,
		RoutineAmount: defaultRoutineAmount,
		RoutineSize:   defaultRoutineSize,
	}
	for _, o := range option {
		o.apply(&options)
	}
	b.socketMap = make(map[int64]*Channel, options.SocketSize)
	b.routines = make([]chan kafka.Message, options.RoutineAmount)
	b.routineNum = uint64(options.RoutineAmount)
	b.routineIdx = 0
	for i := 0; i < options.RoutineAmount; i++ {
		c := make(chan kafka.Message, options.RoutineSize)
		b.routines[i] = c
		// TODO: 开始监听每个请求队列发送过来的消息
		go b.PushGroupMsg(c)
	}
	return
}

func (b *Bucket) BroadcastRoom(msg kafka.Message) {
	//TODO:原子加法，将推送消息给对应群聊的请求，按照顺序分发到不同的routines缓冲区
	idx := atomic.AddUint64(&b.routineIdx, 1) % b.routineNum
	b.routines[idx] <- msg
}

//PushGroupMsg 专门处理群聊消息推送
func (b *Bucket) PushGroupMsg(ch chan kafka.Message) {
	for {
		var msg kafka.Message
		select {
		// TODO: MQ中读取的数据会写入ch中
		case msg = <-ch:
			// TODO：消息进入处理队列中，并被取出时必须马上进行消费
			var payload common.MsgSend
			_ = json.Unmarshal(msg.Value, &payload)
			switch payload.Op {
			case common.OpGroupMsgSend:
				payload.Msg = new(common.GroupMsg)
				_ = json.Unmarshal(msg.Value, &payload)
				groupId := payload.Msg.(common.GroupMsg).GroupId
				if groupNode := b.GetGroupNode(groupId); groupNode != nil {
					groupNode.PushGroupMsg(msg)
				} else {
					//TODO:对应群聊可能在其他BUCKET中
					zlog.Debug(fmt.Sprintf("the group of  groupId = %d not in the bucket", groupId))
				}

			case common.OpGroupOlineUserCountSend:
				payload.Msg = new(common.GroupCountMsg)
				_ = json.Unmarshal(msg.Value, &payload)
				groupId := payload.Msg.(common.GroupCountMsg).GroupId
				if groupNode := b.GetGroupNode(groupId); groupNode != nil {
					groupNode.PushGroupMsg(msg)
				} else {
					zlog.Debug(fmt.Sprintf("the group of  groupId = %d not in the bucket", groupId))
				}
			case common.OpGroupInfoSend:
				payload.Msg = new(common.GroupInfoMsg)
				_ = json.Unmarshal(msg.Value, &payload)
				groupId := payload.Msg.(common.GroupInfoMsg).GroupId
				if groupNode := b.GetGroupNode(groupId); groupNode != nil {
					groupNode.PushGroupMsg(msg)
				} else {
					zlog.Debug(fmt.Sprintf("the group of  groupId = %d not in the bucket", groupId))
				}
			default:
				zlog.Error(fmt.Sprintf("the msg get %v", payload))
			}
		}
	}
}

func (b *Bucket) PutChannel(userid int64, groupId int64, ch *Channel) (err error) {
	var (
		groupNode *GroupNode
		ok        bool
	)
	b.mutex.Lock()
	// 判断用户是否有加入群聊,如果没有则由Bucket直接管理对应Channel
	if groupId != NoGroup {
		if groupNode, ok = b.GroupNode[groupId]; !ok {
			groupNode = new(GroupNode)
			b.GroupNode[groupId] = groupNode
		}
		ch.GroupNodes = append(ch.GroupNodes, groupNode)
	}
	ch.Userid = userid
	b.socketMap[userid] = ch
	b.mutex.Unlock()
	// 如果当前用户有加入群聊则将该Channel加入对应GroupNode的双向链表中
	if groupNode != nil {
		err = groupNode.Put(ch)
	}
	return
}

// DeleteChanel 用户下线时调用
func (b *Bucket) DeleteChanel(ch *Channel) {
	var (
		groupNodeList []*GroupNode
		ok            bool
	)
	// 这里涉及对数据写操作，加读锁，而不加写锁的原因是
	// 因为有多个Channel是相互独立的，加写锁的化，其他用户就不能进行任何数据的读取了，造成整个消息推送阻塞
	b.mutex.RLock()
	if ch, ok = b.socketMap[ch.Userid]; ok {
		groupNodeList = ch.GroupNodes
		delete(b.socketMap, ch.Userid)
	}
	if len(groupNodeList) != 0 {
		for _, node := range groupNodeList {
			if node.DeleteChannel(ch) {
				// 如果当前GroupNode对应的Chanel为0，则表示该群的所有人均下线
				if node.drop == true {
					delete(b.GroupNode, node.groupId)
				}
			}
		}
	}
	b.mutex.RUnlock()
}

func (b *Bucket) GetGroupNode(groupId int64) (groupNode *GroupNode) {
	b.mutex.RLock()
	groupNode, _ = b.GroupNode[groupId]
	b.mutex.RUnlock()
	return
}

func (b *Bucket) GetChannel(userid int64) (ch *Channel) {
	b.mutex.RLock()
	ch = b.socketMap[userid]
	b.mutex.RUnlock()
	return
}
