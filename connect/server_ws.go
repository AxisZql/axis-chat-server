package connect

import (
	"axisChat/common"
	"axisChat/db"
	"axisChat/proto"
	"axisChat/utils/zlog"
	"encoding/json"
	"fmt"
	"github.com/gorilla/websocket"
	"time"
)

type MsgOp struct {
	Op int32 `json:"op"`
}

// writePump 监听从websocket中读取数据
func (ws *WsServer) writePump(ch *Channel) {
	var (
		err error
	)
	ticker := time.NewTicker(ws.Options.PingPeriod)
	commitTicker := time.NewTicker(10 * time.Second) // 每10持久化一次聊天记录

	defer func() {
		commitTicker.Stop()
		err := ch.Conn.Close()
		if err != nil {
			zlog.Error(err.Error())
		}
		saveChatDataToDb(ch)
	}()
	if err != nil {
		zlog.Error(fmt.Sprintf("get kafka consum reader err:%v", err))
		return
	}

	for {
		select {
		case msg, ok := <-ch.BroadcastStatus:
			go func() {
				err := ch.Conn.SetWriteDeadline(time.Now().Add(ws.Options.WriteWait))
				if err != nil {
					zlog.Warn(fmt.Sprintf("ch.Conn.SetWriteDeadline err : %v", err))
				}
				if !ok {
					// ok==false 证明channel被关闭
					zlog.Warn("SetWriteDeadline not ok")
					// 出现异常给客户端发送关闭连接消息包
					err = ch.Conn.WriteMessage(websocket.CloseMessage, []byte{})
					if err != nil {
						zlog.Warn(fmt.Sprintf("ch.Conn.WriteMessage err :%v", err))
					}
					return
				}
				// 设置消息数据格式
				w, err := ch.Conn.NextWriter(websocket.TextMessage)
				if err != nil {
					zlog.Warn(fmt.Sprintf("ch.Conn.NextWriter err %v", err))
					return
				}
				zlog.Debug(fmt.Sprintf("message write body:%s", string(msg)))

				_, err = w.Write(DealWebSocketResp(msg))
				if err = w.Close(); err != nil {
					zlog.Error(fmt.Sprintf("w.Close err :%v", err))
					return
				}
				if err != nil {
					zlog.Error(fmt.Sprintf("push status msg get err: %v", err))
				}
			}()

		case msg, ok := <-ch.BroadcastMsg:
			err := ch.Conn.SetWriteDeadline(time.Now().Add(ws.Options.WriteWait))
			if err != nil {
				zlog.Warn(fmt.Sprintf("ch.Conn.SetWriteDeadline err : %v", err))
			}
			if !ok {
				zlog.Warn("SetWriteDeadline not ok")
				// 出现异常给客户端发送关闭连接消息包
				err = ch.Conn.WriteMessage(websocket.CloseMessage, []byte{})
				if err != nil {
					zlog.Warn(fmt.Sprintf("ch.Conn.WriteMessage err :%v", err))
				}
				return
			}
			// 设置消息数据格式
			w, err := ch.Conn.NextWriter(websocket.TextMessage)
			if err != nil {
				zlog.Warn(fmt.Sprintf("ch.Conn.NextWriter err %v", err))
				return
			}
			zlog.Debug(fmt.Sprintf("message write body:%s", string(msg.Value)))

			_, err = w.Write(DealWebSocketResp(msg.Value))
			if err = w.Close(); err != nil {
				zlog.Error(fmt.Sprintf("w.Close err :%v", err))
				return
			}
			if err != nil {
				zlog.Error(fmt.Sprintf("push msg get err: %v", err))
			} else {
				// todo 消息消费成功后释放redis分布式锁
				redisLocker, _ := common.NewRedisLocker(fmt.Sprintf(common.RedisLock, msg.Topic), msg.Topic)
				// todo 如果当前消息被成功消费了offset>=msg.Offset 则不用提交确认消费的偏移量，因为消费后的消息已经持久化到db中了
				res, err := common.RedisGetString(fmt.Sprintf(common.KafkaTopicOffset, msg.Topic))
				if err != nil {
					zlog.Error(fmt.Sprintf("common.RedisGetString(fmt.Sprintf(common.KafkaTopicOffset, %s))", msg.Topic))
				}
				var hasCommit common.KafkaMsgInfo
				_ = json.Unmarshal(res, &hasCommit)
				offset := hasCommit.Offset
				if offset >= msg.Offset && offset != 0 {
					// 证明该消息已经被其他服务消费过（群聊消息会出现这种情况）
					zlog.Info(fmt.Sprintf("msg:%s have been consumed by other server 「curOffset=%d,preOffset=%d」", string(msg.Value), msg.Offset, offset))
					// todo 释放redis分布式锁
					_, _ = redisLocker.Release()
				} else {
					offsetInfo := common.KafkaMsgInfo{
						Topic:     msg.Topic,
						Partition: msg.Partition,
						Offset:    msg.Offset,
					}
					offsetInfoPayload, _ := json.Marshal(offsetInfo)
					// 以redis提交的偏移量为准
					err = common.RedisSetString(fmt.Sprintf(common.KafkaTopicOffset, msg.Topic), offsetInfoPayload, 0)
					if err != nil {
						zlog.Error(fmt.Sprintf("common.RedisSetString(fmt.Sprintf(common.KafkaTopicOffset, msg.Topic), msg.Offset, 0) err: %v", err))
						return
					}
					// todo 释放redis分布式锁
					_, _ = redisLocker.Release()

					//todo task 此处需要持久化消息到db中
					var msgOp MsgOp
					_ = json.Unmarshal(msg.Value, &msgOp)
					// 只有聊天消息才会被持久化
					switch int(msgOp.Op) {
					case common.OpGroupMsgSend:
						var _msg proto.PushGroupMsgReq_Msg
						_ = json.Unmarshal(msg.Value, &_msg)
						dbMsg := &db.TMessage{
							Belong:      _msg.GroupId,
							SnowID:      _msg.SnowId,
							Type:        "group",
							Content:     _msg.Content,
							FromA:       _msg.Userid,
							ToB:         _msg.GroupId,
							MessageType: _msg.MessageType,
						}
						body, _ := json.Marshal(&dbMsg)
						rLock, _ := common.NewRedisLocker(fmt.Sprintf(common.PersistentLock, fmt.Sprintf(common.GroupLetterBox, _msg.GroupId)), "lock")
						rLock.SetExpire(60)
						lock, _ := rLock.Acquire()
						for !lock {
							lock, _ = rLock.Acquire()
							time.Sleep(time.Millisecond * 280)
						}
						err = common.RedisHSet(fmt.Sprintf(common.GroupLetterBox, _msg.GroupId), _msg.SnowId, body)
						_, _ = rLock.Release()
						if err != nil {
							zlog.Error(fmt.Sprintf("Failed to temporarily store data 「err=%v」", err))
						}
					case common.OpFriendMsgSend:
						var _msg proto.PushFriendMsgReq_Msg
						_ = json.Unmarshal(msg.Value, &_msg)
						dbMsg := &db.TMessage{
							Belong:      ch.Userid,
							SnowID:      _msg.SnowId,
							Type:        "friend",
							Content:     _msg.Content,
							FromA:       _msg.Userid,
							ToB:         _msg.FriendId,
							MessageType: _msg.MessageType,
						}
						body, _ := json.Marshal(&dbMsg)
						rLock, _ := common.NewRedisLocker(fmt.Sprintf(common.PersistentLock, fmt.Sprintf(common.UserLetterBox, ch.Userid)), "lock")
						rLock.SetExpire(60)
						lock, _ := rLock.Acquire()
						for !lock {
							lock, _ = rLock.Acquire()
							time.Sleep(time.Millisecond * 290)
						}
						err = common.RedisHSet(fmt.Sprintf(common.GroupLetterBox, ch.Userid), _msg.SnowId, body)
						_, _ = rLock.Release()
						if err != nil {
							zlog.Error(fmt.Sprintf("Failed to temporarily store data 「err=%v」", err))
						}
					}
				}
			}
		case <-commitTicker.C:
			// 定时持久化数据到db
			saveChatDataToDb(ch)
		case <-ticker.C:
			// 利用心跳包定期检测客户端是否存活
			err := ch.Conn.SetWriteDeadline(time.Now().Add(ws.Options.WriteWait))
			if err != nil {
				zlog.Warn(fmt.Sprintf("ch.Conn.SetWriteDeadline err :%v", err))
			}
			zlog.Debug(fmt.Sprintf("websocket.PingMessage :%v", websocket.PingMessage))
			if err := ch.Conn.WriteMessage(websocket.PingMessage, nil); err != nil {
				// 确保已经推送的消息被正确提交偏移量后才退出
				ticker.Stop()
				return
			}
		}
	}
}

