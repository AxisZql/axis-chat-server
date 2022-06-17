package task

import (
	"axisChat/common"
	"axisChat/utils/zlog"
	"fmt"
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

	newTopicTarget = make(chan string, 5)
	once           sync.Once
)

// UpdateOnlineObjTrigger 定时更新在线群聊和用户当id列表
func (task *Task) UpdateOnlineObjTrigger() {
	once.Do(func() {
		// 运行程序时首先获取所有在线对象信息
		task.updateOnlineObj()
	})
	// 每5秒检测异常redis更新在线用户和群聊
	ticker := time.NewTicker(5 * time.Second)
	for {
		select {
		case <-ticker.C:
			task.updateOnlineObj()
		}
	}
}

func (task *Task) updateOnlineObj() {
	groupMap, err := common.RedisHGetAll(common.GroupOnlineUserCount)
	if err != nil {
		zlog.Error(err.Error())
	}
	userMap, err := common.RedisHGetAll(common.AllOnlineUser)
	if err != nil {
		zlog.Error(err.Error())
	}
	newGroupIdMap := make(map[int64]struct{})

	onlineObj.mutex.RLock()
	for k, v := range groupMap {
		count, _ := strconv.Atoi(v)
		groupId, _ := strconv.Atoi(k)
		if count != 0 {
			if _, ok := onlineObj.GroupIdMap[int64(groupId)]; !ok {
				// 如果是新上线的Group
				newTopicTarget <- "group_" + k
				topic := fmt.Sprintf(common.GroupQueuePrefix, groupId)
				objOffTrigger[topic] = &Trigger{
					trigger: make(chan int64),
				}
			}
			newGroupIdMap[int64(groupId)] = struct{}{}
		}
	}
	newUserIdMap := make(map[int64]struct{})
	for k, v := range userMap {
		userId, _ := strconv.Atoi(k)
		if v == "on" {
			if _, ok := onlineObj.GroupIdMap[int64(userId)]; !ok {
				// 如果是新上线的User
				newTopicTarget <- "user_" + k
				topic := fmt.Sprintf(common.UserQueuePrefix, userId)
				objOffTrigger[topic] = &Trigger{
					trigger: make(chan int64),
				}
			}
			newUserIdMap[int64(userId)] = struct{}{}
		}
	}
	// todo 筛选出下线的对象,即不使用的topic监听goroutine
	for k := range onlineObj.GroupIdMap {
		if _, ok := newGroupIdMap[k]; !ok {
			topic := fmt.Sprintf(common.GroupQueuePrefix, k)
			objOffTrigger[topic].trigger <- k
		}
	}
	for k := range onlineObj.UserIdMap {
		if _, ok := newUserIdMap[k]; !ok {
			topic := fmt.Sprintf(common.UserQueuePrefix, k)
			objOffTrigger[topic].trigger <- k
		}
	}

	onlineObj.mutex.RUnlock()
	// 更新
	onlineObj.mutex.Lock()
	onlineObj.GroupIdMap = newGroupIdMap
	onlineObj.UserIdMap = newUserIdMap
	onlineObj.mutex.Unlock()
}

//todo bug 用户下线后必须把对应的topic监听goroutine回收，否则当用户下线后又上线时，监听同一个topic的goroutine 数目就会大于1

// startTopic 如果有新当群聊或者用户上线时，则开始监听其对应的topic
func (task *Task) startTopic() {
	for {
		select {
		case obj := <-newTopicTarget:
			strList := strings.Split(obj, "_")
			var topic string
			var ty string
			id, _ := strconv.Atoi(strList[1])
			if strList[0] == "group" {
				ty = "group"
				topic = fmt.Sprintf(common.UserQueuePrefix, id)
			} else if strList[0] == "user" {
				ty = "user"
				topic = fmt.Sprintf(common.GroupQueuePrefix, id)
			}
			// 创建kafka消费者实例后开始不断监听和读取目标topic中的消息
			go task.fetchMsgFromTopic(ty, int64(id), topic)
		}
	}
}

// fetchMsgFromTopic 当对象在在线表onlineObj中时，如果connect消费了前一个消息则开始推送下一条消息
func (task *Task) fetchMsgFromTopic(ty string, id int64, topic string) {
	connectIsCommit := make(chan bool)
	reader, err := common.GetConsumeReader(id, topic)
	if err != nil {
		zlog.Error(err.Error())
	}
	defer func() {
		err = reader.Close()
		if err != nil {
			zlog.Error(err.Error())
		}
	}()
	for {
		offset, err := common.RedisBRPOP(fmt.Sprintf(common.KafkaCommitTrigger, topic), 10*time.Second)
		if err != nil {
			zlog.Error(err.Error())
		}
		// todo 保证消息的时序性
		if offset != "" {
			connectIsCommit <- true
		}
		// todo：只有当对应用户在线时才完connect层推送对应的消息
		select {
		// todo: 只有当connect进行消息提交后才会继续从topic中读取消息
		case <-connectIsCommit:
			switch ty {
			case "group":
				onlineObj.mutex.RLock()
				_, ok := onlineObj.GroupIdMap[id]
				onlineObj.mutex.RUnlock()
				if ok {
					// todo 在线则推送消息
					msg, err := common.TopicConsume(reader)
					if err != nil {
						zlog.Error(err.Error())
					}
					zlog.Info(fmt.Sprintf("pre offset = %v, cur offset = %v", offset, msg.Offset))
					task.Push(&msg)
				}
			case "user":
				onlineObj.mutex.RLock()
				_, ok := onlineObj.UserIdMap[id]
				onlineObj.mutex.RUnlock()
				if ok {
					// todo 在线则推送消息
					msg, err := common.TopicConsume(reader)
					if err != nil {
						zlog.Error(err.Error())
					}
					zlog.Info(fmt.Sprintf("pre offset = %v, cur offset = %v", offset, msg.Offset))
					task.Push(&msg)
				}
			}

		case <-objOffTrigger[topic].trigger:
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
