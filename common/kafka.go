package common

/**
*Author: AxisZql
*Date: 2022-5-31
*DESC: MQ producer and consumer
 */

import (
	"context"
	"fmt"
	"github.com/pkg/errors"
	"github.com/segmentio/kafka-go"
	"log"
	"time"
)

func TopicProduce(topic string, msg []byte) error {
	// 为保证整体消息的有序性，每个topic的partition数为1
	conn, err := kafka.DialLeader(context.Background(), "tcp", "localhost:9092", topic, 0)
	if err != nil {
		err = errors.Wrap(err, "get topic:%s from Kafka failure")
		return err
	}
	defer func() {
		err := conn.Close()
		if err != nil {
			// TODO: print the error to log
			fmt.Println(err)
		}
	}()
	err = conn.SetWriteDeadline(time.Now().Add(5 * time.Second))
	if err != nil {
		err = errors.Wrap(err, "conn.SetWriteDeadlin get err")
		return err
	}
	_, err = conn.WriteMessages(kafka.Message{
		Value: msg,
	})
	if err != nil {
		err = errors.Wrap(err, "conn.WriteMessages get err")
		return err
	}
	return nil
}

func TopicConsume(cGroupId string, topic string) error {
	reader := kafka.NewReader(kafka.ReaderConfig{
		Brokers:  []string{"localhost:9092"}, //topic所在的broker
		Topic:    topic,
		GroupID:  cGroupId, // 消费者在topic中的groupId，通过groupId可以获取当前消费的偏移量
		MinBytes: 10e3,     //10KB
		MaxBytes: 10e6,     //10MB
	})
	defer func() {
		err := reader.Close()
		// TODO: print the error to log
		fmt.Println(err)
	}()

	for {
		msg, err := reader.FetchMessage(context.Background())
		if err != nil {
			err = errors.Wrap(err, fmt.Sprintf("read the msg from topic:%s get error", topic))
			return err
		}
		fmt.Println(msg.Offset, string(msg.Key), string(msg.Value))
		// TODO：消费确认，提交偏移量，只有提交量偏移量才能证明消息被正常消费
		if err := reader.CommitMessages(context.Background(), msg); err != nil {
			log.Fatal("failed to commit messages:", err)
		}
	}

}
