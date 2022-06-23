package common

/**
*Author: AxisZql
*Date: 2022-5-31
*DESC: MQ producer and consumer
 */

import (
	"axisChat/config"
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

type KafkaConsumerReader struct {
	mutex      sync.RWMutex
	ReaderMap  map[string]*kafka.Reader
	CreateTime map[string]time.Time
}

var (
	producerConnMap = &KafkaProducerConn{
		SocketMap:  make(map[string]*kafka.Conn),
		CreateTime: make(map[string]time.Time),
	}

	consumerReader = &KafkaConsumerReader{
		ReaderMap:  make(map[string]*kafka.Reader),
		CreateTime: make(map[string]time.Time),
	}

	_once sync.Once
)

const (
	FriendQueuePrefix = "friend_chat_%d"
	GroupQueuePrefix  = "group_chat_%d"
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
			// 超过1.5min没有使用的连接将会被断开
			if time.Since(val) > 30*time.Second {
				producerConnMap.mutex.RUnlock()
				producerConnMap.mutex.Lock()
				err := producerConnMap.SocketMap[key].Close()
				if err != nil {
					zlog.Error(err.Error())
				}
				delete(producerConnMap.SocketMap, key)
				delete(producerConnMap.CreateTime, key)
				producerConnMap.mutex.Unlock()
				producerConnMap.mutex.RLock()
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
	if _type == "friend" {
		topic = fmt.Sprintf(FriendQueuePrefix, objectId)
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

	// todo 为了确保kafka客户端能够被正常初始化，且不产生死锁，故将加写锁操作提前
	producerConnMap.mutex.Lock()
	defer producerConnMap.mutex.Unlock()
	// 为保证整体消息的有序性，每个topic的partition数为0
	conn, err := kafka.DialLeader(context.Background(), "tcp", "localhost:9092", topic, 0)
	if err != nil {
		err = errors.Wrap(err, "get topic:%s from Kafka failure")
		return nil, err
	}

	zlog.Info("success init kafka topic producer client!!!")
	producerConnMap.SocketMap[topic] = conn
	producerConnMap.CreateTime[topic] = time.Now()
	return conn, nil
}

func TopicProduce(objectId int64, _type string, msg []byte) error {
	var topic string
	if _type == "friend" {
		topic = fmt.Sprintf(FriendQueuePrefix, objectId)
	} else if _type == "group" {
		topic = fmt.Sprintf(GroupQueuePrefix, objectId)
	} else {
		return errors.New(fmt.Sprintf("_type = %s is not alllow", _type))
	}
	conn, err := getProducerConn(objectId, _type)
	if err != nil {
		zlog.Error(err.Error())
		return err
	}
	// 设置写入消息的超时时间
	err = conn.SetWriteDeadline(time.Now().Add(5 * time.Second))
	if err != nil {
		err = errors.Wrap(err, "conn.SetWriteDeadlin get err")

		// 删除不可用的kafka实例
		producerConnMap.mutex.Lock()
		delete(producerConnMap.SocketMap, topic)
		delete(producerConnMap.CreateTime, topic)
		producerConnMap.mutex.Unlock()
		return err
	}
	_, err = conn.WriteMessages(kafka.Message{
		Value: msg,
	})
	if err != nil {
		err = errors.Wrap(err, "conn.WriteMessages get err")
		zlog.Error(err.Error())

		//删除不可用的Kafka实例
		producerConnMap.mutex.Lock()
		delete(producerConnMap.SocketMap, topic)
		delete(producerConnMap.CreateTime, topic)
		producerConnMap.mutex.Unlock()
		return err
	}
	zlog.Debug(fmt.Sprintf("success write msg=%s", string(msg)))
	return nil
}

// ===================消费者===============

func watchLongTimeNotUseReader() {
	ticker := time.NewTicker(5 * time.Second)
	defer func() {
		ticker.Stop()
	}()
	select {
	case <-ticker.C:
		consumerReader.mutex.RLock()
		for key, val := range consumerReader.CreateTime {
			// 超过5min没有使用的连接将会被断开
			if time.Since(val) > 5*time.Minute {
				consumerReader.mutex.RUnlock()
				consumerReader.mutex.Lock()
				err := consumerReader.ReaderMap[key].Close()
				if err != nil {
					zlog.Error(err.Error())
				}
				delete(consumerReader.ReaderMap, key)
				delete(consumerReader.ReaderMap, key)
				consumerReader.mutex.Unlock()
				consumerReader.mutex.RLock()
			}
		}
		consumerReader.mutex.RUnlock()
	}
}

func getConsumerReader(groupId string, topic string) (*kafka.Reader, error) {
	_once.Do(func() {
		// 监听并删除长时间不使用的Reader连接
		go watchLongTimeNotUseReader()
	})

	consumerReader.mutex.RLock()
	if reader, ok := consumerReader.ReaderMap[groupId]; ok {
		consumerReader.CreateTime[groupId] = time.Now()
		consumerReader.mutex.RUnlock()
		return reader, nil
	}
	consumerReader.mutex.RUnlock()

	// todo 为了确保kafka Reader客户端能够被正常初始化，且不产生死锁，故将加写锁操作提前
	consumerReader.mutex.Lock()
	defer consumerReader.mutex.Unlock()
	// 为保证整体消息的有序性，每个topic的partition数为1
	reader := kafka.NewReader(kafka.ReaderConfig{
		Brokers:  []string{config.GetConfig().Common.Kafka.Address},
		Topic:    topic,
		GroupID:  groupId,
		MinBytes: 10e3,
		MaxBytes: 10e6,
	})

	zlog.Info("success init kafka topic consumer reader!!!")
	consumerReader.ReaderMap[topic] = reader
	consumerReader.CreateTime[topic] = time.Now()
	return reader, nil
}

func GetConsumeReader(consumerSuffix string, topic string) (*kafka.Reader, error) {
	groupId := fmt.Sprintf("%s-%s", topic, consumerSuffix)
	reader, err := getConsumerReader(groupId, topic)

	// todo bug: 这里有可能出现reader close的异常
	if err != nil {
		zlog.Error(fmt.Sprintf("get consumer reader failure「err:%v」", err))
		return nil, err
	}
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
