package common

import (
	"axisChat/config"
	"axisChat/utils"
	"axisChat/utils/zlog"
	"encoding/json"
	"fmt"
	"github.com/go-redis/redis"
	"github.com/pkg/errors"
	"go.uber.org/zap"
	"strings"
	"sync"
	"time"
)

/**
*Author: AxisZql
*Date: 2022-5-31
*DESC: redis相关配置
 */

// SESSION map token to userinfo
const SESSION string = "axis:session:%s"

// UseridMapToken map the access_token to userid
const UseridMapToken string = "axis:user_map_token:%d"

// GroupOnlineUser map groupId to record the group all online user`s serverId
const GroupOnlineUser string = "axis:group_online_user:%d"

// GroupOnlineUserCount map groupId to the group online user count todo:改变存储方式，利用hash table存放（因为一致性哈希的造成的key分散问题
const GroupOnlineUserCount string = "axis:group_online_user_count"

// UseridMapServerId TODO:记录对应id的用户在connect上哪个独立服务上，方便后期投递消息时可以找到对应用户的连接实例，从而实现消息的投递
const UseridMapServerId string = "axis:userid_map_serverid:%d"

// UserGroupList 用户加入的群聊id列表
const UserGroupList string = "axis:user_group_list:%d"

// UserFriendList  用户所有好友id
const UserFriendList string = "axis:user_friend_list:%d"

// AllOnlineUser 记录所有在线用户
const AllOnlineUser string = "axis:online_user"

// KafkaTopicOffset 记录kafka最后一次提交的偏移量,topic名和最新提交偏移量的映射
const KafkaTopicOffset string = "axis:kafka_topic_offset:%s"

// StatusMsgQueue  利用List实现简单的消息队列，存放对象上下线状态消息
const StatusMsgQueue string = "axis:status_msg_queue"

// RedisLock redis distributed lock , 后缀是topic名称，每个topic持有一个分布式锁
const RedisLock string = "axis:redis_lock:%s"

// UserLetterBox GroupLetterBox 是暂时存放已经消费的数据但是为被持久化的数据，然后定时批量持久化
const UserLetterBox string = "axis_user_letter_box:%d"

const GroupLetterBox string = "axis_group_letter_box:%d"

// PersistentLock 数据批量持久化时要加上分布式锁
const PersistentLock string = "axis_persistent_lock:%s"

type RedisClient struct {
	Client map[string]*redis.Client
}

var (
	once           sync.Once
	redisClientMap = &RedisClient{
		Client: make(map[string]*redis.Client),
	}
	consistentHash *utils.HashBalance
)

func InitRedis() {
	once.Do(func() {
		// 虚拟节点设置为16个
		consistentHash = utils.NewHashBalance(16, nil)
		Conf := config.GetConfig()
		hostList := strings.Split(Conf.Common.Redis.Address, ";")
		for _, val := range hostList {
			host, port, err := utils.ParseAddress(val)
			if err != nil {
				panic(err)
			}
			redisCli := redis.NewClient(&redis.Options{
				Addr:     fmt.Sprintf("%s:%d", host, port),
				Password: Conf.Common.Redis.Password,
				DB:       Conf.Common.Redis.Db,
			})
			zlog.Info("Ping Redis", zap.String("address", fmt.Sprintf("%s:%d", host, port)))
			_, err = redisCli.Ping().Result()
			if err != nil {
				zlog.Error(fmt.Sprintf("connect redis:%s:%d failed	err:%v", host, port, err))
				continue
			}
			zlog.Info(fmt.Sprintf("connect to redis %s:%d", host, port))
			redisClientMap.Client[val] = redisCli
			// TODO:将成功连接的节点加入一致性哈希
			_ = consistentHash.Add(val)
		}
	})

}

// GetRedisClientByKey 根据请求方的key获取从一致性哈希表中获取合适的redis服务器
func GetRedisClientByKey(key string) (*redis.Client, error) {
	address, err := consistentHash.Get(key)
	if err != nil {
		return nil, errors.Wrap(err, "get redis address from consistent hash table failed")
	}
	if address == "" {
		return nil, errors.New(fmt.Sprintf("address got :%v", address))
	}
	redisCli, ok := redisClientMap.Client[address]
	if !ok {
		return nil, errors.New("abnormal can't get redis client")
	}
	return redisCli, nil
}

// CacheOptions 对redis的string类型的操作进行封装
type CacheOptions struct {
	Key      string
	Duration time.Duration
	Fun      func() (interface{}, error)
	Receiver interface{}
}

// GetSet 封装获取key的流程，存在则获取，不存在则设置缓存
func (c *CacheOptions) GetSet() (interface{}, error) {
	// GetSet 利用接口抽象获取缓存的流程
	err := getSetCache(c)
	if err != nil {
		return nil, err
	}
	return c.Receiver, nil
}

