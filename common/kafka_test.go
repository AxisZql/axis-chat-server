package common

import (
	"fmt"
	"github.com/segmentio/kafka-go"
	"testing"
)

func GetAllTopic() {
	// 获取当前broke下所有topic
	conn, err := kafka.Dial("tcp", "localhost:9092")
	if err != nil {
		panic(err.Error())
	}
	defer conn.Close()

	partitions, err := conn.ReadPartitions()
	if err != nil {
		panic(err.Error())
	}

	m := map[string]struct{}{}

	for _, p := range partitions {
		m[p.Topic] = struct{}{}
	}
	for k := range m {
		fmt.Println(k)
	}
}

func TestTopicProduce(t *testing.T) {
	GetAllTopic()
	//for i := 0; i < 10; i++ {
	//	_ = TopicProduce("userA", []byte("你好👌2"+fmt.Sprintf("%d", i)))
	//}
}

func TestTopicConsume(t *testing.T) {
	GetAllTopic()
	//TopicConsume("userA-g2", "userA")
	//TopicConsume("userA-g1", "userA")
}