type wsConnReq struct {
	AccessToken string `json:"accessToken"`
}

func (ws *WsServer) readPump(ch *Channel, c *Connect) {
	defer func() {
		zlog.Info("start exec disConnect ...")
		// TODO：如果当前连接并没有通过身份验证，则直接关闭没有通过鉴权的连接
		if ch.Userid == 0 {
			zlog.Info("the websocket conn not a valid user")
			err := ch.Conn.Close()
			if err != nil {
				zlog.Warn(fmt.Sprintf(" ch.conn.Close err :%s  ", err.Error()))
			}
			return
		}
		zlog.Info("exec disConnect ...")
		// 从桶中删除对应用户或者房间的数据，以此表示对应用户、房间已经下线
		ws.Bucket(ch.Userid).DeleteChanel(ch)
		// 调用logic层rpc服务来下线对应房间和用户
		if err := ws.Operator.DisConnect(ch.Userid); err != nil {
			zlog.Warn(fmt.Sprintf("DisConnect err :%s", err.Error()))
		}
		err := ch.Conn.Close()
		if err != nil {
			zlog.Warn(fmt.Sprintf("  ch.Conn.Close err :%s  ", err.Error()))
		}
	}()

	ch.Conn.SetReadLimit(int64(ws.Options.MaxMessageSize))
	err := ch.Conn.SetReadDeadline(time.Now().Add(ws.Options.PongWait))
	if err != nil {
		zlog.Warn(fmt.Sprintf("ch.Conn.SetReadDeadline err%v", err))
	}
	ch.Conn.SetPongHandler(func(string) error {
		err = ch.Conn.SetReadDeadline(time.Now().Add(ws.Options.PongWait))
		if err != nil {
			zlog.Warn(fmt.Sprintf("ch.Conn.SetReadDeadline err %v", err))
		}
		return nil
	})

	for {
		msgType, message, err := ch.Conn.ReadMessage()
		if err != nil {
			// 判断连接关闭类型是否websocket.CloseGoingAway、websocket.CloseAbnormalClosure（异常关闭）其中之一
			// 如果不是则返回true，如果是则返回false
			if websocket.IsUnexpectedCloseError(err, websocket.CloseGoingAway, websocket.CloseAbnormalClosure) {
				zlog.Error(fmt.Sprintf("readPump ReadMessage err:%v", err))
				return
			}
		}
		if message == nil {
			return
		}
		var connReq wsConnReq
		zlog.Info(fmt.Sprintf("get a message :%s", message))
		// 发连接请求
		if err := json.Unmarshal(message, &connReq); err != nil {
			zlog.Error(fmt.Sprintf("message struct %+v", connReq))
		}
		if connReq.AccessToken == "" {
			zlog.Error(fmt.Sprintf("s.operator.Connect no authToken"))
			return
		}
		serverId := c.ServerId //config.Conf.Connect.ConnectWebsocket.ServerId
		// 调用logic层连接的rpc服务
		userId, err := ws.Operator.Connect(connReq.AccessToken, serverId)
		if err != nil {
			zlog.Error(fmt.Sprintf("s.operator.Connect error %s", err.Error()))
			return
		}
		if userId == 0 {
			zlog.Error(fmt.Sprintf("Invalid AuthToken ,userId empty"))
			return
		}

		// 连接后成功身份验证的响应
		err = ch.Conn.WriteMessage(msgType, []byte("ok"))
		if err != nil {
			zlog.Error(err.Error())
			return
		}

		zlog.Info(fmt.Sprintf("websocket rpc call return userId:%d", userId))
		// 获取桶
		b := ws.Bucket(userId)
		//insert into a bucket,TODO：由于当用户退出群聊和加入群聊时，bucket对应的节点都要发生变化
		//TODO:所以应该在客户端保证，加入和退出群聊后客户端要先断开连接后重连
		var groupIdList []int64
		res, err := common.RedisGetString(fmt.Sprintf(common.UserGroupList, userId))
		if err != nil {
			zlog.Error(fmt.Sprintf("common.RedisGetString("+fmt.Sprintf(common.UserGroupList, userId)+") err :%v", err))
			return
		}
		_ = json.Unmarshal(res, &groupIdList)
		for _, val := range groupIdList {
			b.PutChannel(userId, val, ch)
		}
	}
}