// getSetCache 获取缓存，不存在则调用Fun函数获取对应数据加入缓存,适用k-v单一映射
func getSetCache(c *CacheOptions) (err error) {
	if c == nil || c.Receiver == nil || c.Key == "" {
		err = fmt.Errorf("illegal arguments")
		zlog.Error(err.Error())
		return
	}
	//从一致性哈希表中获取可用主机
	redisCli, err := GetRedisClientByKey(c.Key)
	if err != nil {
		return err
	}
	//查询缓存
	val, err := redisCli.Get(c.Key).Result()
	if err != nil && err != redis.Nil {
		zlog.Error(err.Error())
		return
	}
	if err == redis.Nil {
		//调用对应函数设置并获取缓存
		c.Receiver, err = c.Fun()
		if err != nil {
			return
		}
		if c.Receiver == nil {
			return nil
		}
		zlog.Debug(fmt.Sprintf("Set cache %s", c.Key))
		var buf []byte
		if data, ok := c.Receiver.([]byte); ok {
			buf = data
		} else {
			buf, err = json.Marshal(&c.Receiver)
			if err != nil {
				zlog.Error(err.Error())
				return
			}
		}
		err = redisCli.Set(c.Key, buf, c.Duration).Err()
		if err != nil {
			zlog.Error(err.Error())
			return
		}
	} else {
		//如果存在则解析缓存
		zlog.Debug(fmt.Sprintf("Hit cache %s", c.Key))
		if _, ok := c.Receiver.([]byte); ok {
			c.Receiver = []byte(val)
			return
		}
		err = json.Unmarshal([]byte(val), &c.Receiver)
		if err != nil {
			zlog.Error(fmt.Sprintf("解析缓存失败 key:%s value:%v", c.Key, val))
			return
		}
	}
	return
}

func RedisSetString(key string, value []byte, expire time.Duration) error {
	client, err := GetRedisClientByKey(key)
	if err != nil {
		zlog.Error(err.Error())
		return err
	}
	err = client.Set(key, value, expire).Err()
	if err != nil {
		zlog.Error(err.Error())
		return err
	}
	return nil
}

func RedisGetString(key string) ([]byte, error) {
	client, err := GetRedisClientByKey(key)
	if err != nil {
		zlog.Error(err.Error())
		return nil, err
	}
	res, err := client.Get(key).Bytes()
	if err != nil && err != redis.Nil {
		zlog.Error(err.Error())
		return nil, err
	}
	return res, nil
}

func RedisDelString(key string) error {
	client, err := GetRedisClientByKey(key)
	if err != nil {
		zlog.Error(err.Error())
		return err
	}
	err = client.Del(key).Err()
	if err != nil && err != redis.Nil {
		zlog.Error(err.Error())
		return err
	}
	return nil
}

// RedisIsNotExistHSet  只有当hash table没有对应filed才会进行设置
func RedisIsNotExistHSet(key string, filed string, value interface{}) (has bool, err error) {
	// has 为true时表示之前存在，false表示之前不存在
	client, err := GetRedisClientByKey(key)
	if err != nil {
		zlog.Error(err.Error())
		return false, err
	}
	if client.HGet(key, fmt.Sprintf(filed)).Val() == "" {
		err = client.HSet(key, filed, value).Err()
		if err != nil {
			zlog.Error(err.Error())
			return false, err
		}
		return false, nil
	}
	err = client.HSet(key, filed, value).Err()
	if err != nil {
		zlog.Error(err.Error())
		return false, err
	}
	return true, nil
}

func RedisHSet(key string, filed string, value interface{}) error {
	client, err := GetRedisClientByKey(key)
	if err != nil {
		zlog.Error(err.Error())
		return err
	}
	if err := client.HSet(key, filed, value).Err(); err != nil {
		zlog.Error(err.Error())
		return err
	}
	return nil
}

// RedisHINCRBY hash table filed 增量
func RedisHINCRBY(key string, filed string, increment int64) error {
	client, err := GetRedisClientByKey(key)
	if err != nil {
		zlog.Error(err.Error())
		return err
	}
	err = client.HIncrBy(key, filed, increment).Err()
	if err != nil {
		zlog.Error(err.Error())
		return err
	}
	return nil
}

func RedisHGetAll(key string) (map[string]string, error) {
	client, err := GetRedisClientByKey(key)
	if err != nil {
		zlog.Error(err.Error())
		return nil, err
	}
	res, err := client.HGetAll(key).Result()
	if err != nil {
		zlog.Error(err.Error())
		return nil, err
	}
	return res, nil
}

func RedisHGet(key string, filed string) (string, error) {
	client, err := GetRedisClientByKey(key)
	if err != nil {
		zlog.Error(err.Error())
		return "", err
	}
	res, err := client.HGet(key, filed).Result()
	if err != nil {
		zlog.Error(err.Error())
		return "", err
	}
	return res, nil
}

func RedisHDel(key string, filed string) (has bool, err error) {
	client, err := GetRedisClientByKey(key)
	if err != nil {
		zlog.Error(err.Error())
		return false, err
	}
	if client.HGet(key, fmt.Sprintf(filed)).Val() != "" {
		err = client.HDel(key, filed).Err()
		if err != nil {
			zlog.Error(err.Error())
			return false, err
		}
		return true, nil
	}
	return false, nil
}

func RedisInc(key string) error {
	client, err := GetRedisClientByKey(key)
	if err != nil {
		zlog.Error(err.Error())
		return err
	}
	err = client.Incr(key).Err()
	if err != nil {
		zlog.Error(err.Error())
		return err
	}
	return nil
}

func RedisDecr(key string) error {
	client, err := GetRedisClientByKey(key)
	if err != nil {
		zlog.Error(err.Error())
		return err
	}
	err = client.Decr(key).Err()
	if err != nil {
		zlog.Error(err.Error())
		return err
	}
	return nil
}

func RedisLPUSH(key string, value []byte) error {
	client, err := GetRedisClientByKey(key)
	if err != nil {
		zlog.Error(err.Error())
		return err
	}
	err = client.LPush(key, value).Err()
	if err != nil {
		zlog.Error(err.Error())
		return err
	}
	return nil
}

func RedisBRPOP(key string, timeOut time.Duration) ([]string, error) {
	client, err := GetRedisClientByKey(key)
	if err != nil {
		zlog.Error(err.Error())
		return []string{}, err
	}
	var result []string
	result, err = client.BRPop(timeOut, key).Result()
	if err != nil && err != redis.Nil {
		zlog.Error(err.Error())
		return []string{}, err
	}
	return result, nil
}
