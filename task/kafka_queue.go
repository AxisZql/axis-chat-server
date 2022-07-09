package task

import (
	"axisChat/common"
	"axisChat/utils/zlog"
	"context"
	"encoding/json"
	"fmt"
	"github.com/segmentio/kafka-go"
	"strconv"
	"strings"
	"sync"
	"time"
)

/*
*Author:AxisZql
*Date:2022-6-14 2:50 PM
*Desc: Push online group chats and user message to  Connect layer
 */

type OnlineObject struct {
	mutex       sync.RWMutex
	OnlineTopic map[string]struct{}
}

type ObjTrigger struct {
	mutex         sync.RWMutex
	preOffTrigger map[string]chan int64 // kafka topic 监听进程销毁触发器
}

var (
	onlineObj = &OnlineObject{
		OnlineTopic: make(map[string]struct{}),
	}

	objTrigger = &ObjTrigger{
		preOffTrigger: make(map[string]chan int64),
	}

	newTopicTarget = make(chan string, 5)
)

// UpdateOnlineObjTrigger 定时更新在线群聊和用户当id列表
func (task *Task) UpdateOnlineObjTrigger() {
	// 前一次更新完毕后才能进行下一次更新
	done := make(chan struct{})
	// 运行程序时首先获取所有在线对象信息
	go task.updateOnlineObj(done)

	// 每10秒检测异常redis更新在线用户和群聊,弥补redis方式只能在上线初期触发的缺陷
	ticker := time.NewTicker(5 * time.Second)
	for {
		select {
		case <-ticker.C:
			select {
			case _ = <-done:
				go task.updateOnlineObj(done)
			}
		}
	}
}

func (task *Task) updateOnlineObj(done chan struct{}) {
	groupMap, err := common.RedisHGetAll(common.GroupOnlineUserCount)
	if err != nil {
		zlog.Error(err.Error())
	}
	userMap, err := common.RedisHGetAll(common.AllOnlineUser)
	if err != nil {
		zlog.Error(err.Error())
	}

	newTopicMap := make(map[string]struct{})

	// 更新阶段不允许其他进程进行修改
	for k, v := range groupMap {
		count, _ := strconv.Atoi(v)
		groupId, _ := strconv.Atoi(k)
		if count != 0 {
			topic := fmt.Sprintf(common.GroupQueuePrefix, groupId)
			newTopicMap[topic] = struct{}{}
			onlineObj.mutex.RLock()
			_, ok := onlineObj.OnlineTopic[topic]
			onlineObj.mutex.RUnlock()
			if !ok {
				// 更新在线表
				onlineObj.mutex.Lock()
				onlineObj.OnlineTopic[topic] = struct{}{}
				onlineObj.mutex.Unlock()
				// 如果是新上线的Group

				newTopicTarget <- topic
			}
		}
	}
	for k, v := range userMap {
		userId, _ := strconv.Atoi(k)
		if v == "on" {
			topic := fmt.Sprintf(common.FriendQueuePrefix, userId)
			newTopicMap[topic] = struct{}{}

			onlineObj.mutex.RLock()
			_, ok := onlineObj.OnlineTopic[topic]
			onlineObj.mutex.RUnlock()
			if !ok {
				// 如果是新上线的User
				onlineObj.mutex.Lock()
				onlineObj.OnlineTopic[topic] = struct{}{}
				onlineObj.mutex.Unlock()

				newTopicTarget <- topic
			}
		}
	}

	var oldTopic []string
	onlineObj.mutex.RLock()
	for topic := range onlineObj.OnlineTopic {
		if _, ok := newTopicMap[topic]; !ok {

			objTrigger.mutex.RLock()
			objTrigger.preOffTrigger[topic] <- 1
			objTrigger.mutex.RUnlock()

			oldTopic = append(oldTopic, topic)
		}
	}
	onlineObj.mutex.RUnlock()

	for _, topic := range oldTopic {
		onlineObj.mutex.Lock()
		delete(onlineObj.OnlineTopic, topic)
		onlineObj.mutex.Unlock()
	}
	done <- struct{}{}
}

