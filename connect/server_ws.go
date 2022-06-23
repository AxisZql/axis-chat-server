package connect

import (
	"axisChat/common"
	"axisChat/db"
	"axisChat/proto"
	"axisChat/utils/zlog"
	"encoding/json"
	"fmt"
	"github.com/gorilla/websocket"
	"github.com/segmentio/kafka-go"
	"strconv"
	"strings"
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
	commitTicker := time.NewTicker(10 * time.Second)

	defer func() {
		ticker.Stop()
		commitTicker.Stop()
		err := ch.Conn.Close()
		if err != nil {
			zlog.Error(err.Error())
		}
	}()
	if err != nil {
		zlog.Error(fmt.Sprintf("get kafka consum reader err:%v", err))
		return
	}

	var preMsg *kafka.Message

	for {
		select {
		case msg, ok := <-ch.Broadcast:
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

			n, err := w.Write(msg.Value)
			fmt.Println(n)
			if err = w.Close(); err != nil {
				zlog.Error(fmt.Sprintf("w.Close err :%v", err))
				return
			}
			if err != nil {
				zlog.Error(fmt.Sprintf("push msg get err: %v", err))
			} else {
				go func() {
					// todo 如果当前消息被成功消费了offset>=msg.Offset 则不用提交确认消费的偏移量，因为消费后的消息已经持久化到db中了
					res, err := common.RedisGetString(fmt.Sprintf(common.KafkaTopicOffset, msg.Topic))
					if err != nil {
						zlog.Error(fmt.Sprintf("common.RedisGetString(fmt.Sprintf(common.KafkaTopicOffset, %s))", msg.Topic))
					}
					offset, _ := strconv.Atoi(string(res))
					if int64(offset) >= msg.Offset && offset != 0 {
						// 证明该消息已经被其他服务消费过（群聊消息会出现这种情况）
						zlog.Info(fmt.Sprintf("msg:%s have been consumed by other server 「curOffset=%d,preOffset=%d」", string(msg.Value), msg.Offset, offset))
					} else {
						preMsg = &msg
						// todo 消息消费成功后释放redis分布式锁
						redisLocker, _ := common.NewRedisLocker(fmt.Sprintf(common.RedisLock, msg.Topic), msg.Topic)

						//todo task 此处需要持久化消息到db中
						var msgOp MsgOp
						_ = json.Unmarshal(msg.Value, &msgOp)
						// 只有聊天消息才会被持久化
						switch int(msgOp.Op) {
						case common.OpGroupMsgSend:
							var _msg proto.PushGroupMsgReq_Msg
							_ = json.Unmarshal(msg.Value, &_msg)
							dbMsg := &db.TMessage{
								SnowID:      _msg.SnowId,
								Type:        "group",
								Content:     _msg.Content,
								FromA:       _msg.Userid,
								ToB:         _msg.GroupId,
								MessageType: _msg.MessageType,
							}
							db.SaveMsg(dbMsg)
						case common.OpFriendMsgSend:
							var _msg proto.PushFriendMsgReq_Msg
							_ = json.Unmarshal(msg.Value, &_msg)
							dbMsg := &db.TMessage{
								SnowID:      _msg.SnowId,
								Type:        "friend",
								Content:     _msg.Content,
								FromA:       _msg.Userid,
								ToB:         _msg.FriendId,
								MessageType: _msg.MessageType,
							}
							db.SaveMsg(dbMsg)
						}
						// todo 释放redis分布式锁
						_, _ = redisLocker.Release()
					}
				}()
			}
		case <-commitTicker.C:
			// 当前面有消息发送成功后每10s提交一次偏移量
			if preMsg == nil {
				break
			}
			go func() {
				msg := *preMsg
				// 使用正确的消费组进行消息的提交
				var consumerSuffix string
				if strings.Contains(msg.Topic, "friend") {
					consumerSuffix = fmt.Sprintf("userid-%d", ch.Userid)
				} else if strings.Contains(msg.Topic, "group") {
					var msgSend common.MsgSend
					msgSend.Msg = new(common.GroupCountMsg)
					_ = json.Unmarshal(msg.Value, &msgSend.Msg)
					consumerSuffix = fmt.Sprintf("groupid-%d", msgSend.Msg.(*common.GroupCountMsg).GroupId)
				} else {
					zlog.Error(fmt.Sprintf("invalid topic name,the project not use this topic=%s", msg.Topic))
					return
				}
				groupId := fmt.Sprintf("%s-%s", msg.Topic, consumerSuffix)
				err = common.RedisSetString(fmt.Sprintf(common.KafkaTopicOffset, msg.Topic), []byte(fmt.Sprintf("%d", msg.Offset)), 0)
				if err != nil {
					zlog.Error(fmt.Sprintf("common.RedisSetString(fmt.Sprintf(common.KafkaTopicOffset, msg.Topic), msg.Offset, 0) err: %v", err))
					return
				}

				kafkaReader, err := common.GetConsumeReader(consumerSuffix, msg.Topic)
				if err != nil {
					zlog.Error(fmt.Sprintf("get kafka reader to commit offset is failure!!!「err:%v」 groupId = %s", err, groupId))
				}

				// 第一次被成功消费
				// TODO 消息成功被消费，向kafka中当前topic对应的消费组中提交偏移量
				err = common.TopicConsumerConfirm(kafkaReader, msg)
				if err != nil {
					zlog.Error(fmt.Sprintf("common.TopicConsumerConfirm(KafkaConsumeReader, %s) err %v", msg.Topic, err))
					return
				}
			}()
		case <-ticker.C:
			// 利用心跳包定期检测客户端是否存活
			err := ch.Conn.SetWriteDeadline(time.Now().Add(ws.Options.WriteWait))
			if err != nil {
				zlog.Warn(fmt.Sprintf("ch.Conn.SetWriteDeadline err :%v", err))
			}
			zlog.Debug(fmt.Sprintf("websocket.PingMessage :%v", websocket.PingMessage))
			if err := ch.Conn.WriteMessage(websocket.PingMessage, nil); err != nil {
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
		_, message, err := ch.Conn.ReadMessage()
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
