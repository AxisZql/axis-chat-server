package common

/**
*Author: AxisZql
*Date: 2022-5-31
*DESC: MQ producer and consumer
 */

import (
	"axisChat/utils/zlog"
	"context"
	"fmt"
	"github.com/pkg/errors"
	"github.com/segmentio/kafka-go"
	"sync"
	"time"
)

type KafkaProducerConn struct {
	mutex      sync.RWMutex
	SocketMap  map[string]*kafka.Conn
	CreateTime map[string]time.Time // 记录连接建立的时间
}

var (
	producerConnMap KafkaProducerConn
	_once           sync.Once
)

const (
	UserQueuePrefix  = "user_chat_%d"
	GroupQueuePrefix = "group_chat_%d"
)

func watchLongTimeNotUseConn() {
	ticker := time.NewTicker(5 * time.Second)
	defer func() {
		ticker.Stop()
	}()
	select {
	case <-ticker.C:
		producerConnMap.mutex.RLock()
		for key, val := range producerConnMap.CreateTime {
			// 超过5min没有使用的连接将会被断开
			if time.Since(val) > 5*time.Minute {
				producerConnMap.mutex.Lock()
				err := producerConnMap.SocketMap[key].Close()
				if err != nil {
					zlog.Error(err.Error())
				}
				delete(producerConnMap.SocketMap, key)
				delete(producerConnMap.CreateTime, key)
				producerConnMap.mutex.Unlock()
			}
		}
		producerConnMap.mutex.RUnlock()
	}
}

func getProducerConn(objectId int64, _type string) (*kafka.Conn, error) {
	_once.Do(func() {
		// 监听并删除长时间不使用的连接
		go watchLongTimeNotUseConn()
	})
	var topic = ""
	if _type == "user" {
		topic = fmt.Sprintf(UserQueuePrefix, objectId)
	} else if _type == "group" {
		topic = fmt.Sprintf(GroupQueuePrefix, objectId)
	} else {
		return nil, errors.New(fmt.Sprintf("_type = %s is not alllow", _type))
	}
	producerConnMap.mutex.RLock()
	if conn, ok := producerConnMap.SocketMap[topic]; ok {
		producerConnMap.CreateTime[topic] = time.Now()
		producerConnMap.mutex.RUnlock()
		return conn, nil
	}
	producerConnMap.mutex.RUnlock()
	// 为保证整体消息的有序性，每个topic的partition数为1
	conn, err := kafka.DialLeader(context.Background(), "tcp", "localhost:9092", topic, 1)
	if err != nil {
		err = errors.Wrap(err, "get topic:%s from Kafka failure")
		return nil, err
	}
	defer func() {
		err := conn.Close()
		if err != nil {
			zlog.Error(err.Error())
		}
	}()
	err = conn.SetWriteDeadline(time.Now().Add(5 * time.Second))
	if err != nil {
		err = errors.Wrap(err, "conn.SetWriteDeadlin get err")
		return nil, err
	}
	producerConnMap.mutex.Lock()
	defer producerConnMap.mutex.Unlock()
	producerConnMap.SocketMap[topic] = conn
	producerConnMap.CreateTime[topic] = time.Now()
	return conn, nil
}

func TopicProduce(objectId int64, _type string, msg []byte) error {
	conn, err := getProducerConn(objectId, _type)
	if err != nil {
		zlog.Error(err.Error())
		return err
	}
	_, err = conn.WriteMessages(kafka.Message{
		Value: msg,
	})
	if err != nil {
		err = errors.Wrap(err, "conn.WriteMessages get err")
		zlog.Error(err.Error())
		return err
	}
	return nil
}

func GetConsumeReader(objectId int64, topic string) (*kafka.Reader, error) {
	reader := kafka.NewReader(kafka.ReaderConfig{
		Brokers:  []string{"localhost:9092"}, //topic所在的broker
		Topic:    topic,
		GroupID:  fmt.Sprintf("%d", objectId), // 消费者在topic中的groupId，通过groupId可以获取当前消费的偏移量,本项目中groupID设置为每个对象的id
		MinBytes: 10e3,                        //10KB
		MaxBytes: 10e6,                        //10MB
	})
	return reader, nil
}

func TopicConsume(reader *kafka.Reader) (msg kafka.Message, err error) {
	msg, err = reader.FetchMessage(context.Background())
	if err != nil {
		zlog.Error(err.Error())
		return
	}
	return
}

func TopicConsumerConfirm(reader *kafka.Reader, msg kafka.Message) error {
	if err := reader.CommitMessages(context.Background(), msg); err != nil {
		zlog.Error(fmt.Sprintf("failed to commit messages:%v", err))
		return err
	}
	return nil
}
