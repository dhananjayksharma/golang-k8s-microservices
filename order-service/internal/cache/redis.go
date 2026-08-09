package cache

import (
	"context"
	"fmt"
	"os"
	"time"

	"github.com/redis/go-redis/v9"
)

type Client struct{ RDB *redis.Client }

func New() *Client {
	addr := os.Getenv("REDIS_ADDR")
	if addr == "" {
		addr = "localhost:6379"
	}
	// Username: os.Getenv("REDIS_USERNAME"),
	return &Client{redis.NewClient(&redis.Options{Addr: addr, Password: os.Getenv("REDIS_PASSWORD"), DB: 0})}
}
func (c *Client) Ping(ctx context.Context) error { return c.RDB.Ping(ctx).Err() }
func (c *Client) Close() error                   { return c.RDB.Close() }
func (c *Client) PutOrder(ctx context.Context, id string, raw []byte) error {
	return c.RDB.Set(ctx, "order:"+id, raw, 5*time.Minute).Err()
}
func (c *Client) GetOrder(ctx context.Context, id string) ([]byte, error) {
	v, err := c.RDB.Get(ctx, "order:"+id).Bytes()
	if err != nil {
		return nil, fmt.Errorf("redis get: %w", err)
	}
	return v, nil
}
