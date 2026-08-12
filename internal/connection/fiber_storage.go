package connection

import (
	"context"
	"time"

	"github.com/redis/go-redis/v9"
)

// FiberRedisStorage mengadaptasi *redis.Client (github.com/redis/go-redis/v9) yang sudah
// dipakai app lewat connection.RDB ke interface fiber.Storage. Dipakai bareng-bareng oleh
// semua middleware limiter (global /api dan /api/auth) supaya rate limit-nya konsisten
// lintas instance Cloud Run (state di Redis, bukan memori proses) — beda sama storage
// default fiber yang in-memory per-instance dan gak kelihatan pas di-debug.
type FiberRedisStorage struct {
	client *redis.Client
}

// NewFiberRedisStorage membungkus client redis milik app sebagai fiber.Storage.
// Client TIDAK akan ditutup oleh storage ini — umurnya dikelola connection.RDB.
func NewFiberRedisStorage(client *redis.Client) *FiberRedisStorage {
	return &FiberRedisStorage{client: client}
}

func (s *FiberRedisStorage) Get(key string) ([]byte, error) {
	return s.GetWithContext(context.Background(), key)
}

func (s *FiberRedisStorage) GetWithContext(ctx context.Context, key string) ([]byte, error) {
	if len(key) == 0 {
		return nil, nil
	}
	val, err := s.client.Get(ctx, key).Bytes()
	if err == redis.Nil {
		return nil, nil
	}
	return val, err
}

func (s *FiberRedisStorage) Set(key string, val []byte, exp time.Duration) error {
	return s.SetWithContext(context.Background(), key, val, exp)
}

func (s *FiberRedisStorage) SetWithContext(ctx context.Context, key string, val []byte, exp time.Duration) error {
	if len(key) == 0 || len(val) == 0 {
		return nil
	}
	return s.client.Set(ctx, key, val, exp).Err()
}

func (s *FiberRedisStorage) Delete(key string) error {
	return s.DeleteWithContext(context.Background(), key)
}

func (s *FiberRedisStorage) DeleteWithContext(ctx context.Context, key string) error {
	if len(key) == 0 {
		return nil
	}
	return s.client.Del(ctx, key).Err()
}

func (s *FiberRedisStorage) Reset() error {
	return s.ResetWithContext(context.Background())
}

func (s *FiberRedisStorage) ResetWithContext(ctx context.Context) error {
	return s.client.FlushDB(ctx).Err()
}

// Close no-op: client global (connection.RDB) dikelola di tempat lain, jangan ditutup di sini.
func (s *FiberRedisStorage) Close() error {
	return nil
}
