package connect

import (
	"github.com/gorilla/websocket"
	"github.com/segmentio/kafka-go"
	"net"
)

/**
*Author: AxisZql
*Date: 2022-6-3
*DESC: 客户端连接结构
 */

type Channel struct {
	Broadcast  chan kafka.Message //要推送给你当前连接对应用户的消息数据
	Userid     int64
	Conn       *websocket.Conn
	CnnTcp     *net.TCPConn
	Next       *Channel // 当bucket管理当是群聊节点时，对应当群聊节点维护该群聊所有在在线当用户端连接
	Prev       *Channel
	GroupNodes []*GroupNode // 这里需要记录对应的GroupNode，为了后期用户下线的时候，能快速从对应的GroupNode的双向链表中删除对应的Channel，因为一位用户可能有多个群聊
}

func NewChannel(size int) (s *Channel) {
	return &Channel{
		Broadcast: make(chan kafka.Message, size), //最多可以往信道中写入的消息条数为size
	}
}

// Push Channel
//进行消息推送准备
func (ch *Channel) Push(msg kafka.Message) {
	select {
	case ch.Broadcast <- msg:
	default:
	}
}
