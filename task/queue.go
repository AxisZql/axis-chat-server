package task

import (
	"axisChat/common"
	"axisChat/utils/zlog"
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
	mutex      sync.RWMutex
	GroupIdMap map[int64]struct{}
	UserIdMap  map[int64]struct{}
}

type Trigger struct {
	trigger chan int64
}

var (
	onlineObj = &OnlineObject{
		GroupIdMap: make(map[int64]struct{}),
		UserIdMap:  make(map[int64]struct{}),
	}

	objOffTrigger = make(map[string]*Trigger)
	preOffTrigger = make(map[string]*Trigger) // start方法结束后，调用start方法的函数也应该结束

	newTopicTarget = make(chan string, 5)
	//once           sync.Once
)

// UpdateOnlineObjTrigger 定时更新在线群聊和用户当id列表
func (task *Task) UpdateOnlineObjTrigger() {
	// 前一次更新完毕后才能进行下一次更新
	done := make(chan struct{})
	// 运行程序时首先获取所有在线对象信息
	go task.updateOnlineObj(done)
	// 每5秒检测异常redis更新在线用户和群聊
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

	newGroupIdMap := make(map[int64]struct{})

	// 更新阶段不允许其他进程进行
	onlineObj.mutex.Lock()
	for k, v := range groupMap {
		count, _ := strconv.Atoi(v)
		groupId, _ := strconv.Atoi(k)
		if count != 0 {
			newGroupIdMap[int64(groupId)] = struct{}{}
			if _, ok := onlineObj.GroupIdMap[int64(groupId)]; !ok {
				// 更新在线表
				onlineObj.GroupIdMap[int64(groupId)] = struct{}{}
				// 如果是新上线的Group
				topic := fmt.Sprintf(common.GroupQueuePrefix, groupId)
				newTopicTarget <- topic
			}
		}
	}
	newUserIdMap := make(map[int64]struct{})
	for k, v := range userMap {
		userId, _ := strconv.Atoi(k)
		if v == "on" {
			newUserIdMap[int64(userId)] = struct{}{}
			if _, ok := onlineObj.UserIdMap[int64(userId)]; !ok {
				onlineObj.UserIdMap[int64(userId)] = struct{}{}
				// 如果是新上线的User
				topic := fmt.Sprintf(common.FriendQueuePrefix, userId)
				newTopicTarget <- topic
			}
		}
	}
	onlineObj.mutex.Unlock()

	onlineObj.mutex.RLock()

	// todo 筛选出下线的对象,即不使用的topic监听goroutine
	var oldGroupId []int64
	for k := range onlineObj.GroupIdMap {
		if _, ok := newGroupIdMap[k]; !ok {
			topic := fmt.Sprintf(common.GroupQueuePrefix, k)
			objOffTrigger[topic].trigger <- k
			oldGroupId = append(oldGroupId, k)
		}
	}
	var oldUserid []int64
	for k := range onlineObj.UserIdMap {
		if _, ok := newUserIdMap[k]; !ok {
			topic := fmt.Sprintf(common.FriendQueuePrefix, k)
			objOffTrigger[topic].trigger <- k
			oldUserid = append(oldUserid, k)
		}
	}
	onlineObj.mutex.RUnlock()

	onlineObj.mutex.Lock()
	for _, k := range oldGroupId {
		delete(onlineObj.GroupIdMap, k)
	}
	for _, k := range oldUserid {
		delete(onlineObj.UserIdMap, k)
	}
	onlineObj.mutex.Unlock()
	done <- struct{}{}
}

//todo bug 用户下线后必须把对应的topic监听goroutine回收，否则当用户下线后又上线时，监听同一个topic的goroutine 数目就会大于1

