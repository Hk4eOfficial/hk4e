package dao

import (
	"context"
	"strconv"
)

const RedisPlayerKeyPrefix = "HK4E"

const (
	AccountIdRedisKey        = "AccountId"
	AccountIdBegin    uint32 = 10000
)

func (d *Dao) GetNextAccountId() (uint32, error) {
	return d.redisInc(RedisPlayerKeyPrefix + ":" + AccountIdRedisKey)
}

func (d *Dao) GetAccountId() (uint32, error) {
	return d.redisGet(RedisPlayerKeyPrefix + ":" + AccountIdRedisKey)
}

func (d *Dao) SetAccountId(accountId uint32) error {
	return d.redisSet(RedisPlayerKeyPrefix+":"+AccountIdRedisKey, accountId)
}

func (d *Dao) redisInc(keyName string) (uint32, error) {
	var exist int64 = 0
	var err error = nil
	if d.redisCluster != nil {
		exist, err = d.redisCluster.Exists(context.TODO(), keyName).Result()
	} else {
		exist, err = d.redis.Exists(context.TODO(), keyName).Result()
	}
	if err != nil {
		return 0, err
	}
	if exist == 0 {
		var err error = nil
		if d.redisCluster != nil {
			err = d.redisCluster.Set(context.TODO(), keyName, AccountIdBegin, 0).Err()
		} else {
			err = d.redis.Set(context.TODO(), keyName, AccountIdBegin, 0).Err()
		}
		if err != nil {
			return 0, err
		}
	}
	var id int64 = 0
	if d.redisCluster != nil {
		id, err = d.redisCluster.Incr(context.TODO(), keyName).Result()
	} else {
		id, err = d.redis.Incr(context.TODO(), keyName).Result()
	}
	if err != nil {
		return 0, err
	}
	return uint32(id), nil
}

func (d *Dao) redisGet(keyName string) (uint32, error) {
	var result = ""
	var err error = nil
	if d.redisCluster != nil {
		result, err = d.redisCluster.Get(context.TODO(), keyName).Result()
	} else {
		result, err = d.redis.Get(context.TODO(), keyName).Result()
	}
	if err != nil {
		return 0, err
	}
	value, err := strconv.Atoi(result)
	if err != nil {
		return 0, err
	}
	return uint32(value), nil
}

func (d *Dao) redisSet(keyName string, value uint32) error {
	var err error = nil
	if d.redisCluster != nil {
		err = d.redisCluster.Set(context.TODO(), keyName, value, 0).Err()
	} else {
		err = d.redis.Set(context.TODO(), keyName, value, 0).Err()
	}
	return err
}
