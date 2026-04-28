package lock

import (
	"context"
	"errors"
	"time"

	"github.com/redis/go-redis/v9"
)

var ErrLockAlreadyHeld = errors.New("lock already held")

type RedisLockManager struct {
	client *redis.Client
}

func NewRedisLockManager(client *redis.Client) *RedisLockManager {
	return &RedisLockManager{client: client}
}

func (m *RedisLockManager) Acquire(ctx context.Context, key string, ttl time.Duration) error {
	ok, err := m.client.SetNX(ctx, key, "1", ttl).Result()
	if err != nil {
		return err
	}
	if !ok {
		return ErrLockAlreadyHeld
	}
	return nil
}

func (m *RedisLockManager) Release(ctx context.Context, key string) error {
	return m.client.Del(ctx, key).Err()
}
