package cache

import (
	"context"
	"os"
	"time"

	"github.com/redis/go-redis/v9"
)

type Client struct{ RDB *redis.Client }

func New() *Client {
	a := os.Getenv("REDIS_ADDR")
	if a == "" {
		a = "localhost:6379"
	}
	// Username: os.Getenv("REDIS_USERNAME"),
	return &Client{redis.NewClient(&redis.Options{Addr: a, Password: os.Getenv("REDIS_PASSWORD"), DB: 0})}
}
func (c *Client) Ping(ctx context.Context) error { return c.RDB.Ping(ctx).Err() }
func (c *Client) Close() error                   { return c.RDB.Close() }
func (c *Client) SetReservation(ctx context.Context, orderID, status string) error {
	return c.RDB.Set(ctx, "inventory:reservation:"+orderID, status, 24*time.Hour).Err()
}
