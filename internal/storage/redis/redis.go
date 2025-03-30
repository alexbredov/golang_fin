package redisclient

import (
	"antibf/internal/storage/storageData"
	"context"
	redisMock "github.com/alicebob/miniredis/v2"
	redis "github.com/redis/go-redis/v9"
	"strconv"
)

type RedisStorage struct {
	redisdb    *redis.Client
	mockServer *redisMock.Miniredis
}

func New() *RedisStorage {
	return &RedisStorage{}
}
func (red *RedisStorage) Init(ctx context.Context, logger storageData.Logger, config storageData.Config) error {
	red.redisdb = redis.NewClient(&redis.Options{
		Addr:     config.GetRedisAddress() + ":" + config.GetRedisPort(),
		Password: "",
		DB:       0,
	})
	_, err := red.redisdb.Ping(ctx).Result()
	if err != nil {
		logger.Error("RedisDB ping failed: " + err.Error())
		return err
	}
	red.redisdb.FlushDB(ctx)
	return nil
}
func (red *RedisStorage) InitAsMock(ctx context.Context, logger storageData.Logger) error {
	var err error
	red.mockServer, err = redisMock.Run()
	if err != nil {
		logger.Error("Redis mock failed:" + err.Error())
		return err
	}
	red.redisdb = redis.NewClient(&redis.Options{
		Addr:     red.mockServer.Addr(),
		Password: "",
		DB:       0,
	})
	_, err = red.redisdb.Ping(ctx).Result()
	if err != nil {
		logger.Error("Redis mock ping failed:" + err.Error())
		return err
	}
	red.redisdb.FlushDB(ctx)
	return nil
}
func (red *RedisStorage) Close(ctx context.Context, logger storageData.Logger) error {
	err := red.FlushStorage(ctx, logger)
	if err != nil {
		logger.Error("RedisDB flush error while closing: " + err.Error())
		return err
	}
	err = red.redisdb.Close()
	if err != nil {
		logger.Error("RedisDB error while closing: " + err.Error())
		return err
	}
	return nil
}
func (red *RedisStorage) IncreaseAndGetBucketValue(ctx context.Context, logger storageData.Logger, bucketName string) (int64, error) { //nolint:lll
	result, err := red.redisdb.Incr(ctx, bucketName).Result()
	if err != nil {
		logger.Error("RedisDB IncreaseAndGetBucketValue error: " + err.Error())
		return 0, err
	}
	return result, nil
}
func (red *RedisStorage) SetBucketValue(ctx context.Context, logger storageData.Logger, bucketName string, value int) error { //nolint:lll
	strValue := strconv.Itoa(value)
	err := red.redisdb.Set(ctx, bucketName, strValue, 0).Err()
	if err != nil {
		logger.Error("RedisDB SetBucketValue error: " + err.Error())
		return err
	}
	return nil
}
func (red *RedisStorage) FlushStorage(ctx context.Context, _ storageData.Logger) error {
	red.redisdb.FlushDB(ctx)
	return nil
}