// 持久化数据到db
func saveChatDataToDb(ch *Channel) {
	// todo：需要使用分布式锁防止其他新的消息在此刻写入redis
	rLock, err := common.NewRedisLocker(fmt.Sprintf(common.PersistentLock, fmt.Sprintf(common.UserLetterBox, ch.Userid)), "lock")
	if err != nil {
		zlog.Error(err.Error())
		return
	}
	rLock.SetExpire(60)
	lock, _ := rLock.Acquire()
	//分布式锁获取成功的情况才继续
	for !lock {
		lock, _ = rLock.Acquire()
		time.Sleep(time.Millisecond * 300)
	}
	allFriendMsg, err := common.RedisHGetAll(fmt.Sprintf(common.UserLetterBox, ch.Userid))
	if err != nil {
		zlog.Error(err.Error())
		return
	}
	var friendMsg []db.TMessage
	for snowId, payload := range allFriendMsg {
		zlog.Debug(fmt.Sprintf("snowId==%s", snowId))
		var tmp db.TMessage
		_ = json.Unmarshal([]byte(payload), &tmp)
		friendMsg = append(friendMsg, tmp)
	}
	db.SaveMsgInBatches(friendMsg)
	// 删除已经持久化的数据
	err = common.RedisDelString(fmt.Sprintf(common.UserLetterBox, ch.Userid))
	if err != nil {
		zlog.Error(err.Error())
	}
	_, _ = rLock.Release()

	for _, val := range ch.GroupNodes {
		var groupMsg []db.TMessage
		rLock, _ = common.NewRedisLocker(fmt.Sprintf(common.PersistentLock, fmt.Sprintf(common.GroupLetterBox, val.groupId)), "lock")
		rLock.SetExpire(60)
		for !lock {
			lock, _ = rLock.Acquire()
			time.Sleep(time.Millisecond * 300)
		}
		allGroupMsg, err := common.RedisHGetAll(fmt.Sprintf(common.GroupLetterBox, val.groupId))
		if err != nil {
			zlog.Error(err.Error())
			break
		}
		for snowId, payload := range allGroupMsg {
			zlog.Debug(fmt.Sprintf("snowId==%s", snowId))
			var tmp db.TMessage
			_ = json.Unmarshal([]byte(payload), &tmp)
			groupMsg = append(groupMsg, tmp)
		}
		db.SaveMsgInBatches(groupMsg)
		err = common.RedisDelString(fmt.Sprintf(common.GroupLetterBox, val.groupId))
		if err != nil {
			zlog.Error(err.Error())
		}
		_, _ = rLock.Release()
	}
}
