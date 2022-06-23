package common

import (
	"axisChat/utils/zlog"
	"fmt"
	"github.com/go-redis/redis"
	"strconv"
	"sync/atomic"
)

/*
*Author:AxisZql
*Date:2022-6-19 9:46 AM
*Desc:基于redis的分布式锁实现
 */

const (
	letters = "abcdefghijklmnopqrstuvwxyzABCDEFGHIJKLMNOPQRSTUVWXYZ"
	// 为确保操作的原子性，采用了lua脚本,只有当锁不存在时才可以设置
	lockCommand = `
    return redis.call("SET",KEYS[1],ARGV[1],"NX","PX",ARGV[2])
    `
	// 释放锁
	delCommand = `
    return redis.call("DEL",KEYS[1])`

	// 默认超时时间
	tolerance       = 500  //milliseconds 毫秒
	millisPerSecond = 1000 // 1000ms=1s
)

type RedisLockObj struct {
	client *redis.Client
	timout uint32
	key    string
	value  string
}

func NewRedisLocker(key string, value string) (*RedisLockObj, error) {
	client, err := GetRedisClientByKey(key)
	if err != nil {
		return nil, err
	}
	return &RedisLockObj{
		client: client,
		key:    key,
		value:  value,
	}, nil
}

// SetExpire 设置分布式锁的过期时间
func (rl *RedisLockObj) SetExpire(timout int) {
	atomic.StoreUint32(&rl.timout, uint32(timout))
}

// Acquire 获取分布式锁
func (rl *RedisLockObj) Acquire() (bool, error) {
	// 获取过期时间
	timeout := atomic.LoadUint32(&rl.timout)
	// 默认锁的过期时间是500毫秒
	resp, err := rl.client.Eval(lockCommand, []string{rl.key}, []string{
		rl.value,
		strconv.Itoa(int(timeout)*millisPerSecond + tolerance), //保底锁的有效时间是500ms
	}).Result()
	if err == redis.Nil {
		return false, nil
	} else if err != nil {
		zlog.Error(fmt.Sprintf("Error on acquiring lock for %s,%v", rl.key, err))
		return false, err
	}
	reply, ok := resp.(string)
	if ok && reply == "OK" {
		return true, nil
	}
	zlog.Error(fmt.Sprintf("Unknown reply when acquiring lock for %s:%v", rl.key, resp))
	return false, nil
}

// Release 释放锁
func (rl *RedisLockObj) Release() (bool, error) {
	resp, err := rl.client.Eval(delCommand, []string{rl.key}).Result()
	if err != nil {
		return false, err
	}
	reply, ok := resp.(int64)
	if !ok {
		return false, nil
	}
	return reply == 1, nil
}