//todo bug 用户下线后必须把对应的topic监听goroutine回收，否则当用户下线后又上线时，监听同一个topic的goroutine 数目就会大于1

// startTopic 如果有新当群聊或者用户上线时，则开始监听其对应的topic
func (task *Task) startTopic() {
	for {
		select {
		case topic := <-newTopicTarget:
			// 初始化
			objTrigger.mutex.Lock()
			// 由于无缓冲channel要求发送和接收方同时运行，所以改为有缓冲
			objTrigger.preOffTrigger[topic] = make(chan int64, 1)
			objTrigger.mutex.Unlock()

			strList := strings.Split(topic, "_")
			var ty string
			id, _ := strconv.Atoi(strList[2])
			if strList[0] == "group" {
				ty = "group"
			} else if strList[0] == "friend" {
				ty = "user"
			}
			// 创建kafka消费者实例后开始不断监听和读取目标topic中的消息
			go task.fetchMsgFromTopic(ty, int64(id), topic)
		}
	}
}

type kafkaReader struct {
	mutex  sync.RWMutex
	reader *kafka.Reader
}

// fetchMsgFromTopic 当对象在在线表onlineObj中时，如果connect消费了前一个消息则开始推送下一条消息
func (task *Task) fetchMsgFromTopic(ty string, id int64, topic string) {
	defer func() {
		objTrigger.mutex.Lock()
		defer objTrigger.mutex.Unlock()
		delete(onlineObj.OnlineTopic, topic)
	}()

	//群聊消息，只要被一个reader正确提交偏移量就算成功消费，故对应一个群聊消息的消费组只有一个
	var consumerSuffix string
	switch ty {
	case "group":
		consumerSuffix = fmt.Sprintf("groupid-%d", id)
	case "user":
		consumerSuffix = fmt.Sprintf("userid-%d", id)
	}
	redisLock, err := common.NewRedisLocker(fmt.Sprintf(common.RedisLock, topic), topic)
	//设置锁的过期时间为60s+500ms
	redisLock.SetExpire(60)
	if err != nil {
		zlog.Error(fmt.Sprintf("init redis lock is failure %v", err))
		return
	}

	var (
		kr        kafkaReader
		hasCommit common.KafkaMsgInfo
		offset    int64
	)
	defer func() {
		kr.mutex.Lock()
		defer kr.mutex.Unlock()
		err = kr.reader.Close()
		if err != nil {
			zlog.Error(err.Error())
		}
	}()

	// 初始化或者reader中途close时重新打开
	initKafkaReader := func() {
		kr.mutex.Lock()
		defer kr.mutex.Unlock()

		kr.reader, err = common.GetConsumeReader(consumerSuffix, topic)
		if err != nil {
			zlog.Error(err.Error())
			return
		}
		//根据redis上一次读取的最新的偏移量，来从对应topic中获取最新未被读取的消息
		var res []byte
		res, err = common.RedisGetString(fmt.Sprintf(common.KafkaTopicOffset, topic))
		if err != nil {
			zlog.Error(fmt.Sprintf("common.RedisGetString(fmt.Sprintf(common.KafkaTopicOffset, %s))", topic))
			return
		}

		_ = json.Unmarshal(res, &hasCommit)
		offset = hasCommit.Offset
		if offset != 0 {
			// 在Kafka设置正常的偏移量，以redis为准
			err = kr.reader.CommitMessages(context.Background(), kafka.Message{
				Topic:     hasCommit.Topic,
				Partition: hasCommit.Partition,
				Offset:    offset, //因为CommitMessage提交的偏移量是在Msg的基础上加1，所以-1
			})
			if err != nil {
				zlog.Error(err.Error())
				return
			}
		}
	}

	// todo 在程序运行前必须保证connect释放分布式锁，确保之前的消息被持久化到db中
	for {
		lock, _ := redisLock.Acquire()
		if !lock {
			// 没有获取分布式锁则继续获取
			time.Sleep(time.Millisecond * 50)
			continue
		} else {
			break
		}
	}

	initKafkaReader()
	_, _ = redisLock.Release() // 初始化reader后，释放分布式锁

	var (
		ctx    context.Context
		cancel context.CancelFunc
	)

	// 第一次投递消息
	ctx, cancel = context.WithCancel(context.Background())
	fetchNext := make(chan struct{}) //触发topic中下一个偏移量消息的获取
	resChannel := make(chan kafkaResp)
	go task.fetchNextOffsetMsg(ctx, kr.reader, fetchNext, resChannel, topic)
	fetchNext <- struct{}{} // 推动量,开始监听对应topic
	defer func() {
		cancel()
		close(fetchNext)
		close(resChannel)
	}()

	quit := make(chan struct{})

	objTrigger.mutex.RLock()
	ch := objTrigger.preOffTrigger[topic]
	objTrigger.mutex.RUnlock()

	var (
		curOffset int64
	)

	go func() {
		for {
			select {
			case <-ch:
				objTrigger.mutex.Lock()
				delete(objTrigger.preOffTrigger, topic)
				objTrigger.mutex.Unlock()
				_, _ = redisLock.Release()
				quit <- struct{}{}
				return

			case kResp, ok := <-resChannel:
				if !ok {
					break
				}

				if kResp.err != nil {
					zlog.Error(fmt.Sprintf("%v", err))
					initKafkaReader()

					// 出现异常后重新初始化监听对应的topic,并销毁之前监听的进程
					cancel()
					ctx, cancel = context.WithCancel(context.Background())
					go task.fetchNextOffsetMsg(ctx, kr.reader, fetchNext, resChannel, topic)
					fetchNext <- struct{}{}
					break
				}

				// 循环获取分布式锁
				// 这里可以判定如果投递消息后connect层没有成功消费，则分布式锁必须等到60秒才会被释放，所以通过这个特征判定消息是否成功投递
				checkPreSend := time.NewTicker(30 * time.Second)

			over:
				for {
					select {
					case <-checkPreSend.C:
						// 如果获取分布式锁超过30s则证明前一次投递消息失败
						if curOffset > 0 {
							zlog.Error(fmt.Sprintf("topic = %s 消息投递超时，触发超时重传机制", topic))
							initKafkaReader()
							cancel()
							ctx, cancel = context.WithCancel(context.Background())
							go task.fetchNextOffsetMsg(ctx, kr.reader, fetchNext, resChannel, topic)
							fetchNext <- struct{}{}
						}
						fetchNext <- struct{}{} // 应该由fetchNextOffsetMsg抛出异常然后触发reader重新初始化
						break over
					default:
						lock, _ := redisLock.Acquire()
						if !lock {
							// 没有获取分布式锁则继续获取
							time.Sleep(time.Millisecond * 50)
							continue
						} else {
							// todo 这里会一直阻塞，直到成功投递消息或者抛出异常
							task.Push(&kResp.msg)
							curOffset = kResp.msg.Offset
							fetchNext <- struct{}{}
							break over
						}
					}

				}
			}
		}
	}()

	<-quit // 优雅结束
	zlog.Debug(fmt.Sprintf("fetchMsgFromTopic 成功推出 topic=%v 的监听", topic))
}

type kafkaResp struct {
	msg kafka.Message
	err error
}

func (task *Task) fetchNextOffsetMsg(ctx context.Context, reader *kafka.Reader, fetchNext chan struct{}, resChannel chan kafkaResp, topic string) {
	zlog.Debug(topic)
	defer zlog.Debug(fmt.Sprintf("fetchNextOffsetMsg「退出」topic=%s 的监听", topic))
	for {
		select {
		case <-fetchNext: // 前一次消息被消费成功才能进行新对消息推送
			select {
			case <-ctx.Done():
				return
			default:
				begin := time.Now()
				msg, err := reader.FetchMessage(context.Background())

				select {
				case _, ok := <-resChannel:
					// 确保父goroutine没有结束
					if !ok {
						return
					}
				default:
					resChannel <- kafkaResp{
						msg: msg,
						err: err,
					}
				}
				zlog.Info(fmt.Sprintf("「 fetch Msg 」 topic = %s cur offset = %v Spend time = %v", msg.Topic, msg.Offset, time.Since(begin)))
			}
		}
	}
}