// startTopic 如果有新当群聊或者用户上线时，则开始监听其对应的topic
func (task *Task) startTopic() {
	for {
		select {
		case topic := <-newTopicTarget:
			objOffTrigger[topic] = &Trigger{
				trigger: make(chan int64),
			}
			preOffTrigger[topic] = &Trigger{
				trigger: make(chan int64),
			}
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

// fetchMsgFromTopic 当对象在在线表onlineObj中时，如果connect消费了前一个消息则开始推送下一条消息
func (task *Task) fetchMsgFromTopic(ty string, id int64, topic string) {
	connectIsCommit := make(chan struct{})
	done := make(chan struct{}) // todo 确保task已经将新消息推送到connect层

	// todo 群聊消息，只要被一个reader正确提交偏移量就算成功消费，故对应一个群聊消息的消费组只有一个
	var consumerSuffix string
	switch ty {
	case "group":
		consumerSuffix = fmt.Sprintf("groupid-%d", id)
	case "user":
		consumerSuffix = fmt.Sprintf("userid-%d", id)
	}
	reader, err := common.GetConsumeReader(consumerSuffix, topic)
	if err != nil {
		zlog.Error(err.Error())
		return
	}
	//根据redis上一次读取的最新的偏移量，来从对应topic中获取最新未被读取的消息
	res, err := common.RedisGetString(fmt.Sprintf(common.KafkaTopicOffset, topic))
	if err != nil {
		zlog.Error(fmt.Sprintf("common.RedisGetString(fmt.Sprintf(common.KafkaTopicOffset, %s))", topic))
		return
	}

	redisLock, err := common.NewRedisLocker(fmt.Sprintf(common.RedisLock, topic), topic)
	// todo 设置锁的过期时间为100s+500ms
	redisLock.SetExpire(60)
	if err != nil {
		zlog.Error(fmt.Sprintf("init redis lock is failure %v", err))
		return
	}

	// 从最近一次提交的偏移量处读取消息
	pm, err := common.TopicConsume(reader)
	if err != nil {
		zlog.Error(fmt.Sprintf("初始化推送topic=%s 消息失败 err=%v", topic, err))
		return
	}
	offset, _ := strconv.Atoi(string(res))
	for pm.Offset <= int64(offset) {
		pm, err = common.TopicConsume(reader)
		if err != nil {
			zlog.Error(fmt.Sprintf("初始化推送topic=%s 消息失败 err=%v", topic, err))
			return
		}
	}

	// 第一次循环获取分布式锁获取锁成功后才能进行第一次消息推送
	var lock bool
	for {
		lock, _ = redisLock.Acquire()
		if !lock {
			// 没有获取分布式锁则继续获取
			time.Sleep(time.Millisecond * 50)
			continue
		} else {
			break
		}
	}
	//第一次推送消息
	task.Push(&pm)
	// 第一次循环获取分布式锁,要第二次获取分布式锁成功后才能进行第二条消息的推送
	for {
		lock, _ = redisLock.Acquire()
		if !lock {
			// 没有获取分布式锁则继续获取
			time.Sleep(time.Millisecond * 50)
			continue
		} else {
			break
		}
	}
	// 释放分布式锁
	_, _ = redisLock.Release()
	defer func() {
		err = reader.Close()
		if err != nil {
			zlog.Error(err.Error())
		}
	}()

	go task.start(connectIsCommit, done, ty, id, reader, topic)
	connectIsCommit <- struct{}{} //程序刚开始启动时进行第一条消息到推送
	for {
		select {
		case <-preOffTrigger[topic].trigger:
			delete(preOffTrigger, topic)
			return
		case <-done:
			// 前一条消息成功推送到connect层时才开始继续获取分布式锁
			//todo 循环获取分布式锁
			for {
				lock, _ = redisLock.Acquire()
				if !lock {
					// 没有获取分布式锁则继续获取
					time.Sleep(time.Millisecond * 50)
					continue
				} else {
					break
				}
			}
			connectIsCommit <- struct{}{}
		}
	}
}

func (task *Task) start(st, done chan struct{}, ty string, id int64, reader *kafka.Reader, topic string) {
	zlog.Debug("enter.........")
	// todo：只有当对应用户在线时才完connect层推送对应的消息

	for {
		select {
		// todo: 只有当connect进行消息提交后才会继续从topic中读取消息
		case <-st:
			switch ty {
			case "group":
				onlineObj.mutex.RLock()
				_, ok := onlineObj.GroupIdMap[id]
				onlineObj.mutex.RUnlock()
				if ok {
					// todo 在线则推送消息，获取消息的操作是阻塞操作
					go func() {
						msg, err := common.TopicConsume(reader)
						if err != nil {
							zlog.Error(err.Error())
						}
						zlog.Info(fmt.Sprintf(" cur offset = %v", msg.Offset))
						//todo bug,如果这次消息没有发送成功则会一直阻塞，没有发送成功那connect会抛异常断开ws连接，触发重连，进而触发
						// 该消息的重新投递
						task.Push(&msg)
						done <- struct{}{}
					}()
				}
			case "user":
				onlineObj.mutex.RLock()
				_, ok := onlineObj.UserIdMap[id]
				onlineObj.mutex.RUnlock()
				if ok {
					// todo 在线则推送消息
					go func() {
						msg, err := common.TopicConsume(reader)
						if err != nil {
							zlog.Error(err.Error())
						}
						zlog.Info(fmt.Sprintf(" cur offset = %v", msg.Offset))
						task.Push(&msg)
						done <- struct{}{}
					}()
				}
			}

		case <-objOffTrigger[topic].trigger:
			zlog.Error("触发下线。。。。。。。。。")
			preOffTrigger[topic].trigger <- 0
			// 对应obj下线触发
			switch ty {
			case "group":
				delete(objOffTrigger, topic)
				return
			case "user":
				delete(objOffTrigger, topic)
				return
			}
		}
	}
}
